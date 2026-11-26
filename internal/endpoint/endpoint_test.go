package endpoint

import (
	"net/http"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	got, err := Parse(`[
		{"pattern":"POST /hook","status":202,"headers":{"Content-Type":"text/plain"},"body":"ok"},
		{"pattern":"GET /users/{id}"}
	]`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d endpoints, want 2", len(got))
	}
	first := got[0]
	if first.Pattern != "POST /hook" || first.Status != http.StatusAccepted || first.Body != "ok" ||
		first.Headers["Content-Type"] != "text/plain" {
		t.Errorf("unexpected first endpoint: %+v", first)
	}
	if got[1].Status != http.StatusOK || got[1].Body != "" {
		t.Errorf("unexpected defaults: %+v", got[1])
	}
}

func TestParse_None(t *testing.T) {
	for _, raw := range []string{"", "  \n", "[]"} {
		got, err := Parse(raw)
		if err != nil || len(got) != 0 {
			t.Errorf("Parse(%q) = %v, %v; want no endpoints", raw, got, err)
		}
	}
}

func TestParse_Allowed(t *testing.T) {
	for _, raw := range []string{
		`[{"pattern":"GET /{$}"}]`,
		`[{"pattern":"/health/live"}]`,
		`[{"pattern":"/a","status":204}]`,
		`[{"pattern":"GET /a"},{"pattern":"POST /a"}]`,
		`[{"pattern":"/a","status":599}]`,
	} {
		if _, err := Parse(raw); err != nil {
			t.Errorf("Parse(%s): %v", raw, err)
		}
	}
}

func TestParse_Invalid(t *testing.T) {
	tests := map[string]struct{ raw, want string }{
		"not JSON":          {`nope`, "invalid JSON"},
		"not an array":      {`{"pattern":"/a"}`, "invalid JSON"},
		"trailing data":     {`[] []`, "after the array"},
		"unknown field":     {`[{"pattern":"/a","stauts":201}]`, "stauts"},
		"missing pattern":   {`[{"status":201}]`, "endpoint 1: pattern is required"},
		"bad pattern":       {`[{"pattern":"nope"}]`, "endpoint 1"},
		"duplicate":         {`[{"pattern":"/a"},{"pattern":"/a"}]`, "endpoint 2"},
		"default route":     {`[{"pattern":"/"}]`, "endpoint 1"},
		"health":            {`[{"pattern":"/health"}]`, "reserved"},
		"health method":     {`[{"pattern":"GET /health"}]`, "reserved"},
		"health host":       {`[{"pattern":"example.com/health"}]`, "reserved"},
		"health conflict":   {`[{"pattern":"GET /{name}"}]`, "endpoint 1"},
		"status low":        {`[{"pattern":"/a","status":101}]`, "status"},
		"status high":       {`[{"pattern":"/a","status":600}]`, "status"},
		"body with 204":     {`[{"pattern":"/a","status":204,"body":"x"}]`, "body"},
		"body with 304":     {`[{"pattern":"/a","status":304,"body":"x"}]`, "body"},
		"header name":       {`[{"pattern":"/a","headers":{"Bad Name":"x"}}]`, "header"},
		"empty header name": {`[{"pattern":"/a","headers":{"":"x"}}]`, "header"},
		"header value":      {`[{"pattern":"/a","headers":{"X-A":"a\r\nb"}}]`, "header"},
		"content length":    {`[{"pattern":"/a","headers":{"content-length":"1"}}]`, "header"},
		"transfer encoding": {`[{"pattern":"/a","headers":{"Transfer-Encoding":"chunked"}}]`, "header"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := Parse(tt.raw)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("Parse(%s) error = %v, want it to contain %q", tt.raw, err, tt.want)
			}
		})
	}
}
