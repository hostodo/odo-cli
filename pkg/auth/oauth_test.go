package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestDeviceFlowScopesAreExplicitOptIn(t *testing.T) {
	for _, scopes := range [][]string{nil, {"vms:protection"}, {"vms:read", "vms:console", "vms:power", "account:read", "vms:protection"}} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/oauth/device/authorize" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			var body map[string]json.RawMessage
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			var got []string
			if raw, exists := body["scopes"]; exists {
				if len(scopes) == 0 {
					t.Error("default login must omit scopes")
				}
				if err := json.Unmarshal(raw, &got); err != nil {
					t.Error(err)
				}
			}
			if !reflect.DeepEqual(got, scopes) {
				t.Errorf("scopes=%v want %v", got, scopes)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"device_code":"test","user_code":"test"}`))
		}))
		_, err := NewDeviceFlowClient(srv.URL).InitiateDeviceFlow("device", "id", scopes...)
		srv.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
}
