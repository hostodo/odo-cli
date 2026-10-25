package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProtectionAndPowerRequests(t *testing.T) {
	injectToken(t)
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/protection/") {
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			enabled := body["enabled"] == true
			if enabled && body["confirmation"] != nil {
				t.Error("enable must omit confirmation")
			}
			if !enabled && body["confirmation"] != "UNPROTECT ins::abc" {
				t.Error("missing canonical confirmation")
			}
			writeJSON(w, 200, InstanceDetailResponse{Instance: Instance{InstanceID: "ins::abc", ProtectionEnabled: enabled}})
			return
		}
		writeJSON(w, 200, map[string]string{})
	}))
	defer srv.Close()
	client := newTestClient(t, srv.URL)
	for _, enabled := range []bool{true, false} {
		confirmation := ""
		if !enabled {
			confirmation = "UNPROTECT ins::abc"
		}
		inst, err := client.SetInstanceProtection("ins::abc", enabled, confirmation)
		if err != nil {
			t.Fatal(err)
		}
		if inst.ProtectionEnabled != enabled {
			t.Fatalf("wrong state: %+v", inst)
		}
	}
	if err := client.StartInstance("ins::abc"); err != nil {
		t.Fatal(err)
	}
	for _, force := range []bool{false, true} {
		if err := client.StopInstance("ins::abc", force); err != nil {
			t.Fatal(err)
		}
		if err := client.RebootInstance("ins::abc", force); err != nil {
			t.Fatal(err)
		}
	}
	if len(calls) != 7 || calls[0] != "PATCH /client/instances/ins::abc/protection/" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestStructuredAndLegacyErrors(t *testing.T) {
	for _, tc := range []struct {
		status             int
		body, code, detail string
	}{
		{423, `{"code":"protected","detail":"Protected VM","instance_id":"ins::abc","operation":"rename","protection_enabled":true}`, "protected", "Protected VM"},
		{409, `{"code":"busy","detail":"Busy VM"}`, "busy", "Busy VM"},
		{403, `{"code":"forbidden_scope","detail":"Scope denied"}`, "forbidden_scope", "Scope denied"},
		{400, `{"detail":"Legacy detail"}`, "", "Legacy detail"},
		{400, `{"code":400,"detail":"Numeric code"}`, "400", "Numeric code"},
		{400, `{"message":"Old message"}`, "", "Old message"},
	} {
		t.Run(tc.code+tc.detail, func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}
			err := parseResponse(resp, nil)
			var apiErr *ErrorResponse
			if !errors.As(err, &apiErr) || apiErr.Code != tc.code || apiErr.Detail != tc.detail || apiErr.HTTPStatus != tc.status {
				t.Fatalf("lost error data: %#v", err)
			}
			if tc.code == "protected" && (apiErr.InstanceID != "ins::abc" || apiErr.Operation != "rename" || apiErr.ProtectionEnabled == nil || !*apiErr.ProtectionEnabled) {
				t.Fatalf("lost protection metadata: %+v", apiErr)
			}
			if tc.code == "forbidden_scope" && !strings.Contains(err.Error(), "odo login --protection") {
				t.Fatal("missing reauthorization advice")
			}
		})
	}
}

func TestProtectionFailuresNeverRetryOrLogin(t *testing.T) {
	injectToken(t)
	for _, status := range []int{403, 409, 423} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				writeJSON(w, status, map[string]interface{}{"code": "forbidden_scope", "detail": "Denied"})
			}))
			defer srv.Close()
			client := newTestClient(t, srv.URL)
			if _, err := client.SetInstanceProtection("ins::abc", false, "UNPROTECT ins::abc"); err == nil {
				t.Fatal("expected error")
			}
			if calls != 1 {
				t.Fatalf("unexpected retries: %d", calls)
			}
			if _, err := client.GetInstance("ins::abc"); err == nil {
				t.Fatal("GET must not decode errors as an empty VM")
			}
		})
	}
}
