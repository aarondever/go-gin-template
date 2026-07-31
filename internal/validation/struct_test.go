package validation

import (
	"errors"
	"reflect"
	"testing"

	"github.com/go-playground/validator/v10"
)

// fieldOf pulls a named field off a struct type so its tags can be inspected.
func fieldOf(t *testing.T, s any, name string) reflect.StructField {
	t.Helper()
	fld, ok := reflect.TypeOf(s).FieldByName(name)
	if !ok {
		t.Fatalf("field %q not found on %T", name, s)
	}
	return fld
}

func TestFieldName(t *testing.T) {
	type tagged struct {
		JSONOnly     string `json:"json_only"`
		JSONWithOpts string `json:"json_opts,omitempty"`
		JSONSkipped  string `json:"-"`
		FormOnly     string `form:"form_only"`
		FormWithOpts string `form:"form_opts,default=x"`
		Both         string `json:"from_json" form:"from_form"`
		JSONEmpty    string `json:",omitempty" form:"fallback_form"`
		SkipWinsOver string `json:"-" form:"form_name"`
		FormSkipped  string `form:"-"`
		Untagged     string
		OtherTag     string `validate:"required" db:"other"`
	}

	tests := []struct {
		name  string
		field string
		want  string
	}{
		{name: "json tag", field: "JSONOnly", want: "json_only"},
		{name: "json tag options are stripped", field: "JSONWithOpts", want: "json_opts"},
		{name: "json dash is skipped", field: "JSONSkipped", want: ""},
		{name: "form tag when json is absent", field: "FormOnly", want: "form_only"},
		{name: "form tag options are stripped", field: "FormWithOpts", want: "form_opts"},
		{name: "json wins over form", field: "Both", want: "from_json"},
		{name: "empty json name falls through to form", field: "JSONEmpty", want: "fallback_form"},
		{name: "json dash wins over form", field: "SkipWinsOver", want: ""},
		{name: "form dash is skipped", field: "FormSkipped", want: ""},
		{name: "untagged field", field: "Untagged", want: ""},
		{name: "unrelated tags are ignored", field: "OtherTag", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FieldName(fieldOf(t, tagged{}, tt.field)); got != tt.want {
				t.Errorf("FieldName(%s) = %q, want %q", tt.field, got, tt.want)
			}
		})
	}
}

func TestUseFieldNames(t *testing.T) {
	type payload struct {
		Email string `json:"email" validate:"required"`
	}

	v := validator.New()
	UseFieldNames(v)

	err := v.Struct(payload{})

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("Struct() error = %v, want validator.ValidationErrors", err)
	}
	if ve[0].Field() != "email" {
		t.Errorf("Field() = %q, want the json name %q", ve[0].Field(), "email")
	}
}

// UseFieldNames takes `any` because Gin hands back its engine untyped; anything
// that is not a *validator.Validate is ignored rather than panicking.
func TestUseFieldNamesIgnoresOtherTypes(t *testing.T) {
	for _, engine := range []any{nil, "not a validator", 42, struct{}{}, validator.New()} {
		UseFieldNames(engine)
	}
}

func TestValidateStruct(t *testing.T) {
	type payload struct {
		Email string `json:"email" validate:"required,email"`
		Name  string `json:"name" validate:"required"`
		Age   int    `json:"age" validate:"gte=18"`
	}

	tests := []struct {
		name    string
		input   payload
		wantErr bool
		// field name -> failed tag
		wantFields map[string]string
	}{
		{
			name:  "valid",
			input: payload{Email: "a@b.com", Name: "Aaron", Age: 30},
		},
		{
			name:       "single failure",
			input:      payload{Email: "a@b.com", Name: "Aaron", Age: 10},
			wantErr:    true,
			wantFields: map[string]string{"age": "gte"},
		},
		{
			name:       "multiple failures",
			input:      payload{},
			wantErr:    true,
			wantFields: map[string]string{"email": "required", "name": "required", "age": "gte"},
		},
		{
			name:       "malformed email",
			input:      payload{Email: "not-an-email", Name: "Aaron", Age: 30},
			wantErr:    true,
			wantFields: map[string]string{"email": "email"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.input)

			if !tt.wantErr {
				if err != nil {
					t.Fatalf("ValidateStruct() error = %v, want nil", err)
				}
				return
			}

			var ve validator.ValidationErrors
			if !errors.As(err, &ve) {
				t.Fatalf("ValidateStruct() error = %v, want validator.ValidationErrors", err)
			}

			got := make(map[string]string, len(ve))
			for _, fe := range ve {
				got[fe.Field()] = fe.Tag()
			}
			if len(got) != len(tt.wantFields) {
				t.Fatalf("failures = %v, want %v", got, tt.wantFields)
			}
			for field, tag := range tt.wantFields {
				if got[field] != tag {
					t.Errorf("failures[%q] = %q, want %q", field, got[field], tag)
				}
			}
		})
	}
}

// The package validator is registered with FieldName, so errors carry the json
// name rather than the Go field name.
func TestValidateStructUsesFieldNames(t *testing.T) {
	type payload struct {
		FirstName string `json:"first_name" validate:"required"`
		LastName  string `form:"last_name" validate:"required"`
		Untagged  string `validate:"required"`
	}

	err := ValidateStruct(payload{})

	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		t.Fatalf("ValidateStruct() error = %v, want validator.ValidationErrors", err)
	}

	got := make(map[string]bool, len(ve))
	for _, fe := range ve {
		got[fe.Field()] = true
	}
	// An empty name from FieldName leaves the Go field name in place.
	for _, want := range []string{"first_name", "last_name", "Untagged"} {
		if !got[want] {
			t.Errorf("failures = %v, want a failure named %q", got, want)
		}
	}
}

// Exclusions are named by the Go field name, not the json name.
func TestValidateStructExcludesFields(t *testing.T) {
	type payload struct {
		Email string `json:"email" validate:"required"`
		Name  string `json:"name" validate:"required"`
	}

	t.Run("excluding by Go field name skips it", func(t *testing.T) {
		err := ValidateStruct(payload{Name: "Aaron"}, "Email")

		if err != nil {
			t.Fatalf("ValidateStruct() error = %v, want nil", err)
		}
	})

	t.Run("excluding every failing field passes", func(t *testing.T) {
		if err := ValidateStruct(payload{}, "Email", "Name"); err != nil {
			t.Fatalf("ValidateStruct() error = %v, want nil", err)
		}
	})

	t.Run("excluding by json name has no effect", func(t *testing.T) {
		err := ValidateStruct(payload{Name: "Aaron"}, "email")

		var ve validator.ValidationErrors
		if !errors.As(err, &ve) {
			t.Fatalf("ValidateStruct() error = %v, want the Email failure to remain", err)
		}
		if len(ve) != 1 || ve[0].Field() != "email" {
			t.Errorf("failures = %v, want a single failure on email", ve)
		}
	})

	t.Run("unknown exclusion is harmless", func(t *testing.T) {
		if err := ValidateStruct(payload{Email: "a@b.com", Name: "Aaron"}, "Nope"); err != nil {
			t.Fatalf("ValidateStruct() error = %v, want nil", err)
		}
	})
}

func TestValidateStructPointer(t *testing.T) {
	type payload struct {
		Email string `json:"email" validate:"required"`
	}

	if err := ValidateStruct(&payload{Email: "a@b.com"}); err != nil {
		t.Errorf("ValidateStruct() error = %v, want nil", err)
	}
	if err := ValidateStruct(&payload{}); err == nil {
		t.Error("ValidateStruct() = nil, want a validation error")
	}
}

// The package-level validator is built once and shared, so it must be safe to
// use from concurrent request handlers.
func TestValidateStructConcurrent(t *testing.T) {
	type payload struct {
		Email string `json:"email" validate:"required,email"`
	}

	done := make(chan bool, 50)
	for i := range 50 {
		go func(i int) {
			if i%2 == 0 {
				done <- ValidateStruct(payload{Email: "a@b.com"}) == nil
				return
			}
			done <- ValidateStruct(payload{}) != nil
		}(i)
	}
	for range 50 {
		if !<-done {
			t.Error("concurrent ValidateStruct() gave an unexpected result")
		}
	}
}
