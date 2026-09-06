package router

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// registerValidators cau hinh validator cua gin de bao loi theo ten field json
// (vd "release_date") thay vi ten field Go ("ReleaseDate").
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
