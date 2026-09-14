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

// Business error codes, independent of HTTP status so clients stay stable.
const (
	CodeBadRequest         = 40000
	CodeValidation         = 40001
	CodeUnauthorized       = 40100
	CodeTokenExpired       = 40101
	CodeForbidden          = 40300
	CodeNotFound           = 40400
	CodeConflict           = 40900
	CodeTooManyRequests    = 42900
	CodeInternal           = 50000
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

// TooManyRequests returns 429 when a request exceeds the rate limit.
func TooManyRequests(message string) *AppError {
	return newError(http.StatusTooManyRequests, CodeTooManyRequests, message)
}

func Internal(message string) *AppError {
	return newError(http.StatusInternalServerError, CodeInternal, message)
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
	ErrHallHasBookings     = Forbidden("hall layout can not be changed when it has bookings")
	ErrShowtimeOverlap     = Conflict("showtime overlaps an existing one in this hall")
	ErrShowtimeHasBookings = Conflict("showtime can not be deleted when it has bookings")
	ErrSeatValidation      = Validation("invalid seat layout parameters")
	ErrMovieNotShowing     = Validation("movie must be showing to schedule showtimes")
	ErrShowtimeNotOpen     = NotFound("showtime is not open")
)
