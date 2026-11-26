// Package endpoint defines custom endpoints that answer with a configured status, headers and body.
package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	healthPath = "/health"
	minStatus  = 200
	maxStatus  = 599
	// headerSymbols are the characters besides letters and digits allowed in a header name.
	headerSymbols = "!#$%&'*+-.^_`|~"
)

var (
	errTrailingData   = errors.New("unexpected data after the array")
	errNoPattern      = errors.New("pattern is required")
	errBadPattern     = errors.New("invalid or conflicting pattern")
	errReservedPath   = errors.New(healthPath + " is reserved")
	errBadStatus      = errors.New("status must be between 200 and 599")
	errBodyNotAllowed = errors.New("this status does not allow a body")
	errBadHeader      = errors.New("invalid or unsupported header")
)

// Endpoint is a custom response for the requests matching a pattern.
type Endpoint struct {
	// Pattern is an http.ServeMux pattern, such as "POST /hook" or "/users/{id}".
	Pattern string            `json:"pattern"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// Parse reads a JSON array of endpoints. A blank string defines none.
// The returned endpoints can be registered next to the /health and / routes.
func Parse(raw string) ([]Endpoint, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var endpoints []Endpoint
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&endpoints); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errTrailingData
	}

	// Registering on a scratch router finds bad and conflicting patterns.
	mux := http.NewServeMux()
	mux.Handle(healthPath, http.NotFoundHandler())
	mux.Handle("/", http.NotFoundHandler())

	for i := range endpoints {
		if endpoints[i].Status == 0 {
			endpoints[i].Status = http.StatusOK
		}
		if err := validate(mux, &endpoints[i]); err != nil {
			return nil, fmt.Errorf("endpoint %d: %w", i+1, err)
		}
	}
	return endpoints, nil
}

func validate(mux *http.ServeMux, e *Endpoint) error {
	if e.Pattern == "" {
		return errNoPattern
	}
	// The path starts at the first slash, after the optional method and host.
	if slash := strings.Index(e.Pattern, "/"); slash >= 0 && e.Pattern[slash:] == healthPath {
		return errReservedPath
	}
	if e.Status < minStatus || e.Status > maxStatus {
		return errBadStatus
	}
	if e.Body != "" && (e.Status == http.StatusNoContent || e.Status == http.StatusNotModified) {
		return errBodyNotAllowed
	}
	for name, value := range e.Headers {
		if !validHeader(name, value) {
			return fmt.Errorf("%w: %q", errBadHeader, name)
		}
	}
	return register(mux, e.Pattern)
}

// register turns the panic of http.ServeMux for a rejected pattern into an error.
func register(mux *http.ServeMux, pattern string) (err error) {
	defer func() {
		if reason := recover(); reason != nil {
			err = fmt.Errorf("%w %q: %v", errBadPattern, pattern, reason)
		}
	}()
	mux.Handle(pattern, http.NotFoundHandler())
	return nil
}

func validHeader(name, value string) bool {
	switch http.CanonicalHeaderKey(name) {
	// The server sets these from the body.
	case "", "Content-Length", "Transfer-Encoding":
		return false
	}
	invalidNameRune := func(r rune) bool {
		isAlphanumeric := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
		return !isAlphanumeric && !strings.ContainsRune(headerSymbols, r)
	}
	return strings.IndexFunc(name, invalidNameRune) < 0 && !strings.ContainsAny(value, "\r\n\x00")
}
