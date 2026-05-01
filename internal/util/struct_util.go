package util

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidateStruct trims whitespace on all string fields of s, then validates it
// using go-playground/validator struct tags. Fields named in the optional
// exclude list are skipped during validation but are still trimmed.
// s must be a non-nil pointer to a struct; any other type returns an error.
// Returns validator.ValidationErrors when one or more fields fail their tag constraints.
func ValidateStruct(s any, fields ...string) error {
	v := reflect.ValueOf(s)

	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("must be a pointer to a struct")
	}

	trimValue(v.Elem())

	validate := validator.New()
	return validate.StructExcept(s, fields...)
}

func trimValue(v reflect.Value) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() {
				continue // skip unexported fields
			}
			trimValue(field)
		}

	case reflect.String:
		v.SetString(strings.TrimSpace(v.String()))

	case reflect.Pointer:
		if !v.IsNil() {
			// Only handle *string — set nil if empty after trim
			if v.Elem().Kind() == reflect.String {
				trimmed := strings.TrimSpace(v.Elem().String())
				if trimmed == "" {
					v.Set(reflect.Zero(v.Type())) // assign nil
				} else {
					v.Elem().SetString(trimmed)
				}
			} else {
				trimValue(v.Elem())
			}
		}

	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			trimValue(v.Index(i))
		}
	}
}
