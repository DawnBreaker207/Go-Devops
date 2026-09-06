// Package apperrors dinh nghia kieu loi dung chung cho toan bo service.
// Service tra ve *AppError, handler chi can goi response.Error de map ra HTTP status.
package apperrors

import (
	"errors"
	"fmt"
	"net/http"
)

// Ma loi nghiep vu, doc lap voi HTTP status de client xu ly on dinh.
const (
	CodeBadRequest   = 40000
	CodeValidation   = 40001
	CodeUnauthorized = 40100
	CodeTokenExpired = 40101
	CodeForbidden    = 40300
	CodeNotFound     = 40400
	CodeConflict     = 40900
	CodeInternal     = 50000
)

// AppError la loi co mang theo HTTP status + ma loi nghiep vu.
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

// Wrap gan them loi goc de ghi log, message tra ve client giu nguyen.
func (e *AppError) Wrap(err error) *AppError {
	clone := *e
	clone.err = err
	return &clone
}

// WithDetails gan chi tiet loi theo tung field (dung cho loi validate).
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

func Internal(message string) *AppError {
	return newError(http.StatusInternalServerError, CodeInternal, message)
}

// From tra ve *AppError neu err thuoc chuoi loi cua ung dung,
// nguoc lai quy ve loi he thong 500.
func From(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal("internal server error").Wrap(err)
}

// Loi dung lai nhieu noi.
var (
	ErrUserNotFound       = NotFound("user not found")
	ErrEmailAlreadyExists = Conflict("email already exists")
	ErrInvalidCredentials = Unauthorized("email or password is incorrect")
	ErrInvalidToken       = Unauthorized("invalid or expired token")
	ErrMovieNotFound      = NotFound("movie not found")
)
