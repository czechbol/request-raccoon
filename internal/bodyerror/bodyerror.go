// Package bodyerror classifies failures to read a request body.
package bodyerror

import (
	"errors"
	"net/http"
	"os"
)

// Status returns the HTTP status describing why reading a request body failed.
func Status(err error) int {
	var tooLarge *http.MaxBytesError
	switch {
	case errors.As(err, &tooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, os.ErrDeadlineExceeded):
		return http.StatusRequestTimeout
	default:
		return http.StatusBadRequest
	}
}
