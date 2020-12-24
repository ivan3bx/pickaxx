package minecraft

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPortOpenSuccess(t *testing.T) {

	// set up a dummy server on a dummy port
	ts := httptest.NewServer(http.DefaultServeMux)
	defer ts.Close()

	url, _ := url.Parse(ts.URL)
	testPort, _ := strconv.Atoi(url.Port())

	tests := []struct {
		name     string
		port     int
		expected error
	}{
		{
			name:     "successful check",
			port:     testPort,
			expected: nil,
		},
		{
			name:     "unsuccessful check",
			port:     -99,
			expected: ErrNoResponse,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				time.Sleep(time.Millisecond * 20)
				cancel()
			}()

			assert.Equal(t, tc.expected, checkPort(ctx, tc.port, 0, time.Millisecond*5))
		})
	}
}
