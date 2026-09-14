package service_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/jobs"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
)

// H1: a RUNNING row left by a crashed process is closed at startup and no
// longer blocks the job.
func TestBatch_OrphanRunningRowDoesNotBlock(t *testing.T) {
	e := newEnv(t)
	e.must(e.db.Exec(`INSERT INTO batch_jobs (id, job_name, triggered_by, status)
		VALUES (gen_random_uuid(), 'sweepExpiredHolds', 'cron', 'running')`).Error)
	manager := batch.NewManager(e.db, repository.NewBatchJobRepository(e.db))
	sweep := jobs.NewSweepExpiredHolds(e.svc)
	sweep.Schedule = "" // no cron tick may start a run between Start and the manual run below
	manager.Register(sweep)
	if err := manager.Run(e.ctx, "sweepExpiredHolds", models.TriggerManual); err == nil {
		t.Fatal("orphan RUNNING row did not block before startup (test setup)")
	}

	e.must(manager.Start(e.ctx))
	defer manager.Stop(context.Background())
	if n := e.count(`SELECT COUNT(*) FROM batch_jobs WHERE status = 'stopped' AND error_message LIKE 'interrupted%'`); n != 1 {
		t.Fatalf("orphan rows stopped = %d, want 1", n)
	}
	if err := manager.Run(e.ctx, "sweepExpiredHolds", models.TriggerManual); err != nil {
		t.Fatalf("job still blocked: %v", err)
	}
}

// M19: a manual run answers 202 at once and finishes in the background; a run
// already going is refused with 409.
func TestHTTP_ManualJobRunIsAsync(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	status, _, body := h.call(http.MethodPost, "/api/v1/admin/batch/jobs/closeDay/run", admin, nil)
	if status != http.StatusAccepted || dataMap(body)["run_id"] == nil || dataMap(body)["status"] != models.BatchRunning {
		t.Fatalf("run: HTTP %d %v", status, body)
	}
	runID := dataMap(body)["run_id"].(string)
	deadline := time.Now().Add(10 * time.Second)
	for e := h.env; ; {
		var st string
		e.must(e.db.Raw(`SELECT status FROM batch_jobs WHERE id = ?`, runID).Scan(&st).Error)
		if st == models.BatchSuccess {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("manual run still %q", st)
		}
		time.Sleep(50 * time.Millisecond)
	}

	h.must(h.db.Exec(`INSERT INTO batch_jobs (id, job_name, triggered_by, status)
		VALUES (gen_random_uuid(), 'closeDay', 'cron', 'running')`).Error)
	if status, _, _ := h.call(http.MethodPost, "/api/v1/admin/batch/jobs/closeDay/run", admin, nil); status != http.StatusConflict {
		t.Fatalf("run while running: HTTP %d, want 409", status)
	}
}

func loginAt(t *testing.T, base, email, password, forwardedFor string) int {
	t.Helper()
	body := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/auth/login", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", forwardedFor)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// H3: without trusted proxies a forged X-Forwarded-For does not reset the
// login lockout.
func TestHTTP_ForwardedForIgnoredWithoutTrustedProxy(t *testing.T) {
	h := newHTTPEnv(t)
	h.newUser(models.RoleCustomer, "xff@test.local", "secret123")
	for i := 1; i <= 5; i++ {
		if got := loginAt(t, h.srv.URL, "xff@test.local", "wrong-pass", fmt.Sprintf("203.0.113.%d", i)); got != http.StatusUnauthorized {
			t.Fatalf("attempt %d: HTTP %d", i, got)
		}
	}
	if got := loginAt(t, h.srv.URL, "xff@test.local", "secret123", "203.0.113.99"); got != http.StatusTooManyRequests {
		t.Fatalf("after 5 failures behind forged headers: HTTP %d, want 429", got)
	}
}

// H3: behind a configured proxy the forwarded client IP is used.
func TestHTTP_ForwardedForHonoredFromTrustedProxy(t *testing.T) {
	h := newHTTPEnv(t)
	h.trustedProxies = []string{"127.0.0.1/32", "::1/128"}
	srv := httptest.NewServer(h.buildEngine(h.db))
	defer srv.Close()
	h.newUser(models.RoleCustomer, "proxied@test.local", "secret123")
	for i := 1; i <= 5; i++ {
		loginAt(t, srv.URL, "proxied@test.local", "wrong-pass", "198.51.100.1")
	}
	if got := loginAt(t, srv.URL, "proxied@test.local", "secret123", "198.51.100.2"); got != http.StatusOK {
		t.Fatalf("another client behind the proxy: HTTP %d, want 200", got)
	}
	if got := loginAt(t, srv.URL, "proxied@test.local", "secret123", "198.51.100.1"); got != http.StatusTooManyRequests {
		t.Fatalf("the locked client behind the proxy: HTTP %d, want 429", got)
	}
}

// L7: oversized JSON bodies answer 413; defensive headers on every answer.
func TestHTTP_BodyLimitAndSecurityHeaders(t *testing.T) {
	h := newHTTPEnv(t)
	big := `{"email":"a@test.local","password":"` + strings.Repeat("x", 2<<20) + `"}`
	resp, err := http.Post(h.srv.URL+"/api/v1/auth/login", "application/json", bytes.NewReader([]byte(big)))
	h.must(err)
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("2MB body: HTTP %d, want 413", resp.StatusCode)
	}

	resp, err = http.Get(h.srv.URL + "/api/v1/healthz")
	h.must(err)
	resp.Body.Close()
	for header, want := range map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Cache-Control":          "no-store",
	} {
		if got := resp.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

// L4: locking an account or changing its role takes effect on tokens already
// issued, without waiting for them to expire.
func TestHTTP_LockedAccountTokenRejectedImmediately(t *testing.T) {
	h := newHTTPEnv(t)
	admin, _ := h.login(models.RoleAdmin)
	customer, customerID := h.login(models.RoleCustomer)
	if status, _, _ := h.call(http.MethodGet, "/api/v1/users/me", customer, nil); status != http.StatusOK {
		t.Fatalf("before: HTTP %d", status)
	}

	if status, _, body := h.call(http.MethodPut, "/api/v1/admin/users/"+customerID, admin, map[string]any{"active": false}); status != http.StatusOK {
		t.Fatalf("lock: HTTP %d %v", status, body)
	}
	if status, _, _ := h.call(http.MethodGet, "/api/v1/users/me", customer, nil); status != http.StatusForbidden {
		t.Fatalf("locked account token: HTTP %d, want 403", status)
	}

	if status, _, _ := h.call(http.MethodPut, "/api/v1/admin/users/"+customerID, admin, map[string]any{"active": true, "role": "staff"}); status != http.StatusOK {
		t.Fatalf("unlock and change role: HTTP %d", status)
	}
	if status, _, _ := h.call(http.MethodGet, "/api/v1/users/me", customer, nil); status != http.StatusUnauthorized {
		t.Fatalf("token with the old role: HTTP %d, want 401", status)
	}
}
