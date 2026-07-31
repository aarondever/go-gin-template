package util

import (
	"reflect"
	"strings"
)

// TrimStructStr trims whitespace from every string field of s and returns s.
// s must be a pointer to a struct; otherwise it is returned unmodified.
func TrimStructStr(s any) any {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Pointer && v.Elem().Kind() == reflect.Struct {
		trimStr(v.Elem())
	}
	return s
}

// trimStr recursively trims whitespace from strings reachable from v via
// struct fields and pointers. An empty *string is set to nil.
func trimStr(v reflect.Value) {
	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() {
				continue // skip unexported fields
			}
			trimStr(field)
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
				trimStr(v.Elem())
			}
		}
	}
}
