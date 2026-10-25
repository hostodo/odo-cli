package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestProtectionCommandProcess(t *testing.T) {
	if os.Getenv("ODO_TEST_COMMAND") == "" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("ODO_TEST_COMMAND")), &args); err != nil {
		panic(err)
	}
	rootCmd.SetArgs(args)
	if err := Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestProtectionCommands(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		args                   []string
		protected, fail, patch bool
		input                  string
	}{
		{"protect hostname", []string{"instances", "protect", "mybox"}, false, false, true, ""},
		{"unprotect id yes", []string{"instances", "unprotect", "ins::abc", "--yes"}, true, false, true, ""},
		{"unprotect hostname phrase", []string{"instances", "unprotect", "mybox"}, true, false, true, "UNPROTECT ins::abc\n"},
		{"unprotect incorrect", []string{"instances", "unprotect", "mybox"}, true, true, false, "yes\n"},
		{"rename protected", []string{"instances", "rename", "mybox", "newname"}, true, true, false, ""},
		{"reinstall yes protected", []string{"instances", "reinstall", "mybox", "--yes"}, true, true, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			if err := os.MkdirAll(filepath.Join(home, ".odo"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(home, ".odo", "token"), []byte("test-token"), 0600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			patches := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Content-Type", "application/json")
				instance := map[string]interface{}{"instance_id": "ins::abc", "hostname": "mybox", "protection_enabled": tc.protected}
				if r.Method == "GET" && r.URL.Path == "/client/instances/" {
					json.NewEncoder(w).Encode(map[string]interface{}{"results": []interface{}{instance}})
					return
				}
				if r.Method == "PATCH" && r.URL.Path == "/client/instances/ins::abc/protection/" {
					patches++
					var body map[string]interface{}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if body["enabled"] == false && body["confirmation"] != "UNPROTECT ins::abc" {
						t.Error("wrong confirmation")
					}
					instance["protection_enabled"] = body["enabled"]
					json.NewEncoder(w).Encode(map[string]interface{}{"instance": instance})
					return
				}
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(500)
			}))
			defer srv.Close()
			args := append([]string{"--api-url", srv.URL, "--force-http"}, tc.args...)
			encoded, _ := json.Marshal(args)
			cmd := exec.Command(os.Args[0], "-test.run=^TestProtectionCommandProcess$")
			cmd.Env = append(os.Environ(), "HOME="+home, "ODO_TEST_COMMAND="+string(encoded), "DBUS_SESSION_BUS_ADDRESS=unix:path=/nonexistent")
			cmd.Stdin = strings.NewReader(tc.input)
			output, err := cmd.CombinedOutput()
			if (err != nil) != tc.fail {
				t.Fatalf("exit=%v output=%s", err, output)
			}
			wantCalls := 1
			if tc.patch {
				wantCalls++
			}
			if calls != wantCalls || (patches == 1) != tc.patch {
				t.Fatalf("calls=%d patches=%d output=%s", calls, patches, output)
			}
		})
	}
}

func TestLoginProtectionFlagOnBothPaths(t *testing.T) {
	for _, path := range [][]string{{"login"}, {"auth", "login"}} {
		command, _, err := rootCmd.Find(path)
		if err != nil || command.Flags().Lookup("protection") == nil {
			t.Fatalf("missing --protection on %v", path)
		}
	}
}

func TestLoginForwardsScopesOnlyWithOptIn(t *testing.T) {
	for _, args := range [][]string{{"login"}, {"login", "--protection"}, {"auth", "login", "--protection"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var body map[string]json.RawMessage
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/oauth/device/authorize" {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				// Stop before polling/browser launch: this test covers the authorization request.
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"detail":"test stops after authorize request"}`))
			}))
			defer srv.Close()
			encoded, _ := json.Marshal(append([]string{"--api-url", srv.URL, "--force-http"}, args...))
			cmd := exec.Command(os.Args[0], "-test.run=^TestProtectionCommandProcess$")
			cmd.Env = append(os.Environ(), "HOME="+t.TempDir(), "ODO_TEST_COMMAND="+string(encoded), "DBUS_SESSION_BUS_ADDRESS=unix:path=/nonexistent")
			output, err := cmd.CombinedOutput()
			if err == nil || body == nil {
				t.Fatalf("expected test stop after request, err=%v output=%s", err, output)
			}
			if args[len(args)-1] != "--protection" {
				if _, exists := body["scopes"]; exists {
					t.Fatal("default login must omit scopes")
				}
				return
			}
			var scopes []string
			if err := json.Unmarshal(body["scopes"], &scopes); err != nil {
				t.Fatal(err)
			}
			if strings.Join(scopes, " ") != "vms:read vms:console vms:power account:read vms:protection" {
				t.Fatalf("lost legacy or protection scopes: %v", scopes)
			}
		})
	}
}
