// Command loadtest is a small, real HTTP load generator for the running API
// (SPEC "Hiệu năng (load)": ~300 virtual users reading the catalog, seat map
// and holding seats for N minutes, printing p50/p95/error rate against
// NFR-PERF-01 (read p95 < 300ms) and NFR-PERF-02 (hold/confirm p95 <= 1s).
//
// It talks to the API purely over HTTP, so it exercises a real running
// server (dev, staging, or a local `go run ./cmd/server`) — it does not
// import any internal package.
//
// Usage:
//
//	go run ./cmd/loadtest -base-url http://localhost:8080 -vus 300 -duration 60s
//
// If the target has no showtime scheduled for today, pass -bootstrap to have
// the tool create one itself (needs the default dev admin account and at
// least one hall from `make migrate-seed`).
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// errThrottled marks a 429: the rate limiter did its job. It is reported
// separately and never counted as a system failure or a latency sample —
// against a single-machine target, many virtual users on one IP will hit the
// per-IP public rate limit long before the backend itself is the bottleneck.
var errThrottled = errors.New("rate limited (429)")

func main() {
	baseURL := flag.String("base-url", "http://localhost:8080", "API base URL")
	vus := flag.Int("vus", 300, "number of concurrent virtual users")
	duration := flag.Duration("duration", 60*time.Second, "how long to run")
	holdShare := flag.Float64("hold-share", 0.1, "fraction of virtual users that also hold/cancel a seat each cycle")
	adminEmail := flag.String("admin-email", "admin@cinema.local", "admin account, used only with -bootstrap")
	adminPassword := flag.String("admin-password", "admin123", "admin account, used only with -bootstrap")
	bootstrap := flag.Bool("bootstrap", false, "create a movie+showtime for today if none is on sale")
	readP95Budget := flag.Duration("read-p95-budget", 300*time.Millisecond, "NFR-PERF-01 threshold")
	holdP95Budget := flag.Duration("hold-p95-budget", 1*time.Second, "NFR-PERF-02 threshold")
	flag.Parse()

	client := &http.Client{Timeout: 10 * time.Second}
	api := &apiClient{base: strings.TrimRight(*baseURL, "/"), http: client}

	showID, err := api.discoverShowtime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "discover a showtime on sale: %v\n", err)
		os.Exit(1)
	}
	if showID == "" {
		if !*bootstrap {
			fmt.Fprintln(os.Stderr, "no showtime on sale today; pass -bootstrap to create one, or seed data first (make migrate-seed)")
			os.Exit(1)
		}
		showID, err = api.bootstrapShowtime(*adminEmail, *adminPassword)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bootstrap a showtime: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("target showtime: %s\n", showID)

	holdUsers := int(float64(*vus) * *holdShare)
	if holdUsers == 0 && *holdShare > 0 {
		holdUsers = 1
	}
	poolSize := holdUsers
	if poolSize == 0 {
		poolSize = 1
	}
	pool, err := api.userPool(poolSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "prepare the load-test user pool: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%d virtual users (%d also holding seats), %v\n", *vus, holdUsers, *duration)

	var wg sync.WaitGroup
	results := make(chan vuResult, *vus)
	stop := time.Now().Add(*duration)
	for i := 0; i < *vus; i++ {
		wg.Add(1)
		doesHold := i < holdUsers
		token := ""
		if doesHold {
			token = pool[i%len(pool)]
		}
		go func(rng *rand.Rand) {
			defer wg.Done()
			results <- runVU(api, showID, token, doesHold, stop, rng)
		}(rand.New(rand.NewSource(time.Now().UnixNano() + int64(i))))
	}
	wg.Wait()
	close(results)

	var readLat, holdLat []time.Duration
	var readErrs, holdErrs, readThrottled, holdThrottled, readTotal, holdTotal int
	for r := range results {
		readLat = append(readLat, r.readLatencies...)
		holdLat = append(holdLat, r.holdLatencies...)
		readErrs += r.readErrors
		holdErrs += r.holdErrors
		readThrottled += r.readThrottled
		holdThrottled += r.holdThrottled
		readTotal += len(r.readLatencies) + r.readErrors + r.readThrottled
		holdTotal += len(r.holdLatencies) + r.holdErrors + r.holdThrottled
	}

	fmt.Println()
	readP50, readP95 := percentiles(readLat)
	holdP50, holdP95 := percentiles(holdLat)
	fmt.Printf("catalog/seat-map reads: n=%d errors=%d (%.2f%%) throttled=%d p50=%v p95=%v (over %d non-throttled samples)\n",
		readTotal, readErrs, errRate(readErrs, readTotal), readThrottled, readP50, readP95, len(readLat))
	if holdTotal > 0 {
		fmt.Printf("hold+cancel:            n=%d errors=%d (%.2f%%) throttled=%d p50=%v p95=%v (over %d non-throttled samples)\n",
			holdTotal, holdErrs, errRate(holdErrs, holdTotal), holdThrottled, holdP50, holdP95, len(holdLat))
	}
	if readThrottled+holdThrottled > 0 {
		fmt.Println("note: throttled requests hit the per-IP public rate limit, expected when many virtual users share one IP; raise rate_limit.public on the target for a truer capacity number")
	}

	pass := true
	if readTotal > 0 && readP95 > *readP95Budget {
		fmt.Printf("FAIL NFR-PERF-01: read p95 %v > budget %v\n", readP95, *readP95Budget)
		pass = false
	}
	if holdTotal > 0 && holdP95 > *holdP95Budget {
		fmt.Printf("FAIL NFR-PERF-02: hold p95 %v > budget %v\n", holdP95, *holdP95Budget)
		pass = false
	}
	if !pass {
		os.Exit(1)
	}
	fmt.Println("OK: within budget")
}

type vuResult struct {
	readLatencies, holdLatencies []time.Duration
	readErrors, holdErrors       int
	readThrottled, holdThrottled int
}

// vuThinkTime is a small pause between a virtual user's actions, like a real
// client pausing between page loads — without it, a handful of VUs on one
// machine self-inflict the public rate limit before the backend is the
// bottleneck at all.
const vuThinkTime = 150 * time.Millisecond

func runVU(api *apiClient, showID, token string, doesHold bool, stop time.Time, rng *rand.Rand) vuResult {
	var res vuResult
	for time.Now().Before(stop) {
		if doesHold && rng.Float64() < 0.3 {
			lat, err := api.timedHoldAndCancel(token, showID)
			switch {
			case errors.Is(err, errThrottled):
				res.holdThrottled++
			case err != nil:
				res.holdErrors++
			default:
				res.holdLatencies = append(res.holdLatencies, lat)
			}
			time.Sleep(vuThinkTime)
			continue
		}
		var lat time.Duration
		var err error
		if rng.Float64() < 0.5 {
			lat, err = api.timedGet("/api/v1/movies")
		} else {
			lat, err = api.timedGet("/api/v1/showtimes")
		}
		switch {
		case errors.Is(err, errThrottled):
			res.readThrottled++
		case err != nil:
			res.readErrors++
		default:
			res.readLatencies = append(res.readLatencies, lat)
		}
		time.Sleep(vuThinkTime)
	}
	return res
}

func percentiles(d []time.Duration) (p50, p95 time.Duration) {
	if len(d) == 0 {
		return 0, 0
	}
	sorted := append([]time.Duration(nil), d...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	at := func(p float64) time.Duration {
		idx := int(p * float64(len(sorted)))
		if idx >= len(sorted) {
			idx = len(sorted) - 1
		}
		return sorted[idx]
	}
	return at(0.50), at(0.95)
}

func errRate(errs, total int) float64 {
	if total == 0 {
		return 0
	}
	return 100 * float64(errs) / float64(total)
}

// --- a tiny HTTP client for the public/customer API surface ---

type apiClient struct {
	base string
	http *http.Client
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (a *apiClient) do(method, path, token string, body any) (int, envelope, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, envelope{}, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, a.base+path, reader)
	if err != nil {
		return 0, envelope{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := a.http.Do(req)
	if err != nil {
		return 0, envelope{}, err
	}
	defer resp.Body.Close()
	var env envelope
	_ = json.NewDecoder(resp.Body).Decode(&env)
	return resp.StatusCode, env, nil
}

func (a *apiClient) timedGet(path string) (time.Duration, error) {
	start := time.Now()
	status, env, err := a.do(http.MethodGet, path, "", nil)
	elapsed := time.Since(start)
	if err != nil {
		return 0, err
	}
	if status == http.StatusTooManyRequests {
		return 0, errThrottled
	}
	if status >= 400 {
		return 0, fmt.Errorf("%s: HTTP %d %s", path, status, env.Message)
	}
	return elapsed, nil
}

// timedHoldAndCancel holds one random bookable seat of the showtime and
// immediately cancels it, so the seat pool doesn't drain over the run; the
// measured latency is the hold call only (the one NFR-PERF-02 bounds).
func (a *apiClient) timedHoldAndCancel(token, showID string) (time.Duration, error) {
	seatMapStatus, env, err := a.do(http.MethodGet, "/api/v1/shows/"+showID+"/seats", token, nil)
	if err != nil {
		return 0, err
	}
	if seatMapStatus == http.StatusTooManyRequests {
		return 0, errThrottled
	}
	var seatMap struct {
		Seats []struct {
			ShowtimeSeatID string `json:"showtime_seat_id"`
			IsGap          bool   `json:"is_gap"`
			Status         string `json:"status"`
		} `json:"seats"`
	}
	if err := json.Unmarshal(env.Data, &seatMap); err != nil {
		return 0, err
	}
	var free []string
	for _, s := range seatMap.Seats {
		if !s.IsGap && s.Status == "available" {
			free = append(free, s.ShowtimeSeatID)
		}
	}
	if len(free) == 0 {
		return 0, fmt.Errorf("no available seat right now")
	}
	seatID := free[rand.Intn(len(free))]

	start := time.Now()
	status, env, err := a.do(http.MethodPost, "/api/v1/orders/hold", token, map[string]any{
		"show_id": showID, "seat_ids": []string{seatID},
	})
	elapsed := time.Since(start)
	if err != nil {
		return 0, err
	}
	if status == http.StatusConflict {
		// Another virtual user won the same seat first — expected under load, not a system failure.
		return 0, fmt.Errorf("seat contention (expected under load)")
	}
	if status == http.StatusTooManyRequests {
		return 0, errThrottled
	}
	if status >= 300 {
		return 0, fmt.Errorf("hold: HTTP %d %s", status, env.Message)
	}
	var held struct {
		BookingID string `json:"booking_id"`
		ID        string `json:"id"`
	}
	_ = json.Unmarshal(env.Data, &held)
	id := held.BookingID
	if id == "" {
		id = held.ID
	}
	if id != "" {
		_, _, _ = a.do(http.MethodPost, "/api/v1/orders/"+id+"/cancel", token, nil)
	}
	return elapsed, nil
}

func (a *apiClient) discoverShowtime() (string, error) {
	status, env, err := a.do(http.MethodGet, "/api/v1/showtimes", "", nil)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("HTTP %d %s", status, env.Message)
	}
	var items []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &items); err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", nil
	}
	return items[0].ID, nil
}

// bootstrapShowtime creates a movie and a showtime an hour from now on the
// first hall it finds, so the tool can run against a freshly seeded dev DB
// with no showtime scheduled yet.
func (a *apiClient) bootstrapShowtime(adminEmail, adminPassword string) (string, error) {
	token, err := a.login(adminEmail, adminPassword)
	if err != nil {
		return "", fmt.Errorf("admin login: %w", err)
	}
	status, env, err := a.do(http.MethodGet, "/api/v1/admin/halls", token, nil)
	if err != nil {
		return "", err
	}
	if status >= 400 {
		return "", fmt.Errorf("list halls: HTTP %d %s", status, env.Message)
	}
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(env.Data, &page); err != nil {
		return "", err
	}
	halls := page.Items
	if len(halls) == 0 {
		return "", fmt.Errorf("no halls found; run `make migrate-seed` first")
	}

	status, env, err = a.do(http.MethodPost, "/api/v1/movies", token, map[string]any{
		"title": "Load Test Movie", "genre": "Drama", "duration": 100, "director": "Load Tester",
		"release_date": time.Now().Format("2006-01-02"), "status": "showing",
	})
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("create movie: HTTP %d %s", status, env.Message)
	}
	var movie struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(env.Data, &movie)

	status, env, err = a.do(http.MethodPost, "/api/v1/admin/showtimes", token, map[string]any{
		"movie_id": movie.ID, "hall_id": halls[0].ID, "start_at": time.Now().Add(time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("create showtime: HTTP %d %s", status, env.Message)
	}
	var show struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(env.Data, &show)
	return show.ID, nil
}

func (a *apiClient) login(email, password string) (string, error) {
	status, env, err := a.do(http.MethodPost, "/api/v1/auth/login", "", map[string]any{
		"email": email, "password": password,
	})
	if err != nil {
		return "", err
	}
	if status >= 300 {
		return "", fmt.Errorf("HTTP %d %s", status, env.Message)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(env.Data, &tok); err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

// userPool registers (or logs into, if they already exist from a previous
// run) n customer accounts and returns their access tokens. Register/login
// share the auth rate limiter with real traffic, so this paces itself and
// retries on 429 instead of hammering it.
func (a *apiClient) userPool(n int) ([]string, error) {
	tokens := make([]string, 0, n)
	for i := 0; i < n; i++ {
		email := "loadtest.user" + strconv.Itoa(i) + "@example.com"
		const password = "secret123"
		if err := withRateLimitRetry(func() (int, error) {
			status, _, err := a.do(http.MethodPost, "/api/v1/auth/register", "", map[string]any{
				"email": email, "password": password, "full_name": "Load Test User",
			})
			return status, err
		}); err != nil {
			return nil, fmt.Errorf("register user %d: %w", i, err)
		}
		var token string
		if err := withRateLimitRetry(func() (int, error) {
			tok, err := a.login(email, password)
			token = tok
			if err != nil {
				return 0, err
			}
			return http.StatusOK, nil
		}); err != nil {
			return nil, fmt.Errorf("login user %d: %w", i, err)
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// withRateLimitRetry retries a 429 (or an error whose message names one)
// with a short backoff; any other outcome returns immediately.
func withRateLimitRetry(call func() (status int, err error)) error {
	for attempt := 0; attempt < 10; attempt++ {
		status, err := call()
		limited := status == http.StatusTooManyRequests ||
			(err != nil && strings.Contains(err.Error(), "429"))
		if !limited {
			return err
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("still rate-limited after retries")
}
