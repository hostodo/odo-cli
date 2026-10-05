// Package terminaltext makes untrusted API text safe for human terminal output.
package terminaltext

import (
	"strings"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// Clean removes terminal escape sequences and control/format characters. Call it
// on data fields before adding the CLI's own newlines, tabs, or styling.
func Clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}
