package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/config"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/handlers"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/jobs"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/router"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/sse"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/storage"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/ratelimit"
)

// httpEnv serves the real router with all middleware on top of env.
type httpEnv struct {
	*env
	engine *gin.Engine
	srv    *httptest.Server
	tokens *sse.TokenStore
	// Read by buildEngine; a nil publicLimiter means unlimited.
	trustedProxies []string
	publicLimiter  *ratelimit.Limiter
}

func newHTTPEnv(t *testing.T) *httpEnv {
	h := &httpEnv{env: newEnv(t)}
	h.engine = h.buildEngine(h.db)
	h.srv = httptest.NewServer(h.engine)
	t.Cleanup(h.srv.Close)
	return h
}

func (h *httpEnv) buildEngine(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{
		App:    config.AppConfig{Name: "test", Env: "test"},
		CORS:   config.CORSConfig{AllowedOrigins: []string{"http://localhost"}},
		Server: config.ServerConfig{MaxBodyBytes: 1 << 20, TrustedProxies: h.trustedProxies},
	}
	h.tokens = sse.NewTokenStore(sse.DefaultTokenTTL, nil)
	runs := repository.NewBatchJobRepository(db)
	mediaDir := h.t.TempDir()
	userRepo := repository.NewUserRepository(db)
	accounts := service.NewAccountStatusCache(userRepo, 30*time.Second)
	public := h.publicLimiter
	if public == nil {
		public = ratelimit.New(10000, 1000)
	}
	limits := router.Limiters{
		Auth:   ratelimit.New(10000, 1000),
		Hold:   ratelimit.New(10000, 1000),
		Public: public,
		Events: ratelimit.New(10000, 1000),
	}
	return router.New(cfg, db, h.jwt, accounts, limits, h.providers, router.Handlers{
		Health:   handlers.NewHealthHandler(db, "test"),
		Auth:     handlers.NewAuthHandler(h.auth),
		User:     handlers.NewUserHandler(service.NewUserService(db, userRepo, accounts.Invalidate)),
		Movie:    handlers.NewMovieHandler(h.movies),
		Batch:    handlers.NewBatchHandler(h.batchManager(db, runs), runs, db),
		Hall:     handlers.NewHallHandler(h.halls),
		Showtime: handlers.NewShowtimeHandler(h.showtimes),
		Booking:  handlers.NewBookingHandler(h.svc),
		SSE:      handlers.NewSSEHandler(h.hub, h.tokens, h.showtimes),
		Payment:  handlers.NewPaymentHandler(h.providers, h.svc, ""),
		Staff:    handlers.NewStaffHandler(h.reports, h.svc),
		Report:   handlers.NewReportHandler(h.reports),
		Media:    handlers.NewMediaHandler(service.NewMediaService(storage.NewLocal(mediaDir, "http://test"), 1<<20), mediaDir, 1<<20),
	})
}

func (h *httpEnv) batchManager(db *gorm.DB, runs repository.BatchJobRepository) *batch.Manager {
	m := batch.NewManager(db, runs)
	m.Register(jobs.NewCloseDay(h.reports, time.UTC))
	h.t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		m.Stop(ctx)
	})
	return m
}

// login creates an account of role and returns its access token and user id.
func (h *httpEnv) login(role string) (string, string) {
	h.t.Helper()
	email := fmt.Sprintf("%s-%s@test.local", role, uuid.NewString()[:8])
	u := h.newUser(role, email, "secret123")
	res, err := h.auth.Login(h.ctx, dto.LoginRequest{Email: email, Password: "secret123", ClientIP: "10.9.9.9"})
	h.must(err)
	return res.AccessToken, u.ID
}

func (h *httpEnv) call(method, path, token string, body any) (int, http.Header, map[string]any) {
	return h.callAt(h.srv.URL, method, path, token, body)
}

func (h *httpEnv) callAt(base, method, path, token string, body any) (int, http.Header, map[string]any) {
	h.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		h.must(err)
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, base+path, reader)
	h.must(err)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	h.must(err)
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, resp.Header, out
}

func dataMap(body map[string]any) map[string]any {
	m, _ := body["data"].(map[string]any)
	return m
}

// T15 / T38: every role only reaches its own endpoints.
func TestHTTP_RoleScopes(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	staff, _ := h.login(models.RoleStaff)
	customer, _ := h.login(models.RoleCustomer)

	cases := []struct {
		name, method, path, token string
		body                      any
		want                      int
	}{
		{"T15 staff reads admin report", http.MethodGet, "/api/v1/admin/reports/daily", staff, nil, http.StatusForbidden},
		{"T15 customer reads admin report", http.MethodGet, "/api/v1/admin/reports/daily", customer, nil, http.StatusForbidden},
		{"admin reads report", http.MethodGet, "/api/v1/admin/reports/daily", admin, nil, http.StatusOK},
		{"T38 customer checks a ticket in", http.MethodPost, "/api/v1/tickets/ANY/redeem", customer, map[string]string{"showtime_id": h.showID}, http.StatusForbidden},
		{"customer opens staff board", http.MethodGet, "/api/v1/staff/dashboard", customer, nil, http.StatusForbidden},
		{"staff creates accounts", http.MethodPost, "/api/v1/admin/users", staff, map[string]string{"email": "x@test.local", "password": "secret123", "full_name": "X", "role": "staff"}, http.StatusForbidden},
		{"staff changes prices", http.MethodPut, "/api/v1/halls/" + h.hallID + "/prices", customer, map[string]any{"prices": fullPrices()}, http.StatusForbidden},
		{"staff holds seats", http.MethodPost, "/api/v1/orders/hold", staff, map[string]any{"show_id": h.showID, "seat_ids": h.ids("A1")}, http.StatusForbidden},
		{"no token", http.MethodGet, "/api/v1/orders", "", nil, http.StatusUnauthorized},
	}
	for _, c := range cases {
		if status, _, body := h.call(c.method, c.path, c.token, c.body); status != c.want {
			t.Errorf("%s: HTTP %d (%v), want %d", c.name, status, body["message"], c.want)
		}
	}
}

// T41: another customer gets 403; the owner sees the order with its showtime.
func TestHTTP_OrderScopeAndShowtimeInfo(t *testing.T) {
	h := newHTTPEnv(t)
	ownerToken, ownerID := h.login(models.RoleCustomer)
	otherToken, _ := h.login(models.RoleCustomer)
	id := h.confirmed(ownerID, "A1")

	if status, _, _ := h.call(http.MethodGet, "/api/v1/orders/"+id, otherToken, nil); status != http.StatusForbidden {
		t.Fatalf("other customer: HTTP %d, want 403", status)
	}
	status, _, body := h.call(http.MethodGet, "/api/v1/orders/"+id, ownerToken, nil)
	if status != http.StatusOK {
		t.Fatalf("owner: HTTP %d", status)
	}
	show, _ := dataMap(body)["showtime"].(map[string]any)
	if show["movie_title"] != "Test Movie" || show["hall_name"] != "Hall 1" || show["ended"] != false {
		t.Fatalf("order showtime = %v", show)
	}
	status, _, body = h.call(http.MethodGet, "/api/v1/orders", ownerToken, nil)
	items, _ := dataMap(body)["items"].([]any)
	if status != http.StatusOK || len(items) != 1 || items[0].(map[string]any)["showtime"] == nil {
		t.Fatalf("order list: HTTP %d %v", status, items)
	}
}

// T14: over HTTP the lockout answers 429 with Retry-After.
func TestHTTP_LoginLockoutRetryAfter(t *testing.T) {
	h := newHTTPEnv(t)
	h.newUser(models.RoleCustomer, "lock@test.local", "secret123")
	for i := 1; i <= 5; i++ {
		if status, _, _ := h.call(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "lock@test.local", "password": "wrong-pass"}); status != http.StatusUnauthorized {
			t.Fatalf("attempt %d: HTTP %d", i, status)
		}
	}
	status, header, body := h.call(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "lock@test.local", "password": "secret123"})
	if status != http.StatusTooManyRequests {
		t.Fatalf("HTTP %d, want 429", status)
	}
	if secs, err := strconv.Atoi(header.Get("Retry-After")); err != nil || secs <= 0 {
		t.Fatalf("Retry-After = %q", header.Get("Retry-After"))
	}
	if details, _ := body["details"].(map[string]any); details["retry_after_seconds"] == nil {
		t.Fatalf("body = %v", body)
	}
}

func TestHTTP_ContractEndpoints(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, _ := h.login(models.RoleCustomer)

	if status, _, _ := h.call(http.MethodGet, "/api/v1/healthz", "", nil); status != http.StatusOK {
		t.Fatalf("HSL-01: HTTP %d", status)
	}

	var show models.Showtime
	h.must(h.db.First(&show, "id = ?", h.showID).Error)
	status, _, body := h.call(http.MethodGet, "/api/v1/showtimes?date="+show.StartAt.UTC().Format(dto.DateLayout), customer, nil)
	list, _ := body["data"].([]any)
	if status != http.StatusOK || !slices.ContainsFunc(list, func(v any) bool { return v.(map[string]any)["id"] == h.showID }) {
		t.Fatalf("SHOW-02: HTTP %d %v", status, list)
	}
	status, _, body = h.call(http.MethodGet, "/api/v1/showtimes?date=2031-01-01", customer, nil)
	if list, ok := body["data"].([]any); status != http.StatusOK || !ok || len(list) != 0 {
		t.Fatalf("T28 over HTTP: HTTP %d data=%#v", status, body["data"])
	}

	status, _, body = h.call(http.MethodGet, "/api/v1/halls/"+h.hallID+"/seats", customer, nil)
	if seats, _ := body["data"].([]any); status != http.StatusOK || len(seats) != 10 {
		t.Fatalf("HALL-02: HTTP %d, %d seats", status, len(seats))
	}
	if status, _, _ := h.call(http.MethodPut, "/api/v1/halls/"+h.hallID+"/prices", admin, map[string]any{"prices": fullPrices()}); status != http.StatusOK {
		t.Fatalf("HALL-04: HTTP %d", status)
	}

	staff := h.newUser(models.RoleStaff, "promote@test.local", "secret123")
	status, _, body = h.call(http.MethodPut, "/api/v1/admin/users/"+staff.ID, admin, map[string]string{"role": "admin"})
	if status != http.StatusOK || dataMap(body)["role"] != "admin" {
		t.Fatalf("USR-02: HTTP %d %v", status, body)
	}
}

// T48: 30 sequential reads per endpoint, p95 under 300 ms.
func TestHTTP_ReadLatencyP95(t *testing.T) {
	h := newHTTPEnv(t)
	token, _ := h.login(models.RoleCustomer)
	for _, path := range []string{
		"/api/v1/movies",
		"/api/v1/movies/" + h.movieID + "/showtimes",
		"/api/v1/shows/" + h.showID + "/seats",
	} {
		h.call(http.MethodGet, path, token, nil)
		latencies := make([]time.Duration, 0, 30)
		for i := 0; i < 30; i++ {
			start := time.Now()
			if status, _, _ := h.call(http.MethodGet, path, token, nil); status != http.StatusOK {
				t.Fatalf("%s: HTTP %d", path, status)
			}
			latencies = append(latencies, time.Since(start))
		}
		slices.Sort(latencies)
		p95 := latencies[28]
		t.Logf("%s p95 = %v", path, p95)
		if p95 > 300*time.Millisecond {
			t.Errorf("%s p95 = %v, want < 300ms", path, p95)
		}
	}
}

// T49: with the database gone, probes and API answer 503 fast.
func TestHTTP_DatabaseDownAnswers503Fast(t *testing.T) {
	h := newHTTPEnv(t)
	down, err := openDB(testDBName)
	h.must(err)
	sqlDB, err := down.DB()
	h.must(err)
	h.must(sqlDB.Close())

	srv := httptest.NewServer(h.buildEngine(down))
	defer srv.Close()
	for _, path := range []string{"/healthz", "/api/v1/healthz", "/api/v1/movies"} {
		start := time.Now()
		resp, err := http.Get(srv.URL + path)
		h.must(err)
		resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Errorf("%s: HTTP %d, want 503", path, resp.StatusCode)
		}
		if elapsed := time.Since(start); elapsed > 3*time.Second {
			t.Errorf("%s answered after %v", path, elapsed)
		}
	}
	if status, _, _ := h.call(http.MethodGet, "/api/v1/healthz", "", nil); status != http.StatusOK {
		t.Fatalf("healthy server: HTTP %d", status)
	}
}

// T50: shutdown with an order and a stream open is fast and the order can still be completed.
func TestHTTP_GracefulShutdownKeepsOrders(t *testing.T) {
	h := newHTTPEnv(t)
	token, userID := h.login(models.RoleCustomer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	h.must(err)
	server := &http.Server{Handler: h.engine, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}
	server.RegisterOnShutdown(h.hub.Close)
	go func() { _ = server.Serve(ln) }()
	base := "http://" + ln.Addr().String()

	status, _, body := h.callAt(base, http.MethodPost, "/api/v1/orders/hold", token,
		map[string]any{"show_id": h.showID, "seat_ids": h.ids("A1", "A2")})
	if status != http.StatusCreated {
		t.Fatalf("hold: HTTP %d %v", status, body)
	}
	bookingID := dataMap(body)["booking_id"].(string)

	status, _, body = h.callAt(base, http.MethodGet, "/api/v1/events/token?show_id="+h.showID, token, nil)
	if status != http.StatusOK {
		t.Fatalf("realtime token: HTTP %d", status)
	}
	stream, err := http.Get(base + dataMap(body)["stream_url"].(string))
	h.must(err)
	defer stream.Body.Close()
	if _, err := io.ReadAtLeast(stream.Body, make([]byte, 32), 5); err != nil {
		t.Fatalf("stream not open: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := time.Now()
	if err := server.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	t.Logf("shutdown took %v", time.Since(start))
	ended := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, stream.Body)
		close(ended)
	}()
	select {
	case <-ended:
	case <-time.After(3 * time.Second):
		t.Fatal("realtime stream still open after shutdown")
	}

	h.wantStatus(bookingID, models.BookingPending)
	h.wantSeat("A1", models.SeatStatusHeld)
	h.wantSeat("A2", models.SeatStatusHeld)
	if b := h.payAndNotify(userID, bookingID); b.Status != models.BookingConfirmed {
		t.Fatalf("order after restart: %s", b.Status)
	}
	h.checkInvariants()
}
