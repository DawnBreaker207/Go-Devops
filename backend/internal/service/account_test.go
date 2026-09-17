package service_test

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/dto"
	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/models"
	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
)

func login(email, password, ip string) dto.LoginRequest {
	return dto.LoginRequest{Email: email, Password: password, ClientIP: ip}
}

func setActive(v bool) dto.UpdateUserRequest { return dto.UpdateUserRequest{Active: &v} }

func setRole(role string) dto.UpdateUserRequest { return dto.UpdateUserRequest{Role: &role} }

// T27: registering an existing email is refused, nothing created.
func TestRegister_DuplicateEmail(t *testing.T) {
	e := newEnv(t)
	u, err := e.auth.Register(e.ctx, dto.RegisterRequest{Email: "New@Test.local", Password: "secret123", FullName: "New User"})
	e.must(err)
	if u.Email != "new@test.local" || u.Role != models.RoleCustomer || !u.Active {
		t.Fatalf("registered = %+v", u)
	}
	if _, err := e.auth.Register(e.ctx, dto.RegisterRequest{Email: "new@test.local", Password: "secret123", FullName: "Again"}); httpStatus(err) != http.StatusConflict {
		t.Fatalf("duplicate: err = %v, want 409", err)
	}
	if n := e.count(`SELECT COUNT(*) FROM users WHERE email = 'new@test.local'`); n != 1 {
		t.Fatalf("accounts = %d, want 1", n)
	}
}

// T13 / E-C4: a refresh token works once; replaying a used one revokes the whole family.
func TestRefreshToken_RotationAndReplay(t *testing.T) {
	e := newEnv(t)
	e.newUser(models.RoleCustomer, "c@test.local", "secret123")

	res, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1"))
	e.must(err)
	first := res.RefreshToken
	second, err := e.auth.Refresh(e.ctx, first)
	e.must(err)
	if second.RefreshToken == first {
		t.Fatal("refresh did not rotate the token")
	}
	third, err := e.auth.Refresh(e.ctx, second.RefreshToken)
	e.must(err)

	if _, err := e.auth.Refresh(e.ctx, first); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("replay: err = %v, want 401", err)
	}
	if _, err := e.auth.Refresh(e.ctx, third.RefreshToken); httpStatus(err) != http.StatusUnauthorized {
		t.Fatalf("newest token after replay: err = %v, want 401", err)
	}

	again, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1"))
	e.must(err)
	if _, err := e.auth.Refresh(e.ctx, again.RefreshToken); err != nil {
		t.Fatalf("new login family: %v", err)
	}
}

// T14: 5 wrong passwords lock that email+IP with 429; another IP is unaffected; the lockout ends.
func TestLogin_LockoutAfterFiveFailures(t *testing.T) {
	e := newEnv(t)
	e.newUser(models.RoleCustomer, "c@test.local", "secret123")

	for i := 1; i <= 5; i++ {
		if _, err := e.auth.Login(e.ctx, login("c@test.local", "wrong-pass", "10.0.0.1")); httpStatus(err) != http.StatusUnauthorized {
			t.Fatalf("attempt %d: err = %v, want 401", i, err)
		}
	}
	_, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1"))
	if httpStatus(err) != http.StatusTooManyRequests {
		t.Fatalf("after 5 failures: err = %v, want 429", err)
	}
	if wait := apperrors.From(err).Details["retry_after_seconds"]; wait != "300" {
		t.Fatalf("retry_after_seconds = %q, want 300", wait)
	}
	if _, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.2")); err != nil {
		t.Fatalf("other IP: %v", err)
	}
	e.now = e.now.Add(5*time.Minute + time.Second)
	if _, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1")); err != nil {
		t.Fatalf("after lockout: %v", err)
	}
}

// E-U1: admin creates staff, duplicates are refused, staff can log in.
func TestAccounts_CreateStaffAndList(t *testing.T) {
	e := newEnv(t)
	staff, err := e.accounts.Create(e.ctx, dto.CreateUserRequest{
		Email: "Staff1@Cinema.local", Password: "secret123", FullName: "Staff One", Role: models.RoleStaff,
	})
	e.must(err)
	if staff.Email != "staff1@cinema.local" || staff.Role != models.RoleStaff || !staff.Active {
		t.Fatalf("staff = %+v", staff)
	}
	if _, err := e.accounts.Create(e.ctx, dto.CreateUserRequest{
		Email: "staff1@cinema.local", Password: "secret123", FullName: "Dup", Role: models.RoleStaff,
	}); !isAppErr(err, apperrors.ErrEmailAlreadyExists) {
		t.Fatalf("duplicate: err = %v", err)
	}
	res, err := e.auth.Login(e.ctx, login("staff1@cinema.local", "secret123", "10.0.0.9"))
	e.must(err)
	if res.User.Role != models.RoleStaff {
		t.Fatalf("login role = %s", res.User.Role)
	}
	active := true
	items, total, err := e.accounts.List(e.ctx, dto.UserListQuery{
		PageQuery: dto.PageQuery{Page: 1, PageSize: 10}, Role: models.RoleStaff, Active: &active,
	})
	e.must(err)
	if total != 1 || len(items) != 1 || items[0].ID != staff.ID {
		t.Fatalf("staff list = %d %+v", total, items)
	}
}

// T16 / E-U2: an admin can not lock itself; two admins locking each other leave one active.
func TestAdmins_CanNotLockThemselvesOrTheLastAdmin(t *testing.T) {
	e := newEnv(t)
	a := e.newUser(models.RoleAdmin, "a@test.local", "secret123")
	b := e.newUser(models.RoleAdmin, "b@test.local", "secret123")

	if _, err := e.accounts.Update(e.ctx, a.ID, a.ID, setActive(false)); !isAppErr(err, apperrors.ErrCannotLockSelf) {
		t.Fatalf("lock self: err = %v", err)
	}

	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, pair := range [][2]string{{a.ID, b.ID}, {b.ID, a.ID}} {
		wg.Add(1)
		go func(i int, actor, target string) {
			defer wg.Done()
			_, errs[i] = e.accounts.Update(e.ctx, actor, target, setActive(false))
		}(i, pair[0], pair[1])
	}
	wg.Wait()
	ok, last := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case isAppErr(err, apperrors.ErrLastAdmin):
			last++
		default:
			t.Fatalf("unexpected: %v", err)
		}
	}
	if ok != 1 || last != 1 {
		t.Fatalf("ok=%d last-admin=%d, want 1/1", ok, last)
	}
	if n := e.count(`SELECT COUNT(*) FROM users WHERE role = 'admin' AND active`); n != 1 {
		t.Fatalf("active admins = %d, want 1", n)
	}
}

// E-U2: nobody changes their own role, the last admin stays, the new role applies at login.
func TestAccounts_RoleChanges(t *testing.T) {
	e := newEnv(t)
	root := e.newUser(models.RoleAdmin, "root@test.local", "secret123")
	other := e.newUser(models.RoleStaff, "other@test.local", "secret123")

	res, err := e.accounts.Update(e.ctx, root.ID, other.ID, setRole(models.RoleAdmin))
	e.must(err)
	if res.Role != models.RoleAdmin {
		t.Fatalf("promoted role = %s", res.Role)
	}
	if _, err := e.accounts.Update(e.ctx, root.ID, root.ID, setRole(models.RoleStaff)); !isAppErr(err, apperrors.ErrCannotDemoteSelf) {
		t.Fatalf("demote self: err = %v", err)
	}
	if _, err := e.accounts.Update(e.ctx, root.ID, other.ID, setRole(models.RoleStaff)); err != nil {
		t.Fatalf("demote another admin while one remains: %v", err)
	}
	if _, err := e.accounts.Update(e.ctx, other.ID, root.ID, setRole(models.RoleCustomer)); !isAppErr(err, apperrors.ErrLastAdmin) {
		t.Fatalf("demote the last admin: err = %v", err)
	}
	if _, err := e.accounts.Update(e.ctx, root.ID, other.ID, dto.UpdateUserRequest{}); httpStatus(err) != http.StatusBadRequest {
		t.Fatalf("empty update: err = %v", err)
	}
	session, err := e.auth.Login(e.ctx, login("other@test.local", "secret123", "10.0.0.3"))
	e.must(err)
	if session.User.Role != models.RoleStaff {
		t.Fatalf("login role after demotion = %s", session.User.Role)
	}
}

// T25 / E-U3: a locked account can not log in, refresh, hold or pay, but its ticket still gets in.
func TestLockedAccount_BlocksLoginRefreshAndPurchases(t *testing.T) {
	e := newEnv(t)
	admin := e.newUser(models.RoleAdmin, "admin@test.local", "secret123")
	cust := e.newUser(models.RoleCustomer, "c@test.local", "secret123")

	session, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1"))
	e.must(err)
	bought := e.confirmed(cust.ID, "A1")
	pending := e.mustHold(cust.ID, "A2")

	if _, err := e.accounts.Update(e.ctx, admin.ID, cust.ID, setActive(false)); err != nil {
		t.Fatal(err)
	}

	if _, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1")); !isAppErr(err, apperrors.ErrAccountLocked) {
		t.Fatalf("login: err = %v", err)
	}
	if _, err := e.auth.Refresh(e.ctx, session.RefreshToken); !isAppErr(err, apperrors.ErrAccountLocked) {
		t.Fatalf("refresh: err = %v", err)
	}
	if _, err := e.hold(cust.ID, "A3"); !isAppErr(err, apperrors.ErrAccountLocked) {
		t.Fatalf("hold: err = %v", err)
	}
	if _, err := e.svc.Pay(e.ctx, cust.ID, pending.BookingID, dto.PayRequest{Provider: "mock"}); !isAppErr(err, apperrors.ErrAccountLocked) {
		t.Fatalf("pay: err = %v", err)
	}

	order, err := e.svc.Order(e.ctx, cust.ID, bought)
	e.must(err)
	e.moveShowStart(e.showID, 10*time.Minute)
	if res, err := e.svc.Redeem(e.ctx, order.Tickets[0].Code, e.showID, ""); err != nil || res.Status != models.RedeemOK {
		t.Fatalf("gate for a locked account's ticket: %+v %v", res, err)
	}

	if _, err := e.accounts.Update(e.ctx, admin.ID, cust.ID, setActive(true)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.auth.Login(e.ctx, login("c@test.local", "secret123", "10.0.0.1")); err != nil {
		t.Fatalf("login after unlock: %v", err)
	}
}
