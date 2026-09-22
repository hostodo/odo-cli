package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/config"
	"github.com/hostodo/odo-cli/v2/pkg/terminaltext"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

func TestPoolStableIdentity(t *testing.T) {
	for _, tt := range []struct {
		name string
		user *api.User
		want string
		fail bool
	}{
		{name: "current user_id", user: &api.User{UserID: 42, ID: 7, Email: "ignored@example.test"}, want: "id:42"},
		{name: "legacy id", user: &api.User{ID: 7}, want: "id:7"},
		{name: "verified email fallback", user: &api.User{Email: "  User@Example.Test ", IsEmailVerified: true}, want: "email:user@example.test"},
		{name: "unverified email rejected", user: &api.User{Email: "user@example.test"}, fail: true},
		{name: "missing identity rejected", user: &api.User{}, fail: true},
		{name: "nil user rejected", user: nil, fail: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := poolStableIdentity(tt.user)
			if tt.fail {
				if err == nil {
					t.Fatalf("identity=%q, want failure", got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("identity=%q err=%v, want %q", got, err, tt.want)
			}
		})
	}
}

func TestPoolCheckoutRequiresHardenedServerConfirmation(t *testing.T) {
	for _, method := range []string{"stripe_checkout", "paypal", "alipay", "crypto", "credit", "saved_card"} {
		for _, phrase := range []string{"", "  ", "PURCHASE\x1b[31m", "PURCHASE\nFORGED"} {
			t.Run(method+fmt.Sprintf("/%q", phrase), func(t *testing.T) {
				quotes, checkouts := 0, 0
				client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					var req api.ResourcePoolCheckoutRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Error(err)
					}
					if !req.QuoteOnly {
						checkouts++
						fmt.Fprint(w, `{}`)
						return
					} // Deliberately permissive old server.
					quotes++
					if req.PaymentMethod != method {
						t.Errorf("quote omitted payment method: %+v", req)
					}
					if phrase == "" {
						// Old servers omit the capability field entirely yet accept checkout.
						fmt.Fprint(w, `{"mode":"purchase","unit_price":"5.00","recurring_amount":"5.00","amount_due_after_credit":"5.00","payment_confirmation":"CHARGE SAVED CARD pm::123 ENDING 4242 FOR 5.00"}`)
						return
					}
					json.NewEncoder(w).Encode(api.ResourcePoolCheckoutResponse{Mode: "purchase", UnitPrice: "5.00", RecurringAmount: "5.00", AmountDueAfterCredit: "5.00", Confirmation: phrase, PaymentConfirmation: "CHARGE SAVED CARD pm::123 ENDING 4242 FOR 5.00"})
				})
				id := ""
				if method == "saved_card" {
					id = "pm::123"
				}
				req, err := buildPoolCheckoutRequest(42, "monthly", method, id, "", "old-server", false)
				if err != nil {
					t.Fatal(err)
				}
				cmd, out, _ := poolTestCommand()
				err = runPoolCheckout(cmd, client, req, true, false)
				if err == nil || !strings.Contains(err.Error(), "confirmation") || quotes != 1 || checkouts != 0 || out.Len() != 0 {
					t.Fatalf("err=%v quotes=%d checkouts=%d stdout=%s", err, quotes, checkouts, out)
				}
			})
		}
	}
}

func TestPoolCheckoutExactSeparateConfirmations(t *testing.T) {
	const generic = "APPROVE Capacity 42 for 5.00"
	const card = "CHARGE SAVED CARD pm::123 ENDING 4242 FOR 5.00"
	for _, input := range []string{generic + "\n" + card + "\n", generic + "\r\n" + card + "\r\n", "PURCHASE\n" + card + "\n", generic + " \n" + card + "\n", generic + "\nPURCHASE\n", generic + "\n"} {
		reader := bufio.NewReader(strings.NewReader(input))
		var out bytes.Buffer
		err := confirmPoolPurchase(reader, &out, false, true, "Purchase?", generic)
		if err == nil {
			err = confirmPoolCard(reader, &out, false, true, "Charge?", card)
		}
		wantOK := input == generic+"\n"+card+"\n" || input == generic+"\r\n"+card+"\r\n"
		if (err == nil) != wantOK {
			t.Errorf("input=%q err=%v", input, err)
		}
	}
	var out bytes.Buffer
	if err := confirmPoolPurchase(strings.NewReader(""), &out, true, false, "Purchase?", generic); err != nil || !strings.Contains(out.String(), generic) {
		t.Fatalf("--yes: %v %s", err, &out)
	}
}

func TestPoolRetryKeyStorageAndFailures(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(fmt.Sprintf("fallback=%t", fallback), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			keyring.MockInit()
			if fallback {
				keyring.MockInitWithError(fmt.Errorf("keychain unavailable"))
			}
			t.Cleanup(keyring.MockInit)
			path, _ := config.GetConfigPath()
			dir := filepath.Dir(path)
			first, err := poolRetrySecret(dir)
			if err != nil || len(first) != 32 {
				t.Fatalf("key length=%d err=%v", len(first), err)
			}
			second, err := poolRetrySecret(dir)
			if err != nil || !bytes.Equal(first, second) {
				t.Fatalf("key rotated: %v", err)
			}
			keyPath := filepath.Join(dir, "capacity-retry-key")
			info, err := os.Stat(keyPath)
			if err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0600) {
				t.Fatalf("key permissions: %v %v", info, err)
			}
			stored, err := os.ReadFile(keyPath)
			if err != nil || strings.HasPrefix(string(stored), "fallback:") != fallback {
				t.Fatalf("unexpected key storage: %v", err)
			}
			if !fallback {
				if string(stored) != "keyring" {
					t.Fatal("secret persisted outside keychain")
				}
				keyring.MockInitWithError(fmt.Errorf("keychain locked"))
			} else if err := os.WriteFile(keyPath, []byte("fallback:invalid"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := poolRetrySecret(dir); err == nil {
				t.Fatal("replaced inaccessible/corrupt key")
			}
			if err := os.Mkdir(filepath.Join(dir, poolRetryCacheDir), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(keyPath); err != nil {
				t.Fatal(err)
			}
			if _, err := poolRetrySecret(dir); err == nil {
				t.Fatal("regenerated missing key for existing cache")
			}
		})
	}
}

func TestPoolRetryAllowsSharedConfigDirectoryPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix mode bits do not represent Windows ACLs")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	keyring.MockInitWithError(fmt.Errorf("keychain unavailable"))
	t.Cleanup(keyring.MockInit)
	if err := config.EnsureConfigDir(); err != nil {
		t.Fatal(err)
	}
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Dir(configPath)
	if err := os.Chmod(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	secret, err := poolRetrySecret(configDir)
	if err != nil {
		t.Fatalf("shared ~/.odo permissions rejected: %v", err)
	}
	cache := &poolRetryCache{
		path:         filepath.Join(configDir, poolRetryCacheDir, "shared-config.json"),
		requestHash:  "request",
		secret:       secret,
		confirmation: "CONFIRM",
	}
	quote := &api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "5", RecurringAmount: "5", AmountDueAfterCredit: "5"}
	if err := cache.save(quote); err != nil {
		t.Fatalf("save with shared ~/.odo permissions: %v", err)
	}
	if loaded, err := cache.load(); err != nil || loaded == nil {
		t.Fatalf("load with shared ~/.odo permissions: quote=%+v err=%v", loaded, err)
	}
	for path, mode := range map[string]os.FileMode{
		filepath.Join(configDir, "capacity-retry-key"): 0600,
		filepath.Dir(cache.path):                       0700,
		cache.path:                                     0600,
	} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != mode {
			t.Fatalf("permissions for %s: info=%v err=%v", path, info, err)
		}
	}
}

func TestPoolRetrySyncScope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := config.EnsureConfigDir(); err != nil {
		t.Fatal(err)
	}
	configPath, err := config.GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	configDir := filepath.Dir(configPath)
	cacheDir := filepath.Join(configDir, poolRetryCacheDir)
	cache := &poolRetryCache{
		path:         filepath.Join(cacheDir, "sync-scope.json"),
		requestHash:  "request",
		secret:       bytes.Repeat([]byte{1}, 32),
		confirmation: "CONFIRM",
	}
	var synced []string
	oldSync := poolRetrySyncDir
	poolRetrySyncDir = func(path string) error {
		synced = append(synced, path)
		return nil
	}
	t.Cleanup(func() { poolRetrySyncDir = oldSync })
	quote := &api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "5", RecurringAmount: "5", AmountDueAfterCredit: "5"}
	if err := cache.save(quote); err != nil {
		t.Fatal(err)
	}
	if want := []string{configDir, cacheDir}; !reflect.DeepEqual(synced, want) {
		t.Fatalf("first save synced %v, want %v", synced, want)
	}
	synced = nil
	if _, err := cache.load(); err != nil {
		t.Fatal(err)
	}
	if want := []string{cacheDir}; !reflect.DeepEqual(synced, want) {
		t.Fatalf("load synced %v, want %v", synced, want)
	}
}

func TestPoolRetrySyncsHomeOnlyForNewConfigDirectory(t *testing.T) {
	for _, configExists := range []bool{false, true} {
		t.Run(fmt.Sprintf("config-exists=%t", configExists), func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			configPath, err := config.GetConfigPath()
			if err != nil {
				t.Fatal(err)
			}
			configDir := filepath.Dir(configPath)
			if configExists {
				if err := config.EnsureConfigDir(); err != nil {
					t.Fatal(err)
				}
			}
			keyring.MockInitWithError(fmt.Errorf("keychain unavailable"))
			t.Cleanup(keyring.MockInit)
			var synced []string
			oldSync := poolRetrySyncDir
			poolRetrySyncDir = func(path string) error {
				synced = append(synced, path)
				return nil
			}
			t.Cleanup(func() { poolRetrySyncDir = oldSync })
			if _, err := poolRetrySecret(configDir); err != nil {
				t.Fatal(err)
			}
			want := []string{configDir}
			if !configExists {
				want = []string{home, configDir}
			}
			if !reflect.DeepEqual(synced, want) {
				t.Fatalf("synced %v, want %v", synced, want)
			}
			synced = nil
			if _, err := poolRetrySecret(configDir); err != nil {
				t.Fatal(err)
			}
			if len(synced) != 0 {
				t.Fatalf("existing key synced directories: %v", synced)
			}
		})
	}
}

func TestPoolCheckoutKeyLossAbortsBeforeQuote(t *testing.T) {
	calls := 0
	client := poolTestClient(t, func(http.ResponseWriter, *http.Request) { calls++ })
	req, _ := buildPoolCheckoutRequest(42, "monthly", "credit", "", "", "key-loss", false)
	cache, err := newPoolRetryCache(client, req)
	if err != nil {
		t.Fatal(err)
	}
	cache.confirmation = "CONFIRM"
	if err := cache.save(&api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "5", RecurringAmount: "5", AmountDueAfterCredit: "5"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(filepath.Dir(cache.path))
	if err := keyring.Delete(poolKeyService, "capacity-retry-hmac-v2-"+poolRetryHash([]byte(dir))); err != nil {
		t.Fatal(err)
	}
	cmd, out, _ := poolTestCommand()
	err = runPoolCheckout(cmd, client, req, true, false)
	if err == nil || !strings.Contains(err.Error(), "authentication key") || calls != 0 || out.Len() != 0 {
		t.Fatalf("err=%v calls=%d stdout=%s", err, calls, out)
	}
}

func TestPoolCheckoutMaliciousURLAndRawJSON(t *testing.T) {
	for _, checkoutURL := range []string{"http://pay.example/", "https://pay.example/\nhttps://evil.example/", "https://pay.example/\x1b]52;c;QQ==\a", "https://pay.example/%0aINJECT", "https://user@pay.example/", "https://pay.example/ next", "https://", " https://pay.example/", "https://pay.example/\\evil", "https://pay.example/ok?token=secret"} {
		for _, jsonMode := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/json=%t", checkoutURL, jsonMode), func(t *testing.T) {
				body, _ := json.Marshal(map[string]any{"checkout_url": checkoutURL, "unknown": json.Number("9007199254740993")})
				raw := " \n" + string(body) + " \n"
				client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					var req api.ResourcePoolCheckoutRequest
					json.NewDecoder(r.Body).Decode(&req)
					if req.QuoteOnly {
						fmt.Fprint(w, `{"confirmation":"APPROVE","mode":"purchase","unit_price":"5","recurring_amount":"5","amount_due_after_credit":"5"}`)
						return
					}
					fmt.Fprint(w, raw)
				})
				req, _ := buildPoolCheckoutRequest(42, "monthly", "stripe_checkout", "", "", "url-test", false)
				cmd, out, _ := poolTestCommand()
				err := runPoolCheckout(cmd, client, req, true, jsonMode)
				if jsonMode {
					if err != nil || out.String() != raw {
						t.Fatalf("raw response changed: err=%v out=%q", err, out.String())
					}
				} else if strings.HasSuffix(checkoutURL, "token=secret") {
					if err != nil || out.String() != checkoutURL+"\n" {
						t.Fatalf("valid URL: %v %q", err, out.String())
					}
				} else if err == nil || out.Len() != 0 {
					t.Fatalf("unsafe URL emitted: err=%v out=%q", err, out.String())
				}
			})
		}
	}
}

func TestPoolHumanOutputSanitizesTerminalText(t *testing.T) {
	attack := "SAFE\x1b[31mRED\x1b[0m\x1b]52;c;ZXZpbA==\a\r\n\t\x00\x7f\u202e"
	clean := terminaltext.Clean(attack)
	if clean != "SAFERED" {
		t.Fatalf("sanitizer=%q", clean)
	}
	pool := poolSummary{PoolID: attack, DisplayName: attack, Status: attack, PlanName: attack}
	outputs := []string{
		formatPoolsSimple([]poolSummary{pool}), formatPoolsDetails([]poolSummary{pool}),
		formatPoolQuote(api.ResourcePoolCheckoutResponse{Mode: attack, PlanName: attack, BillingCycle: attack, NextDueDate: attack}),
		formatPoolReplay(api.ResourcePoolCheckoutResponse{Phase: "completed", Status: attack, InvoiceStatus: attack, InvoiceURL: attack, OrderNumber: attack, OrderStatus: attack, InvoiceNumber: attack}),
		poolHumanError(fmt.Errorf("failed: %s", attack)).Error(),
	}
	for _, output := range outputs {
		withoutLayout := strings.NewReplacer("\n", "", "\t", "").Replace(output)
		if terminaltext.Clean(withoutLayout) != withoutLayout || strings.Contains(output, "ZXZpbA") {
			t.Errorf("unsafe output: %q", output)
		}
	}
	for _, row := range newPoolsTableModel([]poolSummary{pool}).table.Rows() {
		for _, value := range row {
			if terminaltext.Clean(value) != value {
				t.Errorf("unsafe TUI cell %q", value)
			}
		}
	}
}

func TestPoolsLegacyFlagPlacement(t *testing.T) {
	oldSimple, oldDetails := poolsSimple, poolsDetails
	t.Cleanup(func() { poolsSimple, poolsDetails = oldSimple, oldDetails })
	for _, args := range [][]string{
		{"pools", "--simple", "list"}, {"pools", "list", "--simple"},
		{"pools", "--details", "show", "pool::abc"}, {"pools", "show", "pool::abc", "--details"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			poolsSimple, poolsDetails = false, false
			root, _, _ := poolTestCommand()
			root.Use = "odo"
			pools := &cobra.Command{Use: "pools"}
			pools.PersistentFlags().AddFlagSet(poolsCmd.PersistentFlags())
			pools.Flags().AddFlagSet(poolsCmd.LocalNonPersistentFlags())
			called := false
			for _, name := range []string{"list", "show"} {
				child := &cobra.Command{Use: name, RunE: func(cmd *cobra.Command, positionals []string) error {
					called = true
					if cmd.Name() == "list" && (!poolsSimple || len(positionals) != 0) {
						t.Errorf("list flags/args: %v %v", poolsSimple, positionals)
					}
					if cmd.Name() == "show" && (!poolsDetails || len(positionals) != 1 || positionals[0] != "pool::abc") {
						t.Errorf("show flags/args: %v %v", poolsDetails, positionals)
					}
					return nil
				}}
				original, _, _ := poolsCmd.Find([]string{name})
				child.Flags().AddFlagSet(original.LocalNonPersistentFlags())
				pools.AddCommand(child)
			}
			root.AddCommand(pools)
			root.SetArgs(args)
			if err := root.Execute(); err != nil || !called {
				t.Fatalf("legacy syntax: called=%t err=%v", called, err)
			}
		})
	}
}

func TestPoolRetryWrongKeyAndTamperedConfirmations(t *testing.T) {
	for _, change := range []string{"key", "card ending", "generic phrase", "hmac"} {
		t.Run(change, func(t *testing.T) {
			calls := 0
			client := poolTestClient(t, func(http.ResponseWriter, *http.Request) { calls++ })
			req, _ := buildPoolCheckoutRequest(42, "monthly", "saved_card", "pm::private", "", "authenticated-cache", false)
			cache, err := newPoolRetryCache(client, req)
			if err != nil {
				t.Fatal(err)
			}
			cache.confirmation = "APPROVE pm::private provider-secret"
			cache.cardLastFour = "4242"
			if err := cache.save(&api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "5.00", RecurringAmount: "5.00", AmountDueAfterCredit: "5.00"}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(cache.path)
			if err != nil {
				t.Fatal(err)
			}
			var record poolRetryRecord
			if err := json.Unmarshal(data, &record); err != nil {
				t.Fatal(err)
			}
			switch change {
			case "key":
				dir := filepath.Dir(filepath.Dir(cache.path))
				if err := keyring.Set(poolKeyService, "capacity-retry-hmac-v2-"+poolRetryHash([]byte(dir)), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="); err != nil {
					t.Fatal(err)
				}
			case "card ending":
				record.CardLastFour = "9999"
			case "generic phrase":
				record.Confirmation = "forged"
			case "hmac":
				record.MAC = strings.Repeat("0", 64)
			}
			data, err = json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(cache.path, data, 0600); err != nil {
				t.Fatal(err)
			}
			cmd, out, _ := poolTestCommand()
			err = runPoolCheckout(cmd, client, req, true, false)
			if err == nil || calls != 0 || out.Len() != 0 {
				t.Fatalf("accepted tampering: err=%v calls=%d stdout=%s", err, calls, out)
			}
		})
	}
}
