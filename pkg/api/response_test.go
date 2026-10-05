package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestResponseErrors(t *testing.T) {
	for _, parser := range []struct {
		name  string
		parse func(*http.Response, interface{}) error
	}{
		{"parseResponse", parseResponse},
		{"parseResponsePreservingRaw", func(resp *http.Response, value interface{}) error {
			raw, err := parseResponsePreservingRaw(resp, value)
			if raw != nil {
				t.Errorf("error response returned successful raw payload: %s", raw)
			}
			return err
		}},
	} {
		t.Run(parser.name, func(t *testing.T) {
			for _, tt := range []struct {
				name, body, message string
			}{
				{"detail", `{"detail":"tier unavailable","message":"secondary"}`, "tier unavailable"},
				{"message", `{"message":"checkout unavailable"}`, "checkout unavailable"},
				{"blank detail", `{"detail":"  ","message":"checkout unavailable"}`, "checkout unavailable"},
				{"field validation", `{"autorenewal_enabled":["No default payment method set"],"display_name":["Too long"]}`, `{"autorenewal_enabled":["No default payment method set"],"display_name":["Too long"]}`},
				{"nested validation", `{"expected_quote":{"unit_price":["A valid number is required."]}}`, `{"expected_quote":{"unit_price":["A valid number is required."]}}`},
				{"raw fallback", "upstream unavailable", "upstream unavailable"},
				{"unknown JSON", `{"error":"upstream unavailable"}`, `{"error":"upstream unavailable"}`},
				{"structured detail", `{"detail":["invalid plan"]}`, `{"detail":["invalid plan"]}`},
				{"empty body", " \n", "Bad Request"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					resp := &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(tt.body))}
					err := parser.parse(resp, nil)
					want := fmt.Sprintf("API error (400): %s", tt.message)
					if err == nil || err.Error() != want {
						t.Fatalf("error = %v, want %q", err, want)
					}
				})
			}
		})
	}
}

func TestResponseSuccessCompatibility(t *testing.T) {
	const body = "{\n  \"pool_id\": \"pool::abc\", \"unknown\": 9007199254740993\n}"
	for _, preserveRaw := range []bool{false, true} {
		for _, decode := range []bool{false, true} {
			t.Run(fmt.Sprintf("raw=%t/decode=%t", preserveRaw, decode), func(t *testing.T) {
				resp := &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
				var pool ResourcePool
				var target interface{}
				if decode {
					target = &pool
				}
				var err error
				if preserveRaw {
					var raw []byte
					raw, err = parseResponsePreservingRaw(resp, target)
					if string(raw) != body {
						t.Fatalf("raw response changed: %s", raw)
					}
				} else {
					err = parseResponse(resp, target)
				}
				if err != nil || (decode && pool.PoolID != "pool::abc") {
					t.Fatalf("pool = %+v, error = %v", pool, err)
				}
			})
		}
	}
}
