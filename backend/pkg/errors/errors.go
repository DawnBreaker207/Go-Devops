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
	err     error
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
	ErrUserNotFound        = NotFound("user not found")
	ErrEmailAlreadyExists  = Conflict("email already exists")
	ErrInvalidCredentials  = Unauthorized("email or password is incorrect")
	ErrInvalidToken        = Unauthorized("invalid or expired token")
	ErrMovieNotFound       = NotFound("movie not found")
	ErrJobNotFound         = NotFound("batch job not found")
	ErrJobRunning          = Conflict("batch job is already running")
	ErrHallNotFound        = NotFound("hall not found")
	ErrShowtimeNotFound    = NotFound("showtime not found")
	ErrSeatNotFound        = NotFound("seat not found")
	ErrHallNameExists      = Conflict("hall name already exists")
	ErrHallHasBookings     = Conflict("hall layout can not be changed when it has bookings")
	ErrShowtimeOverlap     = Conflict("showtime overlaps an existing one in this hall")
	ErrShowtimeHasBookings = Conflict("showtime can not be deleted when it has bookings")
	ErrSeatValidation      = Validation("invalid seat layout parameters")
	ErrMovieNotShowing     = Validation("movie must be showing to schedule showtimes")
	ErrShowtimeNotOpen     = NotFound("showtime is not open")
	ErrShowtimeClosed      = Forbidden("showtime is closed for sales or has already started")

	// Changing a showtime that already sells seats
	ErrShowtimeScheduleLocked = Conflict("showtime with pending or confirmed bookings can only be opened or closed")
	ErrShowtimeHallLocked     = Conflict("showtime hall can not change once it has bookings")
	ErrShowtimeChanged        = Conflict("showtime was changed by someone else, reload and retry")
	ErrShowtimeReopenLocked   = Conflict("showtime can only reopen while its movie is showing and before it starts")

	// Changing a movie that still has showtimes to come (F2 E-M2, E-M4).
	ErrMovieHasShowtimes   = Conflict("movie has open showtimes still to come; close or delete them first")
	ErrMovieDurationLocked = Conflict("movie duration can not change while it has showtimes still to come")

	// An idempotency key backs a single hold request
	ErrIdempotencyKeyReused = Conflict("idempotency key was already used for another request")

	ErrBookingNotFound   = NotFound("booking not found")
	ErrSeatTaken         = Conflict("one or more seats are no longer available")
	ErrSeatNotSellable   = Validation("one or more seats can not be sold")
	ErrSeatLimitExceeded = Validation("too many seats in a single booking")
	ErrBookingExpired    = Conflict("booking hold has expired")
	ErrBookingNotPending = Conflict("booking is not in a payable state")
	ErrBookingNotPaid    = Conflict("booking must be paid before confirmation")
	ErrBookingRefunded   = Conflict("booking could not be confirmed; the payment was refunded")
	ErrPaymentInProgress = Conflict("a payment is already in progress for your pending booking")
	ErrPaymentGateway    = BadGateway("payment provider is unavailable, please retry")
	ErrInvalidSignature  = Unauthorized("invalid payment signature")
	ErrMissingHallPrice  = Conflict("hall has no price for one of the requested seat types")
	ErrTicketNotFound    = NotFound("ticket not found")

	ErrAccountLocked         = Forbidden("account is locked")
	ErrTooManyLoginAttempts  = TooManyRequests("too many failed login attempts, try again later")
	ErrTermsRequired         = PreconditionRequired("you must accept the current terms before continuing")
	ErrAccountHoldsTickets   = Conflict("the account still holds confirmed tickets to come; use or refund them before deleting")
	ErrCannotLockSelf        = Conflict("you can not lock your own account")
	ErrLastAdmin             = Conflict("at least one active admin must remain")
	ErrCannotDemoteSelf      = Conflict("you can not change your own role")
	ErrUploadInvalid         = Validation("file must be a JPEG, PNG or WebP image")
	ErrUploadTooLarge        = Validation("file is too large")
	ErrImageStoreUnavailable = BadGateway("image storage is unavailable, please retry")
)
