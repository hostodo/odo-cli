package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/zalando/go-keyring"
)

func TestPoolCheckoutRetryQuoteDrift(t *testing.T) {
	for _, drift := range []string{"price", "credit", "mode"} {
		for _, jsonMode := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/json=%t", drift, jsonMode), func(t *testing.T) {
				cmd, out, diagnostics := poolTestCommand()
				quotes, checkouts := 0, 0
				var firstBody []byte
				var cache *poolRetryCache
				quote := `{"mode":"purchase","existing_pool_id":null,"unit_price":"7.5000000000000001","recurring_amount":"30.0000","amount_due_after_credit":"5.0001","client_secret":"provider-secret","checkout_url":"https://secret.example/checkout","card_number":"card-secret"}`
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
					fmt.Fprint(w, `{"checkout_url":"https://pay.example/replay","unknown":9007199254740993}`)
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
					if err != nil || info.Mode().Perm() != mode {
						t.Fatalf("permissions for %s: info=%v err=%v", path, info, err)
					}
				}
				switch drift {
				case "price":
					quote = strings.ReplaceAll(quote, "7.5000000000000001", "9.0000000000000001")
					quote = strings.ReplaceAll(quote, "30.0000", "36.0000")
				case "credit":
					quote = strings.ReplaceAll(quote, "5.0001", "0.0000")
				case "mode":
					quote = strings.ReplaceAll(quote, `"purchase"`, `"upgrade"`)
					quote = strings.ReplaceAll(quote, `null`, `"pool::created"`)
				}
				// New command/client instances prove reuse comes from disk.
				cmd, out, diagnostics = poolTestCommand()
				retryClient := &api.Client{BaseURL: client.BaseURL, HTTPClient: client.HTTPClient}
				if err := runPoolCheckout(cmd, retryClient, req, true, jsonMode); err != nil {
					t.Fatal(err)
				}
				if quotes != 1 || checkouts != 2 || !strings.Contains(diagnostics.String(), "saved confirmed") || strings.Contains(diagnostics.String(), "first use") {
					t.Fatalf("quotes=%d checkouts=%d stderr=%s", quotes, checkouts, diagnostics)
				}
				if jsonMode {
					var compact bytes.Buffer
					if err := json.Compact(&compact, out.Bytes()); err != nil || compact.String() != `{"checkout_url":"https://pay.example/replay","unknown":9007199254740993}` {
						t.Fatalf("stdout=%q err=%v", out.String(), err)
					}
				} else if out.String() != "https://pay.example/replay\n" {
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
	for _, problem := range []string{"truncated", "trailing", "missing quote", "missing amount", "missing pool field", "changed price", "version", "plan", "cycle", "promo", "payment method", "saved card", "key", "endpoint", "login", "permissions", "symlink"} {
		t.Run(problem, func(t *testing.T) {
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
					record["version"] = json.RawMessage(`2`)
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
				client.BaseURL += "/other"
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
				fmt.Fprint(w, `{"mode":"purchase","existing_pool_id":null,"unit_price":"7.50","recurring_amount":"30.00","amount_due_after_credit":"5.0001"}`)
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
