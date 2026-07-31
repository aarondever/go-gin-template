package pagination

import (
	"encoding/json"
	"testing"
)

func TestGetPage(t *testing.T) {
	tests := []struct {
		name string
		page int
		want int
	}{
		{name: "zero defaults to one", page: 0, want: 1},
		{name: "negative defaults to one", page: -5, want: 1},
		{name: "one", page: 1, want: 1},
		{name: "arbitrary page", page: 42, want: 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pagination{Page: tt.page}

			if got := p.GetPage(); got != tt.want {
				t.Errorf("GetPage() = %d, want %d", got, tt.want)
			}
			// GetPage normalises the field in place.
			if p.Page != tt.want {
				t.Errorf("p.Page = %d, want %d", p.Page, tt.want)
			}
		})
	}
}

func TestLimit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		want     int
	}{
		{name: "zero defaults to ten", pageSize: 0, want: 10},
		{name: "negative defaults to ten", pageSize: -1, want: 10},
		{name: "one", pageSize: 1, want: 1},
		{name: "within range", pageSize: 50, want: 50},
		{name: "at the cap", pageSize: 100, want: 100},
		{name: "just over the cap", pageSize: 101, want: 100},
		{name: "far over the cap", pageSize: 10000, want: 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pagination{PageSize: tt.pageSize}

			if got := p.Limit(); got != tt.want {
				t.Errorf("Limit() = %d, want %d", got, tt.want)
			}
			// Limit clamps the field in place.
			if p.PageSize != tt.want {
				t.Errorf("p.PageSize = %d, want %d", p.PageSize, tt.want)
			}
		})
	}
}

func TestOffset(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		want     int
	}{
		{name: "first page", page: 1, pageSize: 10, want: 0},
		{name: "second page", page: 2, pageSize: 10, want: 10},
		{name: "third page of twenty", page: 3, pageSize: 20, want: 40},
		{name: "zero page is treated as the first", page: 0, pageSize: 25, want: 0},
		{name: "negative page is treated as the first", page: -3, pageSize: 25, want: 0},
		{name: "zero size uses the default", page: 3, pageSize: 0, want: 20},
		{name: "negative size uses the default", page: 2, pageSize: -7, want: 10},
		{name: "oversized page size is capped", page: 2, pageSize: 500, want: 100},
		{name: "unset pagination", page: 0, pageSize: 0, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pagination{Page: tt.page, PageSize: tt.pageSize}

			if got := p.Offset(); got != tt.want {
				t.Errorf("Offset() = %d, want %d", got, tt.want)
			}
		})
	}
}

// The normalised values must stick, so a second call gives the same answer.
func TestRepeatedCallsAreStable(t *testing.T) {
	p := &Pagination{Page: -1, PageSize: 999}

	first := struct{ page, limit, offset int }{p.GetPage(), p.Limit(), p.Offset()}
	second := struct{ page, limit, offset int }{p.GetPage(), p.Limit(), p.Offset()}

	if first != second {
		t.Errorf("second round = %+v, want %+v", second, first)
	}
	if first.page != 1 || first.limit != 100 || first.offset != 0 {
		t.Errorf("got page=%d limit=%d offset=%d, want 1/100/0",
			first.page, first.limit, first.offset)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	p := Pagination{Page: 2, PageSize: 25, Total: 137}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var got Pagination
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got != p {
		t.Errorf("round trip = %+v, want %+v", got, p)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"page", "page_size", "total"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("marshalled JSON %s is missing %q", data, key)
		}
	}
}
