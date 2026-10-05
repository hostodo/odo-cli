package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestPoolCardConfirmation(t *testing.T) {
	const phrase = "CHARGE SAVED CARD pm::123 ENDING 0042 FOR 5.00"
	for _, amount := range []json.Number{"5", "5.00", "5.0000", "5e0"} {
		got, err := poolCardConfirmation(" pm::123 ", amount, "0042")
		if err != nil || got != phrase {
			t.Fatalf("amount=%s got=%q err=%v", amount, got, err)
		}
	}
	for _, amount := range []json.Number{"", "-1.00", "5.0001", "NaN"} {
		if _, err := poolCardConfirmation("pm::123", amount, "0042"); err == nil {
			t.Fatalf("accepted invalid charge %s", amount)
		}
	}
	for _, tt := range []struct {
		name, input              string
		yes, interactive, wantOK bool
	}{
		{"exact interactive", phrase + "\n", false, true, true},
		{"Windows input", phrase + "\r\n", false, true, true},
		{"generic approval", "PURCHASE\n", false, true, false},
		{"wrong amount", strings.ReplaceAll(phrase, "5.00", "6.00") + "\n", false, true, false},
		{"extra space", phrase + " \n", false, true, false},
		{"EOF", phrase, false, true, false},
		{"pipe is not consent", phrase + "\n", false, false, false},
		{"explicit automation", "", true, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			err := confirmPoolCard(strings.NewReader(tt.input), &out, tt.yes, tt.interactive, "Purchase?", phrase)
			if (err == nil) != tt.wantOK || !strings.Contains(out.String(), phrase) {
				t.Fatalf("err=%v output=%q", err, out.String())
			}
		})
	}
}

func TestPoolCardRejectsUnsafeQuoteApproval(t *testing.T) {
	for _, phrase := range []string{
		"", "CHARGE", "CHARGE SAVED CARD pm::other ENDING 4242 FOR 5.00",
		"CHARGE SAVED CARD pm::123 ENDING 4242 FOR 6.00",
		"CHARGE SAVED CARD pm::123 ENDING 4242 FOR 5.00 ",
		"CHARGE SAVED CARD pm::123 ENDING provider-secret FOR 5.00",
	} {
		t.Run(phrase, func(t *testing.T) {
			calls := 0
			client := poolTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				fmt.Fprintf(w, `{"confirmation":"PURCHASE CAPACITY","mode":"purchase","unit_price":"5.00","recurring_amount":"5.00","amount_due_after_credit":"5.00","payment_confirmation":%q}`, phrase)
			})
			req, _ := buildPoolCheckoutRequest(42, "monthly", "saved_card", "pm::123", "", "test-key", false)
			cmd, out, _ := poolTestCommand()
			err := runPoolCheckout(cmd, client, req, true, false)
			if err == nil || !strings.Contains(err.Error(), "payment_confirmation") || calls != 1 || out.Len() != 0 {
				t.Fatalf("err=%v calls=%d stdout=%s", err, calls, out)
			}
			cache, err := newPoolRetryCache(client, req)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(cache.path); !os.IsNotExist(err) {
				t.Fatalf("unconfirmed charge was cached: %v", err)
			}
		})
	}
}
