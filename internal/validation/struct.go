package validation

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(FieldName)
	return v
}

// FieldName reports the name a struct field is given in validation errors: its
// json tag, then its form tag, falling back to the Go field name.
func FieldName(fld reflect.StructField) string {
	for _, tag := range []string{"json", "form"} {
		name, _, _ := strings.Cut(fld.Tag.Get(tag), ",")
		if name == "-" {
			return ""
		}
		if name != "" {
			return name
		}
	}
	return ""
}

// UseFieldNames applies [FieldName] to another validator instance, such as the
// one Gin binds requests with.
func UseFieldNames(engine any) {
	if v, ok := engine.(*validator.Validate); ok {
		v.RegisterTagNameFunc(FieldName)
	}
}

// ValidateStruct validates s against its validator struct tags. Excluded fields
// are named by their Go field name, not the JSON name reported in errors.
func ValidateStruct(s any, fields ...string) error {
	return validate.StructExcept(s, fields...)
}
