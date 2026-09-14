package router

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Use json field names (e.g. "release_date") in validation errors instead of Go field names.
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
