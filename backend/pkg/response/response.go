// Package response standardizes the { code, message, data } payload.
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"

	"github.com/Cinema-Project-Juann/BackEnd-CP/internal/audit"
)

// Body is the common response shape.
type Body struct {
	Code    int               `json:"code" example:"0"`
	Message string            `json:"message" example:"success"`
	Data    any               `json:"data,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// Meta is the pagination info.
type Meta struct {
	Page       int   `json:"page" example:"1"`
	PageSize   int   `json:"page_size" example:"10"`
	Total      int64 `json:"total" example:"42"`
	TotalPages int   `json:"total_pages" example:"5"`
}

// Paged is a paginated list payload.
type Paged struct {
	Items any  `json:"items"`
	Meta  Meta `json:"meta"`
}

// CodeSuccess is the code returned on success.
const CodeSuccess = 0

// OK returns 200 with data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: CodeSuccess, Message: "success", Data: data})
}

// Created returns 201 with the created data.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{Code: CodeSuccess, Message: "created", Data: data})
}

// NoContentOK returns 200 without data.
func NoContentOK(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Body{Code: CodeSuccess, Message: message})
}

// List returns a paginated list.
func List(c *gin.Context, items any, page, pageSize int, total int64) {
	totalPages := 0
	if pageSize > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	OK(c, Paged{
		Items: items,
		Meta:  Meta{Page: page, PageSize: pageSize, Total: total, TotalPages: totalPages},
	})
}

// Error maps errors to HTTP status and business code.
func Error(c *gin.Context, err error) {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		appErr := apperrors.Validation("validation failed").WithDetails(validationDetails(validationErrs))
		writeError(c, appErr)
		return
	}

	appErr := apperrors.From(err)
	if appErr.Status >= http.StatusInternalServerError {
		logger.L().Error("request failed", logger.Err(err), logger.String("path", c.FullPath()))
	}
	writeError(c, appErr)
}

// Abort writes an error and stops the middleware chain.
func Abort(c *gin.Context, err error) {
	appErr := apperrors.From(err)
	c.Set(audit.ErrorMsgKey, appErr.Message)
	c.AbortWithStatusJSON(appErr.Status, Body{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	})
}

func writeError(c *gin.Context, appErr *apperrors.AppError) {
	c.Set(audit.ErrorMsgKey, appErr.Message)
	c.JSON(appErr.Status, Body{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	})
}

func validationDetails(errs validator.ValidationErrors) map[string]string {
	details := make(map[string]string, len(errs))
	for _, fieldErr := range errs {
		details[fieldErr.Field()] = messageForTag(fieldErr)
	}
	return details
}

func messageForTag(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "field is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fieldErr.Param()
	case "max":
		return "must be at most " + fieldErr.Param()
	case "oneof":
		return "must be one of: " + fieldErr.Param()
	case "datetime":
		return "must match format " + fieldErr.Param()
	default:
		return "failed on rule " + fieldErr.Tag()
	}
}
