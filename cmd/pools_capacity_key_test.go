package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/zalando/go-keyring"
)

func TestPoolRetryLegacyNamespace(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(fmt.Sprintf("fallback=%t", fallback), func(t *testing.T) {
			client := poolTestClient(t, func(http.ResponseWriter, *http.Request) {
				t.Error("unexpected checkout request")
			})
			if fallback {
				keyring.MockInitWithError(fmt.Errorf("keychain unavailable"))
			}
			path, _ := config.GetConfigPath()
			dir := filepath.Dir(path)
			legacyDir := filepath.Join(dir, "capacity-checkouts")
			if err := os.MkdirAll(legacyDir, 0700); err != nil {
				t.Fatal(err)
			}
			req, _ := buildPoolCheckoutRequest(42, "monthly", "credit", "", "", "legacy-key", false)
			legacyPath := filepath.Join(legacyDir, poolRetryHash([]byte(req.IdempotencyKey))+".json")
			legacy := []byte(`{"version":1,"expected_quote":{"mode":"purchase","existing_pool_id":null,"unit_price":"1","recurring_amount":"1","amount_due_after_credit":"1"}}`)
			if err := os.WriteFile(legacyPath, legacy, 0600); err != nil {
				t.Fatal(err)
			}
			if fallback {
				if _, err := poolRetrySecret(dir); err != nil {
					t.Fatal(err)
				}
				// Restore the separate API credential; the published retry key
				// must continue using its originally selected file fallback.
				keyring.MockInit()
				if err := keyring.Set("odo-cli", "access-token", "capacity-test-token"); err != nil {
					t.Fatal(err)
				}
			}
			cache, err := newPoolRetryCache(client, req)
			if err != nil {
				t.Fatal(err)
			}
			if filepath.Dir(cache.path) != filepath.Join(dir, "capacity-checkouts-v2") {
				t.Fatalf("wrong cache namespace: %s", cache.path)
			}
			if quote, err := cache.load(); err != nil || quote != nil {
				t.Fatalf("legacy record replayed: quote=%+v err=%v", quote, err)
			}
			cache.confirmation = "CONFIRM"
			if err := cache.save(&api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "5", RecurringAmount: "5", AmountDueAfterCredit: "5"}); err != nil {
				t.Fatal(err)
			}
			if quote, err := cache.load(); err != nil || quote == nil || quote.UnitPrice.String() != "5" {
				t.Fatalf("new authenticated record unavailable: quote=%+v err=%v", quote, err)
			}
			preserved, err := os.ReadFile(legacyPath)
			if err != nil || !bytes.Equal(preserved, legacy) {
				t.Fatalf("legacy record changed: %q, %v", preserved, err)
			}
			// Even copying a legacy record into the new namespace cannot bless it.
			if err := os.WriteFile(cache.path, legacy, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := cache.load(); err == nil {
				t.Fatal("accepted unauthenticated legacy record in v2 namespace")
			}
		})
	}
}

func TestPoolRetryMissingV2KeyMarker(t *testing.T) {
	for _, populated := range []bool{false, true} {
		t.Run(fmt.Sprintf("populated=%t", populated), func(t *testing.T) {
			calls := 0
			client := poolTestClient(t, func(http.ResponseWriter, *http.Request) { calls++ })
			req, _ := buildPoolCheckoutRequest(42, "monthly", "credit", "", "", "missing-marker", false)
			cache, err := newPoolRetryCache(client, req)
			if err != nil {
				t.Fatal(err)
			}
			if populated {
				cache.confirmation = "CONFIRM"
				if err := cache.save(&api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "5", RecurringAmount: "5", AmountDueAfterCredit: "5"}); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Mkdir(filepath.Dir(cache.path), 0700); err != nil {
				t.Fatal(err)
			}
			before, _ := os.ReadFile(cache.path)
			keyPath := filepath.Join(filepath.Dir(filepath.Dir(cache.path)), "capacity-retry-key")
			if err := os.Remove(keyPath); err != nil {
				t.Fatal(err)
			}
			cmd, out, _ := poolTestCommand()
			err = runPoolCheckout(cmd, client, req, true, false)
			if err == nil || !strings.Contains(err.Error(), "key source missing") || calls != 0 || out.Len() != 0 {
				t.Fatalf("missing marker: err=%v calls=%d stdout=%s", err, calls, out)
			}
			if _, err := os.Lstat(keyPath); !os.IsNotExist(err) {
				t.Fatalf("key marker recreated: %v", err)
			}
			after, _ := os.ReadFile(cache.path)
			if !bytes.Equal(before, after) {
				t.Fatal("changed existing retry record")
			}
		})
	}
}

func poolRetryKeyTestDir(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	keyring.MockInitWithError(fmt.Errorf("keychain unavailable"))
	t.Cleanup(keyring.MockInit)
	if err := config.EnsureConfigDir(); err != nil {
		t.Fatal(err)
	}
	path, _ := config.GetConfigPath()
	return filepath.Dir(path)
}

func TestPoolRetryInterruptedOldLock(t *testing.T) {
	dir := poolRetryKeyTestDir(t)
	oldLock := filepath.Join(dir, "capacity-retry-key.lock")
	if err := os.Mkdir(oldLock, 0700); err != nil {
		t.Fatal(err)
	}
	first, err := poolRetrySecret(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := poolRetrySecret(dir)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatalf("key changed: %v", err)
	}
	if info, err := os.Stat(oldLock); err != nil || !info.IsDir() {
		t.Fatalf("old lock artifact changed: %v", err)
	}
}

// A subprocess exercises real OS lock ownership and crash cleanup on both Unix
// and Windows, without touching the user's keychain. Pipes coordinate startup.
func TestPoolRetryKeyProcess(t *testing.T) {
	mode := os.Getenv("ODO_TEST_RETRY_KEY_PROCESS")
	if mode == "" {
		return
	}
	keyring.MockInitWithError(fmt.Errorf("keychain unavailable"))
	path, err := config.GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if mode == "hold" {
		lock, err := acquirePoolRetryKeyLock(filepath.Join(filepath.Dir(path), "capacity-retry-key.init-lock"))
		if err != nil {
			t.Fatal(err)
		}
		defer lock.Close()
		fmt.Println("ready")
		io.Copy(io.Discard, os.Stdin)
		return
	}
	fmt.Println("ready")
	if _, err := io.ReadFull(os.Stdin, make([]byte, 1)); err != nil {
		t.Fatal(err)
	}
	secret, err := poolRetrySecret(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(poolRetryHash(secret))
}

type poolRetryKeyProcess struct {
	cmd   *exec.Cmd
	lines chan string
	done  chan struct{}
	err   error
}

func startPoolRetryKeyProcess(t *testing.T, mode string) *poolRetryKeyProcess {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestPoolRetryKeyProcess$")
	cmd.Env = append(os.Environ(), "ODO_TEST_RETRY_KEY_PROCESS="+mode)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	p := &poolRetryKeyProcess{cmd: cmd, lines: make(chan string, 32), done: make(chan struct{})}
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			p.lines <- scanner.Text()
		}
		close(p.lines)
		p.err = cmd.Wait()
		close(p.done)
	}()
	t.Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		select {
		case <-p.done:
		case <-time.After(15 * time.Second):
			t.Error("child did not exit")
		}
	})
	if line := p.next(t); line != "ready" {
		t.Fatalf("child startup: %s", line)
	}
	if mode != "hold" {
		if _, err := stdin.Write([]byte{1}); err != nil {
			t.Fatal(err)
		}
		stdin.Close()
	}
	return p
}

func (p *poolRetryKeyProcess) next(t *testing.T) string {
	t.Helper()
	select {
	case line, ok := <-p.lines:
		if !ok {
			t.Fatal("child exited without expected output")
		}
		return line
	case <-time.After(15 * time.Second):
		t.Fatal("child timed out waiting for key initialization")
		return ""
	}
}

func (p *poolRetryKeyProcess) wait(t *testing.T) {
	t.Helper()
	select {
	case <-p.done:
		if p.err != nil {
			t.Fatalf("child failed: %v", p.err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("child did not exit")
	}
}

func TestPoolRetryConcurrentInitialization(t *testing.T) {
	dir := poolRetryKeyTestDir(t)
	lock, err := acquirePoolRetryKeyLock(filepath.Join(dir, "capacity-retry-key.init-lock"))
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	var children []*poolRetryKeyProcess
	for range 6 {
		children = append(children, startPoolRetryKeyProcess(t, "initialize"))
	}
	for _, child := range children {
		select {
		case line := <-child.lines:
			t.Fatalf("initializer did not wait for lock: %s", line)
		case <-time.After(50 * time.Millisecond):
		}
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	var want string
	for _, child := range children {
		got := child.next(t)
		if want == "" {
			want = got
		}
		if len(got) != 64 || got != want {
			t.Fatalf("initializers returned different keys: %q != %q", got, want)
		}
		if line := child.next(t); line != "PASS" {
			t.Fatalf("child failed: %s", line)
		}
		child.wait(t)
	}
	secret, err := poolRetrySecret(dir)
	if err != nil || poolRetryHash(secret) != want {
		t.Fatalf("published key differs from initializers: %v", err)
	}
}

func TestPoolRetryLockReleasedAfterCrash(t *testing.T) {
	dir := poolRetryKeyTestDir(t)
	holder := startPoolRetryKeyProcess(t, "hold")
	waiter := startPoolRetryKeyProcess(t, "initialize")
	select {
	case line := <-waiter.lines:
		t.Fatalf("initializer bypassed live lock: %s", line)
	case <-time.After(150 * time.Millisecond):
	}
	if err := holder.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	got := waiter.next(t)
	if line := waiter.next(t); line != "PASS" {
		t.Fatalf("waiter failed after crash: %s", line)
	}
	waiter.wait(t)
	secret, err := poolRetrySecret(dir)
	if err != nil || got != poolRetryHash(secret) {
		t.Fatalf("could not initialize after crash: %v", err)
	}
}

func TestPoolPurchaseHelpV2Cache(t *testing.T) {
	cmd := newPoolsCheckoutCommand(false)
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Help(); err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"~/.odo/capacity-checkouts-v2", "never replayed or migrated", "authentication key marker fails closed"} {
		if !strings.Contains(out.String(), text) {
			t.Errorf("help missing %q", text)
		}
	}
}
