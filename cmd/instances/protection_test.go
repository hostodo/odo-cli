package instances

import (
	"io"
	"strings"
	"testing"
)

func TestUnprotectConfirmation(t *testing.T) {
	for _, tc := range []struct {
		input   string
		yes, ok bool
	}{
		{"UNPROTECT ins::abc\n", false, true},
		{"UNPROTECT ins::abc\r\n", false, true},
		{"UNPROTECT hostname\n", false, false},
		{" UNPROTECT ins::abc\n", false, false},
		{"unprotect ins::abc\n", false, false},
		{"yes\n", false, false},
		{"", false, false},
		{"", true, true},
	} {
		phrase, err := confirmUnprotect(strings.NewReader(tc.input), io.Discard, "ins::abc", tc.yes)
		if (err == nil) != tc.ok {
			t.Errorf("%q yes=%t: %v", tc.input, tc.yes, err)
		}
		if tc.ok && phrase != "UNPROTECT ins::abc" {
			t.Errorf("wrong phrase %q", phrase)
		}
	}
}
