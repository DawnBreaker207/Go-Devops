// Package response chuan hoa moi payload tra ve dang { code, message, data }.
package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"

	apperrors "github.com/Cinema-Project-Juann/BackEnd-CP/pkg/errors"
	"github.com/Cinema-Project-Juann/BackEnd-CP/pkg/logger"
)

// Body la khung response chung cua toan bo API.
type Body struct {
	Code    int               `json:"code" example:"0"`
	Message string            `json:"message" example:"success"`
	Data    any               `json:"data,omitempty"`
	Details map[string]string `json:"details,omitempty"`
}

// Meta la thong tin phan trang di kem danh sach.
type Meta struct {
	Page       int   `json:"page" example:"1"`
	PageSize   int   `json:"page_size" example:"10"`
	Total      int64 `json:"total" example:"42"`
	TotalPages int   `json:"total_pages" example:"5"`
}

// Paged la payload danh sach co phan trang.
type Paged struct {
	Items any  `json:"items"`
	Meta  Meta `json:"meta"`
}

// CodeSuccess la ma tra ve khi request thanh cong.
const CodeSuccess = 0

// OK tra ve 200 kem du lieu.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Body{Code: CodeSuccess, Message: "success", Data: data})
}

// Created tra ve 201 kem du lieu vua tao.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Body{Code: CodeSuccess, Message: "created", Data: data})
}

// NoContentOK tra ve 200 khong kem du lieu (vd: xoa thanh cong).
func NoContentOK(c *gin.Context, message string) {
	c.JSON(http.StatusOK, Body{Code: CodeSuccess, Message: message})
}

// List tra ve danh sach da phan trang.
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

// Error map moi loai loi ve dung HTTP status + ma loi nghiep vu.
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

// Abort ghi loi va dung chuoi middleware (dung trong middleware auth).
func Abort(c *gin.Context, err error) {
	appErr := apperrors.From(err)
	c.AbortWithStatusJSON(appErr.Status, Body{
		Code:    appErr.Code,
		Message: appErr.Message,
		Details: appErr.Details,
	})
}

func writeError(c *gin.Context, appErr *apperrors.AppError) {
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
