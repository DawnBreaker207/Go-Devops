// Package apperrors defines shared error types. Services return *AppError;
// handlers map it to an HTTP status via response.Error.
package apperrors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation reports a PostgreSQL unique-constraint violation (23505).
// Used to recognize "duplicate pending booking" and "job already running"
// so callers can merge/replace instead of failing.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// IsExclusionViolation reports a PostgreSQL exclusion-constraint violation
// (23P01), the btree_gist backstop for overlapping showtimes in one hall.
func IsExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23P01"
	}
	return false
}

// IsDeadlock reports a PostgreSQL deadlock error (40P01), retried by the
// seat-hold flow instead of surfacing to the client
func IsDeadlock(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40P01"
	}
	return false
}

// IsRetryable reports transaction races worth re-running from the top:
// unique violation (a concurrent insert won), deadlock, serialization failure.
func IsRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505", "40P01", "40001":
			return true
		}
	}
	return false
}

// Business error codes, independent of HTTP status so clients stay stable.
const (
	CodeBadRequest         = 40000
	CodeValidation         = 40001
	CodeUnauthorized       = 40100
	CodeTokenExpired       = 40101
	CodeForbidden          = 40300
	CodeNotFound           = 40400
	CodeConflict           = 40900
	CodePayloadTooLarge    = 41300
	CodeTermsRequired      = 42800
	CodeTooManyRequests    = 42900
	CodeInternal           = 50000
	CodeBadGateway         = 50200
	CodeServiceUnavailable = 50300
)

// AppError carries an HTTP status plus a business error code.
type AppError struct {
	Status  int               `json:"-"`
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
	// Reason is a stable machine-readable key (e.g. "discount_expired") that
	// stays constant while Message wording may change. Omitted when empty so
	// older clients see no difference.
	Reason string `json:"reason,omitempty"`
	err    error
}

func (e *AppError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.err }

// Wrap attaches the root cause for logs; the client message stays as is.
func (e *AppError) Wrap(err error) *AppError {
	clone := *e
	clone.err = err
	return &clone
}

// WithDetails attaches per-field error details (used for validation).
func (e *AppError) WithDetails(details map[string]string) *AppError {
	clone := *e
	clone.Details = details
	return &clone
}

// WithReason attaches a stable key. Like Wrap/WithDetails it returns a clone,
// so errors.Is still matches the unwrapped sentinel.
func (e *AppError) WithReason(reason string) *AppError {
	clone := *e
	clone.Reason = reason
	return &clone
}

func newError(status, code int, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func BadRequest(message string) *AppError {
	return newError(http.StatusBadRequest, CodeBadRequest, message)
}

func Validation(message string) *AppError {
	return newError(http.StatusBadRequest, CodeValidation, message)
}

func Unauthorized(message string) *AppError {
	return newError(http.StatusUnauthorized, CodeUnauthorized, message)
}

func TokenExpired(message string) *AppError {
	return newError(http.StatusUnauthorized, CodeTokenExpired, message)
}

func Forbidden(message string) *AppError {
	return newError(http.StatusForbidden, CodeForbidden, message)
}

func NotFound(message string) *AppError {
	return newError(http.StatusNotFound, CodeNotFound, message)
}

func Conflict(message string) *AppError {
	return newError(http.StatusConflict, CodeConflict, message)
}

// PayloadTooLarge returns 413 when a request body exceeds its limit.
func PayloadTooLarge(message string) *AppError {
	return newError(http.StatusRequestEntityTooLarge, CodePayloadTooLarge, message)
}

// PreconditionRequired returns 428 when the account must accept the current terms first.
func PreconditionRequired(message string) *AppError {
	return newError(http.StatusPreconditionRequired, CodeTermsRequired, message)
}

// TooManyRequests returns 429 when a request exceeds the rate limit.
func TooManyRequests(message string) *AppError {
	return newError(http.StatusTooManyRequests, CodeTooManyRequests, message)
}

func Internal(message string) *AppError {
	return newError(http.StatusInternalServerError, CodeInternal, message)
}

// BadGateway returns 502 when an upstream provider (payment gateway) fails.
func BadGateway(message string) *AppError {
	return newError(http.StatusBadGateway, CodeBadGateway, message)
}

// ServiceUnavailable returns 503 when the server is temporarily unready (e.g. DB down).
func ServiceUnavailable(message string) *AppError {
	return newError(http.StatusServiceUnavailable, CodeServiceUnavailable, message)
}

// From returns the *AppError for app errors, or falls back to a 500 system error.
func From(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "22P02": // invalid input syntax, e.g. a malformed uuid in the path
			return BadRequest("invalid identifier or value").Wrap(err)
		case "23503":
			return BadRequest("a referenced resource does not exist").Wrap(err)
		}
	}
	return Internal("internal server error").Wrap(err)
}

// Shared errors.
var (
	ErrUserNotFound             = NotFound("user not found").WithReason("user_not_found")
	ErrEmailAlreadyExists       = Conflict("email already exists").WithReason("email_already_exists")
	ErrInvalidCredentials       = Unauthorized("email or password is incorrect").WithReason("invalid_credentials")
	ErrInvalidToken             = Unauthorized("invalid or expired token").WithReason("invalid_token")
	ErrMovieNotFound            = NotFound("movie not found").WithReason("movie_not_found")
	ErrJobNotFound              = NotFound("batch job not found").WithReason("job_not_found")
	ErrJobRunning               = Conflict("batch job is already running").WithReason("job_running")
	ErrHallNotFound             = NotFound("hall not found").WithReason("hall_not_found")
	ErrShowtimeNotFound         = NotFound("showtime not found").WithReason("showtime_not_found")
	ErrSeatNotFound             = NotFound("seat not found").WithReason("seat_not_found")
	ErrHallNameExists           = Conflict("hall name already exists").WithReason("hall_name_exists")
	ErrHallHasBookings          = Conflict("hall layout can not be changed when it has bookings").WithReason("hall_has_bookings")
	ErrHallEverHadBookings      = Conflict("hall layout can only be regenerated before it has ever had a booking").WithReason("hall_ever_had_bookings")
	ErrHallInactive             = Conflict("hall is inactive and takes no new showtimes").WithReason("hall_inactive")
	ErrHallStillSelling         = Conflict("hall still has an open showtime still to come").WithReason("hall_still_selling")
	ErrHallHasUpcomingShowtimes = Conflict("hall has showtimes not yet ended").WithReason("hall_has_upcoming_showtimes")
	ErrHallRowLimitReached      = Validation("hall already has the maximum number of rows").WithReason("hall_row_limit_reached")
	ErrSeatEverHadBooking       = Conflict("one or more seats have booking history and can not be changed this way").WithReason("seat_ever_had_booking")
	ErrSeatNotMergeable         = Validation("seats are not adjacent standard seats and can not be merged").WithReason("seat_not_mergeable")
	ErrSeatNotCouple            = Validation("seat is not a couple seat").WithReason("seat_not_couple")
	ErrShowtimeOverlap          = Conflict("showtime overlaps an existing one in this hall").WithReason("showtime_overlap")
	ErrShowtimeHasBookings      = Conflict("showtime can not be deleted when it has bookings").WithReason("showtime_has_bookings")
	ErrSeatValidation           = Validation("invalid seat layout parameters").WithReason("seat_validation")
	ErrMovieNotShowing          = Validation("movie must be showing to schedule showtimes").WithReason("movie_not_showing")
	ErrShowtimeNotOpen          = NotFound("showtime is not open").WithReason("showtime_not_open")
	ErrShowtimeClosed           = Forbidden("showtime is closed for sales or has already started").WithReason("showtime_closed")

	// Changing a showtime that already sells seats
	ErrShowtimeScheduleLocked = Conflict("showtime with pending or confirmed bookings can only be opened or closed").WithReason("showtime_schedule_locked")
	ErrShowtimeHallLocked     = Conflict("showtime hall can not change once it has bookings").WithReason("showtime_hall_locked")
	ErrShowtimeChanged        = Conflict("showtime was changed by someone else, reload and retry").WithReason("showtime_changed")
	ErrShowtimeReopenLocked   = Conflict("showtime can only reopen while its movie is showing and before it starts").WithReason("showtime_reopen_locked")

	// Changing a movie that still has showtimes to come (F2 E-M2, E-M4).
	ErrMovieHasShowtimes   = Conflict("movie has open showtimes still to come; close or delete them first").WithReason("movie_has_showtimes")
	ErrMovieDurationLocked = Conflict("movie duration can not change while it has showtimes still to come").WithReason("movie_duration_locked")

	// An idempotency key backs a single hold request
	ErrIdempotencyKeyReused = Conflict("idempotency key was already used for another request").WithReason("idempotency_key_reused")

	ErrBookingNotFound      = NotFound("booking not found").WithReason("booking_not_found")
	ErrSeatTaken            = Conflict("one or more seats are no longer available").WithReason("seat_taken")
	ErrSeatNotSellable      = Validation("one or more seats can not be sold").WithReason("seat_not_sellable")
	ErrSeatLimitExceeded    = Validation("too many seats in a single booking").WithReason("seat_limit_exceeded")
	ErrBookingExpired       = Conflict("booking hold has expired").WithReason("booking_expired")
	ErrBookingNotPending    = Conflict("booking is not in a payable state").WithReason("booking_not_pending")
	ErrBookingNotPaid       = Conflict("booking must be paid before confirmation").WithReason("booking_not_paid")
	ErrBookingRefunded      = Conflict("booking could not be confirmed; the payment was refunded").WithReason("booking_refunded")
	ErrBookingEmpty         = Conflict("booking has no seats and can not be paid or confirmed").WithReason("booking_empty")
	ErrHoldLifetimeExceeded = Conflict("booking reached its maximum lifetime; start a new booking").WithReason("hold_lifetime_exceeded")
	ErrPaymentInProgress    = Conflict("a payment is already in progress for your pending booking").WithReason("payment_in_progress")
	ErrPaymentGateway       = BadGateway("payment provider is unavailable, please retry").WithReason("payment_gateway")
	ErrInvalidSignature     = Unauthorized("invalid payment signature").WithReason("invalid_signature")
	ErrMissingHallPrice     = Conflict("hall has no price for one of the requested seat types").WithReason("missing_hall_price")
	ErrTicketNotFound       = NotFound("ticket not found").WithReason("ticket_not_found")

	ErrAccountLocked         = Forbidden("account is locked").WithReason("account_locked")
	ErrTooManyLoginAttempts  = TooManyRequests("too many failed login attempts, try again later").WithReason("too_many_login_attempts")
	ErrTermsRequired         = PreconditionRequired("you must accept the current terms before continuing").WithReason("terms_required")
	ErrAccountHoldsTickets   = Conflict("the account still holds confirmed tickets to come; use or refund them before deleting").WithReason("account_holds_tickets")
	ErrCannotLockSelf        = Conflict("you can not lock your own account").WithReason("cannot_lock_self")
	ErrLastAdmin             = Conflict("at least one active admin must remain").WithReason("last_admin")
	ErrCannotDemoteSelf      = Conflict("you can not change your own role").WithReason("cannot_demote_self")
	ErrUploadInvalid         = Validation("file must be a JPEG, PNG or WebP image").WithReason("upload_invalid")
	ErrUploadTooLarge        = Validation("file is too large").WithReason("upload_too_large")
	ErrImageStoreUnavailable = BadGateway("image storage is unavailable, please retry").WithReason("image_store_unavailable")

	ErrSessionNotFound = NotFound("session not found").WithReason("session_not_found")

	ErrShowtimeAlreadyCancelled = Conflict("showtime is already cancelled").WithReason("showtime_already_cancelled")

	ErrComboNotFound   = NotFound("combo not found").WithReason("combo_not_found")
	ErrComboInactive   = Validation("combo is not available").WithReason("combo_inactive")
	ErrComboOrderEmpty = Validation("combo order must have at least one item").WithReason("combo_order_empty")

	// Discount codes. Every reason a code will not apply is a DISTINCT sentence,
	// because they all share code 40001 and the customer can only be told apart
	// by the message. ErrDiscountNotFound is deliberately a 404 on the admin
	// routes but is never used to answer a customer's apply attempt: an unknown
	// code and an expired one both answer ErrDiscountInvalid, so the endpoint
	// cannot be used to enumerate which codes exist.
	ErrDiscountNotFound        = NotFound("discount code not found").WithReason("discount_not_found")
	ErrDiscountInvalid         = Validation("this discount code is not valid").WithReason("discount_invalid")
	ErrDiscountExpired         = Validation("this discount code is no longer valid").WithReason("discount_expired")
	ErrDiscountNotStarted      = Validation("this discount code is not active yet").WithReason("discount_not_started")
	ErrDiscountExhausted   = Validation("this discount code has been fully redeemed").WithReason("discount_exhausted")
	ErrDiscountMinOrder        = Validation("the order total is below this code's minimum").WithReason("discount_min_order")
	ErrDiscountAlreadySet      = Conflict("this order already has a discount code").WithReason("discount_already_set")
	ErrDiscountNone            = Validation("this order has no discount code to remove").WithReason("discount_none")
	ErrDiscountOrderClosed     = Conflict("a discount can only be applied before payment").WithReason("discount_order_closed")
	ErrDiscountCodeExists      = Conflict("this discount code already exists").WithReason("discount_code_exists")
	ErrDiscountAlreadyRedeemed = Conflict("you have already redeemed this discount code").WithReason("discount_already_redeemed")

	// Campaigns gate linked discount codes: a code with a campaign also needs
	// that campaign active and inside [starts_at, ends_at).
	ErrCampaignNotFound = NotFound("campaign not found").WithReason("campaign_not_found")
	ErrCampaignInactive = Validation("this discount code's campaign is not currently running").WithReason("campaign_inactive")

	// Pricing engine (global base prices + adjustment rules).
	ErrPricingRuleNotFound = NotFound("pricing rule not found").WithReason("pricing_rule_not_found")
	ErrPricingTimeInvalid  = Validation("start_time/end_time must be HH:MM or HH:MM:SS, with end_time after start_time").WithReason("pricing_time_invalid")
	ErrPricingDateInvalid  = Validation("specific_date must be YYYY-MM-DD").WithReason("pricing_date_invalid")
	ErrSeatTypeInvalid     = Validation("invalid seat type").WithReason("seat_type_invalid")

	// Article CMS.
	ErrArticleNotFound = NotFound("article not found").WithReason("article_not_found")
)
