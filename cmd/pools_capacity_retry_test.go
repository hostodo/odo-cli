package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/zalando/go-keyring"
)

func TestPoolCheckoutRetryQuoteDrift(t *testing.T) {
	for _, drift := range []string{"price", "credit", "mode", "card ending", "confirmation", "token rotation"} {
		for _, jsonMode := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", drift, jsonMode), func(t *testing.T) {
				cmd, out, diagnostics := poolTestCommand()
				quotes, checkouts := 0, 0
				var firstBody []byte
				var cache *poolRetryCache
				quote := `{"confirmation":"PURCHASE CAPACITY pm::private provider-secret","mode":"purchase","existing_pool_id":null,"unit_price":"7.5000000000000001","recurring_amount":"30.0000","amount_due_after_credit":"5.00","client_secret":"provider-secret","checkout_url":"https://secret.example/checkout","card_number":"card-secret","payment_confirmation":"CHARGE SAVED CARD pm::private ENDING 4242 FOR 5.00"}`
				client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					var body json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
						return
					}
					var req api.ResourcePoolCheckoutRequest
					if err := json.Unmarshal(body, &req); err != nil {
						t.Error(err)
						return
					}
					if req.QuoteOnly {
						quotes++
						fmt.Fprint(w, quote)
						return
					}
					checkouts++
					if req.PaymentConfirmation != "CHARGE SAVED CARD pm::private ENDING 4242 FOR 5.00" || req.ApprovedChargeAmount != "5.00" {
						t.Errorf("checkout missing original approval: %+v", req)
					}
					// The exact snapshot must already be durable when checkout starts.
					saved, err := cache.load()
					if err != nil || !reflect.DeepEqual(saved, req.ExpectedQuote) {
						t.Errorf("checkout before saving snapshot: saved=%+v request=%+v err=%v", saved, req.ExpectedQuote, err)
					}
					if checkouts == 1 {
						firstBody = append([]byte(nil), body...)
						// The backend may have committed despite this unknown outcome.
						w.WriteHeader(http.StatusBadGateway)
						fmt.Fprint(w, `{"detail":"upstream response lost"}`)
						return
					}
					if !bytes.Equal(firstBody, body) {
						w.WriteHeader(http.StatusConflict)
						fmt.Fprint(w, `{"detail":"idempotency payload mismatch"}`)
						return
					}
					fmt.Fprint(w, `{"idempotent_replay":true,"phase":"completed","order_number":"ORD-1","invoice_number":"INV-1","invoice_status":"unpaid","invoice_url":"https://panel.example/invoices/INV-1","checkout_url":"https://pay.example/replay","checkout":{"client_secret":"secret"},"unknown":9007199254740993}`)
				})
				req, err := buildPoolCheckoutRequest(42, "annually", "saved_card", "pm::private", "SAVE", "explicit/../retry-key", false)
				if err != nil {
					t.Fatal(err)
				}
				cache, err = newPoolRetryCache(client, req)
				if err != nil {
					t.Fatal(err)
				}
				if err := runPoolCheckout(cmd, client, req, true, jsonMode); err == nil || out.Len() != 0 {
					t.Fatalf("first checkout: err=%v stdout=%q", err, out.String())
				}
				if !strings.Contains(diagnostics.String(), "first use") || !strings.Contains(diagnostics.String(), "Confirmed quote saved") {
					t.Fatalf("missing first-use diagnostics: %s", diagnostics)
				}
				original, err := os.ReadFile(cache.path)
				if err != nil {
					t.Fatal(err)
				}
				for _, secret := range []string{"provider-secret", "card-secret", "https://secret.example", "pm::private", "capacity-test-token", req.IdempotencyKey} {
					if bytes.Contains(original, []byte(secret)) {
						t.Errorf("cache contains sensitive input %q", secret)
					}
				}
				for path, mode := range map[string]os.FileMode{cache.path: 0600, filepath.Dir(cache.path): 0700} {
					info, err := os.Stat(path)
					if err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != mode) {
						t.Fatalf("permissions for %s: info=%v err=%v", path, info, err)
					}
				}
				switch drift {
				case "price":
					quote = strings.ReplaceAll(quote, "7.5000000000000001", "9.0000000000000001")
					quote = strings.ReplaceAll(quote, "30.0000", "36.0000")
				case "credit":
					quote = strings.ReplaceAll(quote, "5.00", "0.0000")
				case "mode":
					quote = strings.ReplaceAll(quote, `"purchase"`, `"upgrade"`)
					quote = strings.ReplaceAll(quote, `null`, `"pool::created"`)
				case "confirmation":
					quote = strings.ReplaceAll(quote, "PURCHASE CAPACITY", "NEW SERVER PHRASE")
				case "token rotation":
					if err := keyring.Set("odo-cli", "access-token", "rotated-token-same-user"); err != nil {
						t.Fatal(err)
					}
				case "card ending":
					quote = strings.ReplaceAll(quote, "ENDING 4242", "ENDING 9999")
				}
				// New command/client instances prove reuse comes from disk.
				cmd, out, diagnostics = poolTestCommand()
				retryClient := &api.Client{BaseURL: client.BaseURL, HTTPClient: client.HTTPClient}
				if err := runPoolCheckout(cmd, retryClient, req, false, jsonMode); err == nil || !strings.Contains(err.Error(), "--yes") || quotes != 1 || checkouts != 1 {
					t.Fatalf("retry bypassed explicit confirmation: err=%v quotes=%d checkouts=%d", err, quotes, checkouts)
				}
				if err := runPoolCheckout(cmd, retryClient, req, true, jsonMode); err != nil {
					t.Fatal(err)
				}
				if quotes != 1 || checkouts != 2 || !strings.Contains(diagnostics.String(), "saved confirmed") || strings.Contains(diagnostics.String(), "first use") {
					t.Fatalf("quotes=%d checkouts=%d stderr=%s", quotes, checkouts, diagnostics)
				}
				if jsonMode {
					if strings.Contains(out.String(), "pay.example") || strings.Contains(out.String(), "client_secret") || !strings.Contains(out.String(), `"idempotent_replay": true`) {
						t.Fatalf("unsafe replay stdout=%q", out.String())
					}
				} else if out.Len() != 0 || strings.Contains(diagnostics.String(), "pay.example") {
					t.Fatalf("stdout=%q", out.String())
				}
				current, err := os.ReadFile(cache.path)
				if err != nil || !bytes.Equal(original, current) {
					t.Fatalf("retry replaced snapshot: err=%v", err)
				}
			})
		}
	}
}

func TestPoolCheckoutRetryRejectsInvalidCache(t *testing.T) {
	for _, problem := range []string{"truncated", "trailing", "missing quote", "missing amount", "missing pool field", "changed price", "forged HMAC", "version", "plan", "cycle", "promo", "payment method", "saved card", "key", "endpoint", "login", "permissions", "symlink"} {
		t.Run(problem, func(t *testing.T) {
			if runtime.GOOS == "windows" && problem == "permissions" {
				t.Skip("Unix mode bits do not represent Windows ACLs")
			}
			calls := 0
			client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				t.Error("invalid cache caused a network request")
			})
			req, err := buildPoolCheckoutRequest(42, "monthly", "saved_card", "pm::123", "SAVE", "original-key", false)
			if err != nil {
				t.Fatal(err)
			}
			cache, err := newPoolRetryCache(client, req)
			if err != nil {
				t.Fatal(err)
			}
			quote := &api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "7.5000", RecurringAmount: "30.00", AmountDueAfterCredit: "5.0001"}
			cache.confirmation = "PURCHASE CAPACITY"
			if err := cache.save(quote); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(cache.path)
			if err != nil {
				t.Fatal(err)
			}
			var record map[string]json.RawMessage
			if err := json.Unmarshal(data, &record); err != nil {
				t.Fatal(err)
			}
			switch problem {
			case "forged HMAC":
				var forged poolRetryRecord
				if err := json.Unmarshal(data, &forged); err != nil {
					t.Fatal(err)
				}
				forged.Quote.UnitPrice = "999.00"
				forged.MAC = ""
				unsigned, _ := json.Marshal(forged)
				forged.MAC = poolRetryHash(unsigned) // An attacker can recompute SHA-256, but not the HMAC.
				data, err = json.Marshal(forged)
				if err != nil {
					t.Fatal(err)
				}
			case "truncated":
				data = data[:len(data)/2]
			case "trailing":
				data = append(data, []byte(` {}`)...)
			case "missing quote", "missing amount", "missing pool field", "changed price", "version":
				switch problem {
				case "missing quote":
					delete(record, "expected_quote")
				case "missing amount":
					record["expected_quote"] = bytes.Replace(record["expected_quote"], []byte(`,"amount_due_after_credit":5.0001`), nil, 1)
				case "missing pool field":
					record["expected_quote"] = bytes.Replace(record["expected_quote"], []byte(`"existing_pool_id":null,`), nil, 1)
				case "changed price":
					record["expected_quote"] = bytes.Replace(record["expected_quote"], []byte(`"unit_price":7.5000`), []byte(`"unit_price":99`), 1)
				case "version":
					record["version"] = json.RawMessage(`1`)
				}
				data, err = json.Marshal(record)
				if err != nil {
					t.Fatal(err)
				}
			case "plan":
				req.PlanID++
			case "cycle":
				req.BillingCycle = "annually"
			case "promo":
				req.Promocode = "OTHER"
			case "payment method":
				req.PaymentMethod, req.PaymentMethodID = "credit", ""
			case "saved card":
				req.PaymentMethodID = "pm::other"
			case "key":
				req.IdempotencyKey = "different-key"
				other, err := newPoolRetryCache(client, req)
				if err != nil {
					t.Fatal(err)
				}
				cache.path = other.path // A record copied under the wrong key.
			case "endpoint":
				client.BaseURL = strings.Replace(client.BaseURL, "127.0.0.1", "localhost", 1)
			case "login":
				if err := keyring.Set("odo-cli", "access-token", "different-login"); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(cache.path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if problem == "permissions" {
				if err := os.Chmod(cache.path, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if problem == "symlink" {
				if err := os.Rename(cache.path, cache.path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(cache.path+".original", cache.path); err != nil {
					if runtime.GOOS == "windows" {
						t.Skipf("symlink creation unavailable: %v", err)
					}
					t.Fatal(err)
				}
			}
			cmd, out, _ := poolTestCommand()
			err = runPoolCheckout(cmd, client, req, true, false)
			if err == nil || !strings.Contains(err.Error(), "Capacity retry cache") || !strings.Contains(err.Error(), "checkout aborted") || calls != 0 || out.Len() != 0 {
				t.Fatalf("err=%v calls=%d stdout=%q", err, calls, out.String())
			}
			current, err := os.ReadFile(cache.path)
			if err != nil || !bytes.Equal(data, current) {
				t.Fatalf("invalid record overwritten: err=%v", err)
			}
		})
	}
}

func TestPoolCheckoutRetrySaveFailureAborts(t *testing.T) {
	for _, reason := range []string{"concurrent first use", "unwritable cache"} {
		t.Run(reason, func(t *testing.T) {
			var cache *poolRetryCache
			calls := 0
			client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls != 1 {
					t.Error("checkout sent despite persistence failure")
				}
				if err := os.MkdirAll(filepath.Dir(cache.path), 0700); err != nil {
					t.Error(err)
				}
				if reason == "concurrent first use" {
					other := &api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "99.00", RecurringAmount: "99.00", AmountDueAfterCredit: "99.00"}
					cache.confirmation = "PURCHASE CAPACITY"
					if err := cache.save(other); err != nil {
						t.Error(err)
					}
				} else {
					if err := os.Remove(filepath.Dir(cache.path)); err != nil {
						t.Error(err)
					}
					if err := os.WriteFile(filepath.Dir(cache.path), []byte("blocked"), 0600); err != nil {
						t.Error(err)
					}
				}
				fmt.Fprint(w, `{"confirmation":"PURCHASE CAPACITY","mode":"purchase","existing_pool_id":null,"unit_price":"7.50","recurring_amount":"30.00","amount_due_after_credit":"5.0001"}`)
			})
			req, err := buildPoolCheckoutRequest(42, "monthly", "credit", "", "", "same-key", false)
			if err != nil {
				t.Fatal(err)
			}
			cache, err = newPoolRetryCache(client, req)
			if err != nil {
				t.Fatal(err)
			}
			cmd, out, _ := poolTestCommand()
			err = runPoolCheckout(cmd, client, req, true, false)
			if err == nil || !strings.Contains(err.Error(), "checkout aborted") || calls != 1 || out.Len() != 0 {
				t.Fatalf("err=%v calls=%d stdout=%q", err, calls, out.String())
			}
			if reason == "concurrent first use" {
				quote, err := cache.load()
				if err != nil || quote.UnitPrice.String() != "99.00" {
					t.Fatalf("overwrote concurrent quote: quote=%+v err=%v", quote, err)
				}
			}
		})
	}
}

func TestPoolRetryConcurrentPublication(t *testing.T) {
	client := poolTestClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("cache test must not contact API")
	})
	req, _ := buildPoolCheckoutRequest(42, "monthly", "credit", "", "", "same-key", false)
	cache, err := newPoolRetryCache(client, req)
	if err != nil {
		t.Fatal(err)
	}
	cache.confirmation = "PURCHASE CAPACITY"
	const writers = 8
	results := make(chan string, writers)
	start := make(chan struct{})
	var group sync.WaitGroup
	for i := 1; i <= writers; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			<-start
			amount := json.Number(fmt.Sprintf("%d.00", i))
			quote := &api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: amount, RecurringAmount: amount, AmountDueAfterCredit: amount}
			if cache.save(quote) == nil {
				results <- amount.String()
			}
		}(i)
	}
	close(start)
	group.Wait()
	close(results)
	var winners []string
	for amount := range results {
		winners = append(winners, amount)
	}
	if len(winners) != 1 {
		t.Fatalf("successful writers=%v, want exactly one", winners)
	}
	quote, err := cache.load()
	if err != nil || quote == nil || quote.AmountDueAfterCredit.String() != winners[0] {
		t.Fatalf("winning record was overwritten or incomplete: quote=%+v err=%v", quote, err)
	}
	files, err := os.ReadDir(filepath.Dir(cache.path))
	if err != nil || len(files) != 1 {
		t.Fatalf("temporary files not cleaned up: files=%v err=%v", files, err)
	}
}

func TestPoolCardRetryMissingApprovalFailsClosed(t *testing.T) {
	client := poolTestClient(t, func(http.ResponseWriter, *http.Request) {
		t.Error("legacy saved-card retry must not fetch a replacement quote or checkout")
	})
	req, _ := buildPoolCheckoutRequest(42, "monthly", "saved_card", "pm::123", "", "legacy-key", false)
	cache, err := newPoolRetryCache(client, req)
	if err != nil {
		t.Fatal(err)
	}
	quote := &api.ResourcePoolExpectedQuote{Mode: "purchase", UnitPrice: "5.00", RecurringAmount: "5.00", AmountDueAfterCredit: "5.00"}
	cache.confirmation = "PURCHASE CAPACITY"
	if err := cache.save(quote); err != nil {
		t.Fatal(err)
	}
	cmd, out, _ := poolTestCommand()
	err = runPoolCheckout(cmd, client, req, true, false)
	if err == nil || !strings.Contains(err.Error(), "card ending") || out.Len() != 0 {
		t.Fatalf("err=%v stdout=%s", err, out)
	}
}
