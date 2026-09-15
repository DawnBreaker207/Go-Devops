package router

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Validation errors use json/form field names instead of Go field names.
func registerValidators() {
	engine, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return
	}

	engine.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "" {
			name = strings.SplitN(field.Tag.Get("form"), ",", 2)[0]
		}
		if name == "-" {
			return ""
		}
		if name == "" {
			return field.Name
		}
		return name
	})
}
