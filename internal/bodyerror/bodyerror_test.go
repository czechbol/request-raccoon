package bodyerror

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
)

func TestStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "too large", err: &http.MaxBytesError{Limit: 1}, want: http.StatusRequestEntityTooLarge},
		{
			name: "wrapped too large", err: fmt.Errorf("read: %w", &http.MaxBytesError{Limit: 1}),
			want: http.StatusRequestEntityTooLarge,
		},
		{name: "deadline exceeded", err: os.ErrDeadlineExceeded, want: http.StatusRequestTimeout},
		{
			name: "network deadline exceeded", err: &net.OpError{Op: "read", Err: os.ErrDeadlineExceeded},
			want: http.StatusRequestTimeout,
		},
		{name: "truncated body", err: io.ErrUnexpectedEOF, want: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Status(tt.err); got != tt.want {
				t.Errorf("Status(%v): expected %d, got %d", tt.err, tt.want, got)
			}
		})
	}
}
