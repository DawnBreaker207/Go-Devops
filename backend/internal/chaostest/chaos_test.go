//go:build chaos

// Package chaostest is a real out-of-process resilience drill: it builds the
// actual cmd/server binary, runs it as a genuine OS subprocess against a
// scratch Postgres database, and talks to it only over HTTP — exactly like
// the two chaos scenarios from SPEC 5.3/5.4:
//
//   - a crash before startup leaves a batch_jobs row RUNNING; the next real
//     boot (through main.go, not through calling the Manager directly in a
//     unit test) must recover it.
//   - a hard kill (Process.Kill, the Go-runtime equivalent of kill -9 —
//     "kill -9" itself has no meaning on Windows) mid-hold must not corrupt
//     booking state; after a real restart, the held seat is exactly where it
//     was, and the ordinary sweep job still frees it once its hold expires.
//
// Driving the mock payment gateway (checkout, capture) is not reachable over
// HTTP by design — it exists only as in-process Go methods used by the
// service-level tests — so this drill covers the hold/crash/recover path,
// not a crash mid-payment. That gap is a known, documented limit of the
// current mock provider, not something faked here.
//
// Needs: the postgres container from docker-compose.yml running on
// localhost:5432 (docker compose up -d postgres), and the migrate CLI on
// PATH. Run with: go test -tags chaos ./internal/chaostest/...
package chaostest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	dbUser      = "postgres"
	dbPass      = "postgres"
	dbHost      = "localhost"
	dbPort      = "5432"
	chaosDBName = "cinema_chaos"
	sampleHallID = "20000000-0000-0000-0000-000000000001" // from migrations/seed/010002_seed_halls.sql
)

var binPath string

func TestMain(m *testing.M) {
	root, err := filepath.Abs("../..")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	bin := filepath.Join(os.TempDir(), "chaos-server")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "./cmd/server")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build cmd/server: %v\n%s\n", err, out)
		os.Exit(1)
	}
	binPath = bin

	code := m.Run()
	_ = os.Remove(bin)
	os.Exit(code)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	return root
}

func adminDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/postgres?sslmode=disable", dbUser, dbPass, dbHost, dbPort)
}

func chaosDSN(simple bool) string {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, chaosDBName)
	if simple {
		dsn += "&default_query_exec_mode=simple_protocol"
	}
	return dsn
}

// recreateDB drops and recreates the scratch database so every test starts clean.
func recreateDB(t *testing.T) {
	t.Helper()
	db, err := sql.Open("pgx", adminDSN())
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`DROP DATABASE IF EXISTS ` + chaosDBName + ` WITH (FORCE)`); err != nil {
		t.Fatalf("drop scratch db (is the postgres container from docker-compose.yml running?): %v", err)
	}
	if _, err := db.Exec(`CREATE DATABASE ` + chaosDBName); err != nil {
		t.Fatalf("create scratch db: %v", err)
	}
}

func applyMigrations(t *testing.T, root string) {
	t.Helper()
	// migrate's -path is parsed as a file:// URL; Windows backslashes break that parse.
	migPath := filepath.ToSlash(filepath.Join(root, "migrations", "schema"))
	cmd := exec.Command("migrate", "-path", migPath, "-database", chaosDSN(false), "up")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("migrate up (is the migrate CLI on PATH?): %v\n%s", err, out)
	}
}

// applySeeds runs migrations/seed/*.sql in order (3 sample halls + prices)
// so the drill has somewhere to schedule a showtime without going through
// the admin hall API first.
func applySeeds(t *testing.T, root string) {
	t.Helper()
	db, err := sql.Open("pgx", chaosDSN(true))
	if err != nil {
		t.Fatalf("open scratch db: %v", err)
	}
	defer db.Close()
	files, err := filepath.Glob(filepath.Join(root, "migrations", "seed", "*.sql"))
	if err != nil {
		t.Fatalf("glob seeds: %v", err)
	}
	sort.Strings(files)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read seed %s: %v", f, err)
		}
		if _, err := db.Exec(string(b)); err != nil {
			t.Fatalf("apply seed %s: %v", f, err)
		}
	}
}

func chaosDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("pgx", chaosDSN(false))
	if err != nil {
		t.Fatalf("open scratch db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find a free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// server is one real run of the built cmd/server binary as an OS subprocess.
type server struct {
	t       *testing.T
	cmd     *exec.Cmd
	baseURL string
	out     *strings.Builder
}

func startServer(t *testing.T, port int) *server {
	t.Helper()
	bin := binPath
	cmd := exec.Command(bin)
	cmd.Dir = repoRoot(t) // config.yaml is resolved relative to the working directory
	cmd.Env = append(os.Environ(),
		"APP_ENV=development",
		"SERVER_PORT="+strconv.Itoa(port),
		"DATABASE_HOST="+dbHost,
		"DATABASE_PORT="+dbPort,
		"DATABASE_USER="+dbUser,
		"DATABASE_PASSWORD="+dbPass,
		"DATABASE_NAME="+chaosDBName,
		"PAYMENT_PUBLIC_BASE_URL=http://localhost:"+strconv.Itoa(port),
		"REDIS_ADDR=",
		"MAIL_OUTBOX_DIR="+filepath.Join(t.TempDir(), "mail"),
		"STORAGE_LOCAL_DIR="+filepath.Join(t.TempDir(), "uploads"),
	)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start server subprocess: %v", err)
	}
	s := &server{t: t, cmd: cmd, baseURL: "http://localhost:" + strconv.Itoa(port), out: &out}
	s.waitHealthy(15 * time.Second)
	return s
}

func (s *server) waitHealthy(timeout time.Duration) {
	s.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if exited := s.cmd.ProcessState; exited != nil {
			s.t.Fatalf("server exited before becoming healthy:\n%s", s.out.String())
		}
		resp, err := http.Get(s.baseURL + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.t.Fatalf("server did not become healthy within %v:\n%s", timeout, s.out.String())
}

// kill hard-terminates the process (SIGKILL on unix, TerminateProcess on
// Windows) and waits for it to actually die — a real crash, not a graceful
// shutdown.
func (s *server) kill() {
	s.t.Helper()
	if err := s.cmd.Process.Kill(); err != nil {
		s.t.Fatalf("kill server: %v", err)
	}
	_ = s.cmd.Wait()
}

// --- tiny HTTP client for the API, blackbox style ---

func apiDo(t *testing.T, method, url, token string, body any) (int, map[string]any) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = strings.NewReader(string(b))
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	out := map[string]any{}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out
}

func dataOf(body map[string]any) map[string]any {
	if d, ok := body["data"].(map[string]any); ok {
		return d
	}
	return map[string]any{}
}

func mustLogin(t *testing.T, baseURL, email, password string) string {
	t.Helper()
	status, body := apiDo(t, http.MethodPost, baseURL+"/api/v1/auth/login", "", map[string]any{
		"email": email, "password": password,
	})
	if status != http.StatusOK {
		t.Fatalf("login %s: HTTP %d %v", email, status, body)
	}
	tok, _ := dataOf(body)["access_token"].(string)
	if tok == "" {
		t.Fatalf("login %s: no access_token in %v", email, body)
	}
	return tok
}

// TestChaos_OrphanRecoveredOnRealBoot: a crash leaves a batch_jobs row
// RUNNING with no process actually running it. Unlike the unit-level
// TestBatch_OrphanRunningRowDoesNotBlock (which drives internal/batch.Manager
// directly), this starts the real server binary and checks that cmd/server's
// own startup path (main.go) actually calls StopOrphans before anything else
// touches that row.
func TestChaos_OrphanRecoveredOnRealBoot(t *testing.T) {
	root := repoRoot(t)
	recreateDB(t)
	applyMigrations(t, root)

	db := chaosDB(t)
	if _, err := db.Exec(`INSERT INTO batch_jobs (id, job_name, status, started_at)
		VALUES (gen_random_uuid(), 'sweepExpiredHolds', 'running', NOW() - INTERVAL '1 hour')`); err != nil {
		t.Fatalf("seed orphan row: %v", err)
	}

	srv := startServer(t, freePort(t))
	defer srv.kill()

	deadline := time.Now().Add(10 * time.Second)
	var status string
	for time.Now().Before(deadline) {
		if err := db.QueryRow(`SELECT status FROM batch_jobs WHERE job_name = 'sweepExpiredHolds' ORDER BY started_at DESC LIMIT 1`).Scan(&status); err != nil {
			t.Fatalf("query orphan row: %v", err)
		}
		if status != "running" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if status == "running" {
		t.Fatalf("orphaned batch_jobs row still 'running' %v after a real boot, want it flipped to 'stopped'", 10*time.Second)
	}
	if status != "stopped" {
		t.Fatalf("orphan row status = %q, want 'stopped'", status)
	}
}

// TestChaos_HoldSurvivesHardKillThenSweepFreesIt: hold a real seat through
// the real HTTP API, hard-kill the server mid-hold (no graceful shutdown),
// restart it against the same database, and check the hold is exactly where
// it was — then move its expiry into the past and confirm the ordinary
// sweepExpiredHolds job (also run for real, over HTTP) still frees the seat.
func TestChaos_HoldSurvivesHardKillThenSweepFreesIt(t *testing.T) {
	root := repoRoot(t)
	recreateDB(t)
	applyMigrations(t, root)
	applySeeds(t, root)
	db := chaosDB(t)

	port := freePort(t)
	srv := startServer(t, port)

	admin := mustLogin(t, srv.baseURL, "admin@cinema.local", "admin123")

	status, body := apiDo(t, http.MethodPost, srv.baseURL+"/api/v1/movies", admin, map[string]any{
		"title": "Chaos Drill", "genre": "Drama", "duration": 100, "director": "Tester",
		"release_date": time.Now().Format("2006-01-02"), "status": "showing",
	})
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("create movie: HTTP %d %v", status, body)
	}
	movieID, _ := dataOf(body)["id"].(string)
	if movieID == "" {
		t.Fatalf("create movie: no id in %v", body)
	}

	startAt := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
	status, body = apiDo(t, http.MethodPost, srv.baseURL+"/api/v1/admin/showtimes", admin, map[string]any{
		"movie_id": movieID, "hall_id": sampleHallID, "start_at": startAt,
	})
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("create showtime: HTTP %d %v", status, body)
	}
	showID, _ := dataOf(body)["id"].(string)
	if showID == "" {
		t.Fatalf("create showtime: no id in %v", body)
	}

	status, body = apiDo(t, http.MethodPost, srv.baseURL+"/api/v1/auth/register", "", map[string]any{
		"email": "chaos.customer@example.com", "password": "secret123", "full_name": "Chaos Customer",
	})
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("register: HTTP %d %v", status, body)
	}
	customer := mustLogin(t, srv.baseURL, "chaos.customer@example.com", "secret123")

	status, body = apiDo(t, http.MethodGet, srv.baseURL+"/api/v1/shows/"+showID+"/seats", customer, nil)
	if status != http.StatusOK {
		t.Fatalf("seat map: HTTP %d %v", status, body)
	}
	seats, _ := dataOf(body)["seats"].([]any)
	var seatID string
	for _, s := range seats {
		seat, _ := s.(map[string]any)
		if gap, _ := seat["is_gap"].(bool); gap {
			continue
		}
		// HoldRequest.SeatIDs actually takes the per-showtime seat id
		// (showtime_seats.id), not the hall's seat id.
		seatID, _ = seat["showtime_seat_id"].(string)
		if seatID != "" {
			break
		}
	}
	if seatID == "" {
		t.Fatalf("seat map has no bookable seat: %v", body)
	}

	status, body = apiDo(t, http.MethodPost, srv.baseURL+"/api/v1/orders/hold", customer, map[string]any{
		"show_id": showID, "seat_ids": []string{seatID},
	})
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("hold: HTTP %d %v", status, body)
	}
	bookingID, _ := dataOf(body)["booking_id"].(string)
	if bookingID == "" {
		bookingID, _ = dataOf(body)["id"].(string)
	}
	if bookingID == "" {
		t.Fatalf("hold: no booking id in %v", body)
	}

	seatStatus := func() string {
		var st string
		if err := db.QueryRow(`SELECT status FROM showtime_seats WHERE id = $1`, seatID).Scan(&st); err != nil {
			t.Fatalf("query seat status: %v", err)
		}
		return st
	}
	bookingStatus := func() string {
		var st string
		if err := db.QueryRow(`SELECT status FROM bookings WHERE id = $1`, bookingID).Scan(&st); err != nil {
			t.Fatalf("query booking status: %v", err)
		}
		return st
	}

	if got := seatStatus(); got != "held" {
		t.Fatalf("seat status right after hold = %q, want held", got)
	}
	if got := bookingStatus(); got != "pending" {
		t.Fatalf("booking status right after hold = %q, want pending", got)
	}

	// The crash: no graceful shutdown, no draining requests.
	srv.kill()

	srv2 := startServer(t, port)
	defer srv2.kill()

	// I3/booking-state: the hold must survive a hard kill exactly as it was —
	// no half-applied write, no seat silently lost or double-freed.
	if got := seatStatus(); got != "held" {
		t.Fatalf("seat status after a hard kill + restart = %q, want still held", got)
	}
	if got := bookingStatus(); got != "pending" {
		t.Fatalf("booking status after a hard kill + restart = %q, want still pending", got)
	}

	if _, err := db.Exec(`UPDATE bookings SET expires_at = NOW() - INTERVAL '1 minute' WHERE id = $1`, bookingID); err != nil {
		t.Fatalf("force-expire the hold: %v", err)
	}
	if _, err := db.Exec(`UPDATE showtime_seats SET held_until = NOW() - INTERVAL '1 minute' WHERE id = $1`, seatID); err != nil {
		t.Fatalf("force-expire the seat hold: %v", err)
	}

	status, body = apiDo(t, http.MethodPost, srv2.baseURL+"/api/v1/admin/batch/jobs/sweepExpiredHolds/run", admin, nil)
	if status != http.StatusAccepted && status != http.StatusOK {
		t.Fatalf("trigger sweep: HTTP %d %v", status, body)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) && (seatStatus() != "available" || bookingStatus() != "expired") {
		time.Sleep(100 * time.Millisecond)
	}
	if got := seatStatus(); got != "available" {
		t.Fatalf("seat status after the post-restart sweep = %q, want available (freed)", got)
	}
	if got := bookingStatus(); got != "expired" {
		t.Fatalf("booking status after the post-restart sweep = %q, want expired", got)
	}
}
