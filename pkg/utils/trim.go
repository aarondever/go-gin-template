package utils

import (
	"reflect"
	"strings"
)

func TrimStruct(s any) {
	v := reflect.ValueOf(s)

	// must be a pointer to a struct
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return
	}

	trimValue(v.Elem())
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
