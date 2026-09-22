package cmd

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/hostodo/odo-cli/v2/pkg/terminaltext"
)

func poolHumanError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", terminaltext.Clean(err.Error()))
}

func validatePoolCheckoutURL(value string) error {
	invalid := func() error {
		return fmt.Errorf("Capacity checkout returned an invalid HTTPS checkout URL; reconcile using the idempotency key")
	}
	if terminaltext.Clean(value) != value || strings.ContainsAny(value, "\\") || strings.IndexFunc(value, unicode.IsSpace) >= 0 {
		return invalid()
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.Opaque != "" {
		return invalid()
	}
	// Reject escaped whitespace/control characters too; consumers may decode URLs.
	decoded, err := url.PathUnescape(value)
	if err != nil || terminaltext.Clean(decoded) != decoded || strings.IndexFunc(decoded, unicode.IsSpace) >= 0 || strings.Contains(decoded, "\\") {
		return invalid()
	}
	return nil
}
