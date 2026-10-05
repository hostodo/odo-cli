package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hostodo/odo-cli/v2/pkg/terminaltext"
)

type capacityCountingBody struct {
	remaining int
	read      int
	closed    bool
}

func (body *capacityCountingBody) Read(p []byte) (int, error) {
	if body.remaining == 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > body.remaining {
		n = body.remaining
	}
	for i := 0; i < n; i++ {
		p[i] = 'x'
	}
	body.remaining -= n
	body.read += n
	return n, nil
}
func (body *capacityCountingBody) Close() error { body.closed = true; return nil }

func TestCapacityErrorBodiesAreBounded(t *testing.T) {
	for _, preserveRaw := range []bool{false, true} {
		t.Run(fmt.Sprintf("raw=%t", preserveRaw), func(t *testing.T) {
			body := &capacityCountingBody{remaining: 32 * 1024 * 1024}
			resp := &http.Response{StatusCode: http.StatusBadGateway, Body: body}
			var err error
			if preserveRaw {
				_, err = parseResponsePreservingRaw(resp, new(ResourcePoolCheckoutResponse))
			} else {
				err = parseResponse(resp, new(User))
			}
			if err == nil || body.read > maxAPIErrorBody || !body.closed || len(err.Error()) > maxAPIErrorBody+100 {
				t.Fatalf("read=%d closed=%t error length=%d", body.read, body.closed, len(fmt.Sprint(err)))
			}
		})
	}
}

func TestCapacityErrorTextSanitized(t *testing.T) {
	attack := "unsafe\x1b[31mred\x1b[0m\x1b]52;c;ZXZpbA==\a\r\n\x00\x7f\u202e"
	detail, _ := json.Marshal(map[string]string{"detail": attack})
	message, _ := json.Marshal(map[string]string{"message": attack})
	for _, body := range []string{string(detail), string(message), attack} {
		_, err := parseResponsePreservingRaw(&http.Response{StatusCode: 400, Body: io.NopCloser(strings.NewReader(body))}, new(ResourcePoolCheckoutResponse))
		if err == nil || terminaltext.Clean(err.Error()) != err.Error() || strings.Contains(err.Error(), "ZXZpbA") {
			t.Fatalf("unsafe error: %v", err)
		}
	}
}
