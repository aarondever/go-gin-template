package util

import (
	"testing"
	"time"
)

func ptr[T any](v T) *T { return &v }

func TestTrimStructStrFields(t *testing.T) {
	type payload struct {
		Name  string
		Email string
		Bio   string
	}

	in := &payload{
		Name:  "  Aaron  ",
		Email: "\taaron@example.com\n",
		Bio:   "   ",
	}

	TrimStructStr(in)

	want := payload{Name: "Aaron", Email: "aaron@example.com", Bio: ""}
	if *in != want {
		t.Errorf("after trim = %+v, want %+v", *in, want)
	}
}

func TestTrimStructStrWhitespaceForms(t *testing.T) {
	type payload struct{ Value string }

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "leading and trailing spaces", value: "  hi  ", want: "hi"},
		{name: "tabs", value: "\thi\t", want: "hi"},
		{name: "newlines", value: "\nhi\n", want: "hi"},
		{name: "carriage return", value: "hi\r\n", want: "hi"},
		{name: "mixed", value: " \t\n hi \r\n ", want: "hi"},
		{name: "unicode non-breaking space", value: " hi ", want: "hi"},
		{name: "inner spaces are kept", value: "  a  b  ", want: "a  b"},
		{name: "already clean", value: "hi", want: "hi"},
		{name: "empty", value: "", want: ""},
		{name: "only whitespace", value: " \t\n ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := &payload{Value: tt.value}

			TrimStructStr(in)

			if in.Value != tt.want {
				t.Errorf("Value = %q, want %q", in.Value, tt.want)
			}
		})
	}
}

// The argument is returned so calls can be chained; it is the same pointer,
// mutated in place, not a copy.
func TestTrimStructStrReturnsSamePointer(t *testing.T) {
	type payload struct{ Name string }

	in := &payload{Name: "  Aaron  "}

	got := TrimStructStr(in)

	out, ok := got.(*payload)
	if !ok {
		t.Fatalf("TrimStructStr() returned %T, want *payload", got)
	}
	if out != in {
		t.Error("TrimStructStr() returned a different pointer, want the argument back")
	}
	if out.Name != "Aaron" {
		t.Errorf("Name = %q, want %q", out.Name, "Aaron")
	}
}

func TestTrimStructStrStringPointers(t *testing.T) {
	type payload struct {
		Set     *string
		Blank   *string
		Spaces  *string
		Missing *string
	}

	in := &payload{
		Set:    ptr("  Aaron  "),
		Blank:  ptr(""),
		Spaces: ptr("   "),
	}

	TrimStructStr(in)

	if in.Set == nil || *in.Set != "Aaron" {
		t.Errorf("Set = %v, want a pointer to %q", in.Set, "Aaron")
	}
	// An empty *string is normalised to nil, so optional fields read as absent.
	if in.Blank != nil {
		t.Errorf("Blank = %q, want nil", *in.Blank)
	}
	if in.Spaces != nil {
		t.Errorf("Spaces = %q, want nil", *in.Spaces)
	}
	if in.Missing != nil {
		t.Errorf("Missing = %q, want it left nil", *in.Missing)
	}
}

func TestTrimStructStrNestedStructs(t *testing.T) {
	type address struct {
		City string
		Zip  *string
	}
	type payload struct {
		Name    string
		Home    address
		Work    *address
		Missing *address
	}

	in := &payload{
		Name: "  Aaron  ",
		Home: address{City: "  Vancouver  ", Zip: ptr("  V6B  ")},
		Work: &address{City: "\tSeattle\n", Zip: ptr("  ")},
	}

	TrimStructStr(in)

	if in.Name != "Aaron" {
		t.Errorf("Name = %q, want %q", in.Name, "Aaron")
	}
	if in.Home.City != "Vancouver" {
		t.Errorf("Home.City = %q, want %q", in.Home.City, "Vancouver")
	}
	if in.Home.Zip == nil || *in.Home.Zip != "V6B" {
		t.Errorf("Home.Zip = %v, want a pointer to %q", in.Home.Zip, "V6B")
	}
	if in.Work.City != "Seattle" {
		t.Errorf("Work.City = %q, want %q", in.Work.City, "Seattle")
	}
	if in.Work.Zip != nil {
		t.Errorf("Work.Zip = %q, want nil", *in.Work.Zip)
	}
	if in.Missing != nil {
		t.Error("Missing pointer was populated, want it left nil")
	}
}

func TestTrimStructStrDeeplyNested(t *testing.T) {
	type level3 struct{ Value string }
	type level2 struct {
		Deep   level3
		Deeper *level3
	}
	type level1 struct{ Mid level2 }

	in := &level1{Mid: level2{
		Deep:   level3{Value: "  a  "},
		Deeper: &level3{Value: "  b  "},
	}}

	TrimStructStr(in)

	if in.Mid.Deep.Value != "a" {
		t.Errorf("Mid.Deep.Value = %q, want %q", in.Mid.Deep.Value, "a")
	}
	if in.Mid.Deeper.Value != "b" {
		t.Errorf("Mid.Deeper.Value = %q, want %q", in.Mid.Deeper.Value, "b")
	}
}

// A **string reaches the string through the pointer branch twice.
func TestTrimStructStrPointerToPointer(t *testing.T) {
	type payload struct {
		Value *(*string)
		Blank *(*string)
	}

	in := &payload{Value: ptr(ptr("  hi  ")), Blank: ptr(ptr("   "))}

	TrimStructStr(in)

	if **in.Value != "hi" {
		t.Errorf("Value = %q, want %q", **in.Value, "hi")
	}
	// The inner pointer is cleared, not the outer one.
	if *in.Blank != nil {
		t.Errorf("Blank inner pointer = %q, want nil", **in.Blank)
	}
}

func TestTrimStructStrLeavesNonStringsAlone(t *testing.T) {
	type payload struct {
		Name    string
		Count   int
		Ratio   float64
		Active  bool
		Bytes   []byte
		Tags    []string
		Lookup  map[string]string
		Created time.Time
	}

	created := time.Now()
	in := &payload{
		Name:    "  Aaron  ",
		Count:   42,
		Ratio:   1.5,
		Active:  true,
		Bytes:   []byte("  raw  "),
		Tags:    []string{"  a  ", "  b  "},
		Lookup:  map[string]string{"k": "  v  "},
		Created: created,
	}

	TrimStructStr(in)

	if in.Name != "Aaron" {
		t.Errorf("Name = %q, want %q", in.Name, "Aaron")
	}
	if in.Count != 42 || in.Ratio != 1.5 || !in.Active {
		t.Errorf("scalar fields changed: %+v", in)
	}
	if !in.Created.Equal(created) {
		t.Errorf("Created = %v, want %v", in.Created, created)
	}
	// Slices and maps are not walked; their strings keep their whitespace.
	if string(in.Bytes) != "  raw  " {
		t.Errorf("Bytes = %q, want it untouched", in.Bytes)
	}
	if in.Tags[0] != "  a  " || in.Tags[1] != "  b  " {
		t.Errorf("Tags = %q, want them untouched", in.Tags)
	}
	if in.Lookup["k"] != "  v  " {
		t.Errorf("Lookup[k] = %q, want it untouched", in.Lookup["k"])
	}
}

// Unexported fields cannot be set through reflection, so they are skipped rather
// than causing a panic.
func TestTrimStructStrSkipsUnexportedFields(t *testing.T) {
	type payload struct {
		Name   string
		secret string
	}

	in := &payload{Name: "  Aaron  ", secret: "  hidden  "}

	TrimStructStr(in)

	if in.Name != "Aaron" {
		t.Errorf("Name = %q, want %q", in.Name, "Aaron")
	}
	if in.secret != "  hidden  " {
		t.Errorf("secret = %q, want it untouched", in.secret)
	}
}

// A struct whose fields are all unexported (time.Time is the common one) must
// not panic when it is recursed into.
func TestTrimStructStrEmbeddedTimeIsSafe(t *testing.T) {
	type payload struct {
		Name      string
		CreatedAt time.Time
		UpdatedAt *time.Time
	}

	now := time.Now()
	in := &payload{Name: "  Aaron  ", CreatedAt: now, UpdatedAt: &now}

	TrimStructStr(in)

	if in.Name != "Aaron" {
		t.Errorf("Name = %q, want %q", in.Name, "Aaron")
	}
	if !in.CreatedAt.Equal(now) || !in.UpdatedAt.Equal(now) {
		t.Errorf("timestamps changed: %v / %v, want %v", in.CreatedAt, in.UpdatedAt, now)
	}
}

// An embedded field is named after its type, so an embedded *exported* type is
// settable and gets walked like any other nested struct.
func TestTrimStructStrEmbeddedExportedStruct(t *testing.T) {
	type payload struct {
		Base
		Name string
	}

	in := &payload{Base: Base{ID: "  id-1  "}, Name: "  Aaron  "}

	TrimStructStr(in)

	if in.ID != "id-1" {
		t.Errorf("ID = %q, want %q", in.ID, "id-1")
	}
	if in.Name != "Aaron" {
		t.Errorf("Name = %q, want %q", in.Name, "Aaron")
	}
}

// Base is embedded by the test above; it must be exported for that case.
type Base struct{ ID string }

// An embedded *unexported* type is an unexported field as far as reflection is
// concerned, so the CanSet guard skips the whole embedded struct — its exported
// fields are never reached.
func TestTrimStructStrEmbeddedUnexportedStructIsSkipped(t *testing.T) {
	type base struct{ ID string }
	type payload struct {
		base
		Name string
	}

	in := &payload{base: base{ID: "  id-1  "}, Name: "  Aaron  "}

	TrimStructStr(in)

	if in.Name != "Aaron" {
		t.Errorf("Name = %q, want %q", in.Name, "Aaron")
	}
	if in.ID != "  id-1  " {
		t.Errorf("ID = %q, want it left untrimmed by the unexported-field guard", in.ID)
	}
}

// Anything that is not a pointer to a struct comes back untouched.
func TestTrimStructStrIgnoresNonStructPointers(t *testing.T) {
	type payload struct{ Name string }

	t.Run("struct value is not addressable", func(t *testing.T) {
		in := payload{Name: "  Aaron  "}

		got := TrimStructStr(in)

		out, ok := got.(payload)
		if !ok {
			t.Fatalf("TrimStructStr() returned %T, want payload", got)
		}
		if out.Name != "  Aaron  " {
			t.Errorf("Name = %q, want it untouched", out.Name)
		}
	})

	t.Run("pointer to a non-struct", func(t *testing.T) {
		in := ptr("  Aaron  ")

		got := TrimStructStr(in)

		if got.(*string) != in || *in != "  Aaron  " {
			t.Errorf("value = %q, want it untouched", *in)
		}
	})

	t.Run("plain string", func(t *testing.T) {
		if got := TrimStructStr("  Aaron  "); got != "  Aaron  " {
			t.Errorf("TrimStructStr() = %q, want it untouched", got)
		}
	})

	t.Run("nil interface", func(t *testing.T) {
		if got := TrimStructStr(nil); got != nil {
			t.Errorf("TrimStructStr(nil) = %v, want nil", got)
		}
	})

	t.Run("typed nil pointer", func(t *testing.T) {
		var in *payload

		got := TrimStructStr(in)

		if got.(*payload) != nil {
			t.Errorf("TrimStructStr() = %v, want nil", got)
		}
	})

	t.Run("slice", func(t *testing.T) {
		in := []string{"  a  "}

		TrimStructStr(in)

		if in[0] != "  a  " {
			t.Errorf("element = %q, want it untouched", in[0])
		}
	})
}

func TestTrimStructStrEmptyStruct(t *testing.T) {
	type payload struct{}

	if got := TrimStructStr(&payload{}); got == nil {
		t.Error("TrimStructStr() = nil, want the argument back")
	}
}

// Trimming twice is the same as trimming once.
func TestTrimStructStrIsIdempotent(t *testing.T) {
	type payload struct {
		Name string
		Nick *string
	}

	in := &payload{Name: "  Aaron  ", Nick: ptr("  ")}

	TrimStructStr(in)
	first := *in
	TrimStructStr(in)

	if in.Name != first.Name {
		t.Errorf("Name = %q after a second pass, want %q", in.Name, first.Name)
	}
	if in.Nick != nil {
		t.Errorf("Nick = %q, want it to stay nil", *in.Nick)
	}
}
