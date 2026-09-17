package service_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/batch"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/notify"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/payment/mock"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/repository"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/service"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/sse"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/jwt"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/ratelimit"
)

// Tests use a throwaway database on a real PostgreSQL and skip without one; TEST_DB_* override the defaults.
const testDBName = "cinema_booking_test"

const merchantURL = "http://merchant.test"

var testDB *gorm.DB

func TestMain(m *testing.M) {
	os.Exit(runWithDB(m))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func dsn(dbName string) string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Ho_Chi_Minh",
		envOr("TEST_DB_HOST", "localhost"), envOr("TEST_DB_PORT", "5432"),
		envOr("TEST_DB_USER", "postgres"), envOr("TEST_DB_PASSWORD", "postgres"), dbName)
}

func openDB(dbName string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn(dbName)), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(gormlogger.Silent),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(40)
	return db, sqlDB.Ping()
}

func runWithDB(m *testing.M) int {
	admin, err := openDB("postgres")
	if err != nil {
		fmt.Fprintf(os.Stderr, "postgres unavailable, DB integration tests skip: %v\n", err)
		return m.Run()
	}
	defer closeDB(admin)

	if err := admin.Exec("DROP DATABASE IF EXISTS " + testDBName + " WITH (FORCE)").Error; err != nil {
		fmt.Fprintf(os.Stderr, "drop test db: %v\n", err)
		return 1
	}
	if err := admin.Exec("CREATE DATABASE " + testDBName).Error; err != nil {
		fmt.Fprintf(os.Stderr, "create test db: %v\n", err)
		return 1
	}
	defer admin.Exec("DROP DATABASE IF EXISTS " + testDBName + " WITH (FORCE)")

	db, err := openDB(testDBName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open test db: %v\n", err)
		return 1
	}
	if err := applyMigrations(db, filepath.Join("..", "..", "migrations", "schema")); err != nil {
		closeDB(db)
		fmt.Fprintf(os.Stderr, "migrate test db: %v\n", err)
		return 1
	}
	testDB = db
	code := m.Run()
	closeDB(db)
	return code
}

func closeDB(db *gorm.DB) {
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
}

func applyMigrations(db *gorm.DB, dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return err
	}
	slices.Sort(files)
	for _, f := range files {
		sql, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if err := db.Exec(string(sql)).Error; err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(f), err)
		}
	}
	return nil
}

type fakePublisher struct {
	mu  sync.Mutex
	ids []string
}

func (p *fakePublisher) Publish(_ context.Context, queueName string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ids = append(p.ids, queueName+":"+string(body))
	return nil
}

func (p *fakePublisher) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.ids)
}

// env seeds 1 movie, "Hall 1" of 2x5 seats (row A standard with gap A5, row B vip) and a showtime in 3 hours.
type env struct {
	t         *testing.T
	ctx       context.Context
	db        *gorm.DB
	svc       service.BookingService
	emails    service.TicketEmailService
	showtimes service.ShowtimeService
	providers *payment.Registry
	mockP     *mock.Provider
	gw        *mock.Gateway
	mailer    *notify.MockMailer
	hub       *sse.Hub
	pub       *fakePublisher
	movieID   string
	hallID    string
	showID    string
	seat      map[string]string
	users     []string
	emailOf   map[string]string

	auth     service.AuthService
	halls    service.HallService
	jwt      *jwt.Manager
	accounts service.UserService
	reports  service.ReportService
	movies   service.MovieService
	now      time.Time // clock of the login lockout
}

const (
	priceStandard = 70000
	priceVIP      = 100000
)

func newEnv(t *testing.T) *env {
	t.Helper()
	if testDB == nil {
		t.Skip("postgres unavailable")
	}
	ctx := context.Background()
	e := &env{t: t, ctx: ctx, db: testDB, seat: map[string]string{}, emailOf: map[string]string{}}

	// DELETE, not TRUNCATE: far faster on tiny tables (no new relfilenodes and fsync).
	// audit_logs is append-only in production (trg_audit_logs_immutable); the
	// trigger is toggled off only for this between-tests reset.
	e.must(testDB.Exec(`ALTER TABLE audit_logs DISABLE TRIGGER trg_audit_logs_immutable;
		DELETE FROM audit_logs;
		ALTER TABLE audit_logs ENABLE TRIGGER trg_audit_logs_immutable;
		DELETE FROM ledger_entries; DELETE FROM point_transactions; DELETE FROM reward_redemptions;
		DELETE FROM waitlist_entries; DELETE FROM pricing_rules;
		DELETE FROM voucher_redemptions; DELETE FROM booking_combos;
		DELETE FROM batch_jobs; DELETE FROM daily_aggregates;
		DELETE FROM tickets; DELETE FROM booking_seats;
		DELETE FROM payments; DELETE FROM bookings; DELETE FROM showtime_seats;
		DELETE FROM showtimes; DELETE FROM hall_prices; DELETE FROM seats; DELETE FROM halls;
		DELETE FROM movies;
		DELETE FROM rewards; DELETE FROM combo_branch_stock; DELETE FROM combos; DELETE FROM vouchers;
		DELETE FROM user_memberships; DELETE FROM membership_tiers; DELETE FROM user_points;
		DELETE FROM articles; DELETE FROM admin_permissions;
		DELETE FROM refresh_tokens; DELETE FROM password_reset_tokens; DELETE FROM users;
		DELETE FROM branches WHERE name <> 'Chi nhanh chinh';`).Error)

	for i := 0; i < 8; i++ {
		u := &models.User{Email: fmt.Sprintf("user%d@test.local", i), Password: "x", FullName: fmt.Sprintf("User %d", i), Role: models.RoleCustomer}
		e.must(testDB.Create(u).Error)
		e.users = append(e.users, u.ID)
		e.emailOf[u.ID] = u.Email
	}

	movie := &models.Movie{Title: "Test Movie", Genre: "Drama", Duration: 100, Director: "Tester",
		ReleaseDate: time.Now(), Status: models.MovieStatusShowing}
	e.must(testDB.Create(movie).Error)
	e.movieID = movie.ID

	hallRepo := repository.NewHallRepository(testDB)
	e.halls = service.NewHallService(testDB, hallRepo, repository.NewBranchRepository(testDB), nil)
	hall, err := e.halls.Create(ctx, dto.HallRequest{
		Name: "Hall 1", Rows: 2, SeatsPerRow: 5,
		SeatTypes: map[string][]string{"vip": {"2"}},
		Gaps:      []string{"A5"},
		Prices:    fullPrices(),
	})
	e.must(err)
	e.hallID = hall.ID

	e.showtimes = service.NewShowtimeService(testDB, repository.NewShowtimeRepository(testDB), hallRepo,
		repository.NewMovieRepository(testDB), 20, time.UTC, nil, 0)
	e.showID = e.newShowtime(3 * time.Hour)
	e.seat = e.seatsOf(e.showID)

	e.providers = payment.NewRegistry()
	e.mockP = e.addProvider("mock")
	e.gw = e.mockP.Gateway()
	e.must(e.providers.SetDefault("mock"))

	repo := repository.NewBookingRepository(testDB)
	e.hub = sse.NewHub()
	e.pub = &fakePublisher{}
	e.svc = service.NewBookingService(service.BookingOptions{
		DB:            testDB,
		Repo:          repo,
		Payments:      repository.NewPaymentRepository(testDB),
		Vouchers:      repository.NewVoucherRepository(testDB),
		Memberships:   repository.NewMembershipRepository(testDB),
		Combos:        repository.NewComboRepository(testDB),
		Loyalty:       service.NewLoyaltyService(testDB, repository.NewLoyaltyRepository(testDB), repository.NewVoucherRepository(testDB), repository.NewLedgerRepository(testDB)),
		Ledger:        repository.NewLedgerRepository(testDB),
		Waitlist:      repository.NewWaitlistRepository(testDB),
		PricingRules:  repository.NewPricingRuleRepository(testDB),
		Branches:      repository.NewBranchRepository(testDB),
		Providers:     e.providers,
		PublicBaseURL: merchantURL,
		HoldTTL:       10 * time.Minute,
		MaxSeats:      4,
		Publisher:     e.pub,
		Hub:           e.hub,
	})
	e.mailer = notify.NewMockMailer("")
	e.emails = service.NewTicketEmailService(testDB, repo, e.mailer, time.UTC)

	e.now = time.Now()
	userRepo := repository.NewUserRepository(testDB)
	e.jwt = jwt.NewManager("test-access-secret", "test-refresh-secret", "test", 15*time.Minute, time.Hour)
	guard := ratelimit.NewFailureLimiter(5, 5*time.Minute, func() time.Time { return e.now })
	e.auth = service.NewAuthService(testDB, userRepo, repository.NewRefreshTokenRepository(testDB), e.jwt, guard,
		repository.NewPasswordResetTokenRepository(testDB), e.mailer, "http://test.local/reset-password", 30*time.Minute, 0)
	e.accounts = service.NewUserService(testDB, userRepo)
	e.reports = service.NewReportService(repository.NewReportRepository(testDB), repository.NewShowtimeRepository(testDB),
		repository.NewPaymentRepository(testDB), repository.NewBatchJobRepository(testDB), repository.NewBookingRepository(testDB), time.UTC)
	e.movies = service.NewMovieService(testDB, repository.NewMovieRepository(testDB), nil, 0)
	return e
}

func fullPrices() map[string]int64 {
	return map[string]int64{"standard": priceStandard, "vip": priceVIP, "couple": 160000, "recliner": 130000}
}

func (e *env) newUser(role, email, password string) *models.User {
	e.t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	e.must(err)
	u := &models.User{Email: email, Password: string(hash), FullName: "Test " + role, Role: role, Active: true}
	e.must(e.db.Create(u).Error)
	return u
}

func (e *env) runJob(job *batch.Job) (processed, skipped int) {
	e.t.Helper()
	e.must(job.Run(e.ctx, batch.RunOptions{DB: e.db, Progress: func(p, s int) error {
		processed, skipped = p, s
		return nil
	}}))
	return processed, skipped
}

func (e *env) must(err error) {
	e.t.Helper()
	if err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) addProvider(name string) *mock.Provider {
	e.t.Helper()
	p := mock.New(mock.Options{Name: name, Secret: "test-secret-" + name, PublicBaseURL: merchantURL})
	e.must(e.providers.Register(p))
	return p
}

func (e *env) newShowtime(in time.Duration) string {
	e.t.Helper()
	st, err := e.showtimes.Create(e.ctx, dto.ShowtimeRequest{MovieID: e.movieID, HallID: e.hallID, StartAt: time.Now().Add(in)})
	e.must(err)
	return st.ID
}

// moveShowStart makes a showtime start d from the DB clock (negative: already started), keeping its length.
func (e *env) moveShowStart(showID string, d time.Duration) {
	e.t.Helper()
	e.must(e.db.Exec(`UPDATE showtimes SET start_at = NOW() + make_interval(secs => ?),
		end_at = NOW() + make_interval(secs => ?) + (end_at - start_at) WHERE id = ?`,
		d.Seconds(), d.Seconds(), showID).Error)
}

func (e *env) seatsOf(showID string) map[string]string {
	e.t.Helper()
	var rows []struct {
		ID        string
		RowLabel  string
		ColNumber int
	}
	e.must(e.db.Raw(`SELECT ss.id, s.row_label, s.col_number FROM showtime_seats ss
		JOIN seats s ON s.id = ss.seat_id WHERE ss.showtime_id = ?`, showID).Scan(&rows).Error)
	out := make(map[string]string, len(rows))
	for _, r := range rows {
		out[dto.SeatLabel(r.RowLabel, r.ColNumber)] = r.ID
	}
	return out
}

func (e *env) ids(labels ...string) []string {
	out := make([]string, 0, len(labels))
	for _, l := range labels {
		id, ok := e.seat[l]
		if !ok {
			e.t.Fatalf("unknown seat %s", l)
		}
		out = append(out, id)
	}
	return out
}

func (e *env) hold(user string, labels ...string) (*dto.HoldResponse, error) {
	return e.svc.Hold(e.ctx, user, dto.HoldRequest{ShowID: e.showID, SeatIDs: e.ids(labels...)})
}

func (e *env) mustHold(user string, labels ...string) *dto.HoldResponse {
	e.t.Helper()
	res, err := e.hold(user, labels...)
	e.must(err)
	return res
}

func (e *env) pay(user, bookingID string) string {
	e.t.Helper()
	res, err := e.svc.Pay(e.ctx, user, bookingID, dto.PayRequest{Provider: "mock"})
	e.must(err)
	return res.TxnRef
}

func (e *env) capture(ref string, opts ...mock.CaptureOptions) {
	e.t.Helper()
	var o mock.CaptureOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	_, err := e.gw.Capture(ref, o)
	e.must(err)
}

func (e *env) notificationRequest(p *mock.Provider, ref string) *http.Request {
	e.t.Helper()
	req, err := p.Gateway().NotificationRequest(e.ctx, ref)
	e.must(err)
	return req
}

func (e *env) notify(p *mock.Provider, req *http.Request) payment.AckStatus {
	return e.svc.HandleNotification(e.ctx, p, req)
}

func (e *env) ipnVia(p *mock.Provider, ref string) payment.AckStatus {
	return e.notify(p, e.notificationRequest(p, ref))
}

func (e *env) ipn(ref string) payment.AckStatus { return e.ipnVia(e.mockP, ref) }

func (e *env) payAndNotify(user, bookingID string) models.Booking {
	e.t.Helper()
	ref := e.pay(user, bookingID)
	e.capture(ref)
	if ack := e.ipn(ref); ack != payment.AckProcessed {
		e.t.Fatalf("ipn ack = %s", ack)
	}
	return e.booking(bookingID)
}

func (e *env) confirmed(user string, labels ...string) string {
	e.t.Helper()
	h := e.mustHold(user, labels...)
	if b := e.payAndNotify(user, h.BookingID); b.Status != models.BookingConfirmed {
		e.t.Fatalf("booking %s: status %s", h.BookingID, b.Status)
	}
	return h.BookingID
}

func forgedIPN(ref string, amount int64) *http.Request {
	body := fmt.Sprintf(`{"txn_ref":%q,"amount":%d,"status":"paid","gateway_txn_id":"FAKE","signature":%q}`,
		ref, amount, strings.Repeat("0", 64))
	return httptest.NewRequest(http.MethodPost, "/api/v1/payments/mock/ipn", strings.NewReader(body))
}

// shiftHold sets a booking's expiry and its seats' held_until to NOW()+secs.
func (e *env) shiftHold(bookingID string, secs float64) {
	e.t.Helper()
	// Lock in the service's order so concurrent tests do not deadlock on this helper.
	e.must(e.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`SELECT 1 FROM bookings WHERE id = ? FOR UPDATE`, bookingID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`SELECT 1 FROM showtime_seats WHERE id IN (
			SELECT showtime_seat_id FROM booking_seats WHERE booking_id = ?) ORDER BY id FOR UPDATE`, bookingID).Error; err != nil {
			return err
		}
		return tx.Exec(`WITH b AS (
				UPDATE bookings SET expires_at = NOW() + make_interval(secs => ?) WHERE id = ? RETURNING id, expires_at)
			UPDATE showtime_seats ss SET held_until = b.expires_at
			FROM booking_seats bs, b
			WHERE bs.booking_id = b.id AND ss.id = bs.showtime_seat_id AND ss.status = 'held'`, secs, bookingID).Error
	}))
}

func (e *env) booking(id string) models.Booking {
	e.t.Helper()
	var b models.Booking
	e.must(e.db.Unscoped().First(&b, "id = ?", id).Error)
	return b
}

func (e *env) payment(ref string) models.Payment {
	e.t.Helper()
	var p models.Payment
	e.must(e.db.First(&p, "txn_ref = ?", ref).Error)
	return p
}

func (e *env) seatState(label string) models.ShowtimeSeat {
	e.t.Helper()
	var s models.ShowtimeSeat
	e.must(e.db.First(&s, "id = ?", e.seat[label]).Error)
	return s
}

func (e *env) count(sql string, args ...any) int64 {
	e.t.Helper()
	var n int64
	e.must(e.db.Raw(sql, args...).Scan(&n).Error)
	return n
}

func (e *env) wantStatus(bookingID, status string) models.Booking {
	e.t.Helper()
	b := e.booking(bookingID)
	if b.Status != status {
		reason := ""
		if b.StatusReason != nil {
			reason = *b.StatusReason
		}
		e.t.Fatalf("booking %s: status %s (%s), want %s", bookingID, b.Status, reason, status)
	}
	return b
}

func (e *env) wantReason(b models.Booking, reason string) {
	e.t.Helper()
	if b.StatusReason == nil || *b.StatusReason != reason {
		e.t.Fatalf("booking %s: reason %v, want %s", b.ID, b.StatusReason, reason)
	}
}

func (e *env) wantSeat(label, status string) {
	e.t.Helper()
	if got := e.seatState(label).Status; got != status {
		e.t.Fatalf("seat %s: status %s, want %s", label, got, status)
	}
}

// checkInvariants verifies the SPEC 5.3 invariants and the money trail by SQL.
func (e *env) checkInvariants() {
	e.t.Helper()
	checks := []struct{ name, sql string }{
		{"I1 every SOLD seat has exactly one ticket of a confirmed booking", `SELECT COUNT(*) FROM showtime_seats ss
			WHERE ss.status = 'sold' AND (SELECT COUNT(*) FROM tickets t JOIN bookings b ON b.id = t.booking_id
				WHERE t.showtime_seat_id = ss.id AND b.status = 'confirmed') <> 1`},
		{"I1 every ticket sits on a SOLD seat", `SELECT COUNT(*) FROM tickets t
			JOIN showtime_seats ss ON ss.id = t.showtime_seat_id WHERE ss.status <> 'sold'`},
		{"I2 tickets match booking seats and total", `SELECT COUNT(*) FROM bookings b WHERE b.status = 'confirmed' AND (
			b.total_amount <> COALESCE((SELECT SUM(price) FROM tickets WHERE booking_id = b.id), 0)
			OR (SELECT COUNT(*) FROM tickets WHERE booking_id = b.id) <> (SELECT COUNT(*) FROM booking_seats WHERE booking_id = b.id))`},
		{"I3 at most one PENDING per user and show", `SELECT COUNT(*) FROM (SELECT 1 FROM bookings WHERE status = 'pending'
			GROUP BY user_id, showtime_id HAVING COUNT(*) > 1) x`},
		{"I5 no ticket on a non-confirmed booking", `SELECT COUNT(*) FROM tickets t
			JOIN bookings b ON b.id = t.booking_id WHERE b.status <> 'confirmed'`},
		{"confirmed bookings keep exactly their paid amount", `SELECT COUNT(*) FROM bookings b
			LEFT JOIN payments p ON p.id = b.payment_id
			WHERE b.status = 'confirmed' AND (p.id IS NULL OR p.status <> 'paid' OR p.paid_amount <> b.total_amount)`},
		{"refunded bookings send their money back", `SELECT COUNT(*) FROM bookings b
			LEFT JOIN payments p ON p.id = b.payment_id
			WHERE b.status = 'refunded' AND (b.paid_at IS NULL OR p.id IS NULL OR p.status NOT IN ('refund_pending', 'refunded'))`},
		{"no booking keeps money from two attempts", `SELECT COUNT(*) FROM (SELECT 1 FROM payments WHERE status = 'paid'
			GROUP BY booking_id HAVING COUNT(*) > 1) x`},
		{"I6 ticket codes are unique", `SELECT COUNT(*) FROM (SELECT 1 FROM tickets GROUP BY code HAVING COUNT(*) > 1) x`},
		{"I8 no redeemed ticket on a non-confirmed booking", `SELECT COUNT(*) FROM tickets t
			JOIN bookings b ON b.id = t.booking_id WHERE t.status = 'redeemed' AND b.status <> 'confirmed'`},
	}
	for _, c := range checks {
		if n := e.count(c.sql); n != 0 {
			e.t.Errorf("invariant broken: %s (%d rows)", c.name, n)
		}
	}
}

func httpStatus(err error) int {
	if err == nil {
		return 0
	}
	return apperrors.From(err).Status
}

func isAppErr(err error, target *apperrors.AppError) bool {
	if err == nil {
		return false
	}
	got := apperrors.From(err)
	return got.Code == target.Code && got.Message == target.Message
}
