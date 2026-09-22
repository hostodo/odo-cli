package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hostodo/odo-cli/v2/pkg/api"
	"github.com/hostodo/odo-cli/v2/pkg/terminaltext"
)

func marshalPoolReplay(result api.ResourcePoolCheckoutResponse) ([]byte, error) {
	// Replays are reconciliation records, never provider credential delivery.
	safe := map[string]any{
		"idempotent_replay": true,
		"phase":             terminaltext.Clean(result.Phase),
		"status":            terminaltext.Clean(result.Status),
		"order_number":      terminaltext.Clean(result.OrderNumber),
		"order_status":      terminaltext.Clean(result.OrderStatus),
		"invoice_number":    terminaltext.Clean(result.InvoiceNumber),
		"invoice_status":    terminaltext.Clean(result.InvoiceStatus),
	}
	if result.AmountDue.String() != "" {
		safe["amount_due"] = result.AmountDue
	}
	data, err := json.MarshalIndent(safe, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func formatPoolReplay(result api.ResourcePoolCheckoutResponse) string {
	state := "unknown outcome; reconcile the order and invoice before using a new key"
	switch result.Phase {
	case "preparing", "order_linked", "provider_started":
		state = "processing; payment outcome is not yet confirmed"
	case "completed":
		state = "completed"
		if result.InvoiceStatus != "" {
			state += "; invoice " + terminaltext.Clean(result.InvoiceStatus)
		}
	}
	var out strings.Builder
	fmt.Fprintf(&out, "Capacity checkout replay: %s\n", state)
	for _, field := range [][2]string{
		{"Phase", result.Phase}, {"Status", result.Status},
		{"Order", result.OrderNumber}, {"Order status", result.OrderStatus},
		{"Invoice", result.InvoiceNumber}, {"Invoice status", result.InvoiceStatus},
	} {
		if field[1] != "" {
			fmt.Fprintf(&out, "  %s: %s\n", field[0], terminaltext.Clean(field[1]))
		}
	}
	if result.AmountDue != "" {
		fmt.Fprintf(&out, "  Amount due: %s\n", poolMoney(result.AmountDue.String()))
	}
	return out.String()
}
