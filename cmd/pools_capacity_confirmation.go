package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/hostodo/odo-cli/v2/pkg/terminaltext"
	"io"
	"math/big"
	"regexp"
	"strings"
)

var poolCardEndingPattern = regexp.MustCompile(`^[0-9]{4}$`)
var poolCardConfirmationPattern = regexp.MustCompile(` ENDING ([0-9]{4}) FOR [0-9]+\.[0-9]{2}$`)

func poolCardConfirmation(id string, amount json.Number, lastFour string) (string, error) {
	value, ok := new(big.Rat).SetString(amount.String())
	if !ok || value.Sign() < 0 || !poolCardEndingPattern.MatchString(lastFour) || strings.TrimSpace(id) == "" || terminaltext.Clean(id) != id {
		return "", fmt.Errorf("saved-card quote is missing valid card ending or charge amount; purchase aborted")
	}
	// The backend's exact phrase uses two decimal places. Never round approval.
	fixed := value.FloatString(2)
	cents, _ := new(big.Rat).SetString(fixed)
	if value.Cmp(cents) != 0 {
		return "", fmt.Errorf("saved-card quote charge amount must be exact cents; purchase aborted")
	}
	return fmt.Sprintf("CHARGE SAVED CARD %s ENDING %s FOR %s", strings.TrimSpace(id), lastFour, fixed), nil
}

func poolCardEnding(id string, amount json.Number, confirmation string) (string, error) {
	match := poolCardConfirmationPattern.FindStringSubmatch(confirmation)
	if len(match) == 2 {
		expected, err := poolCardConfirmation(id, amount, match[1])
		if err == nil && confirmation == expected {
			return match[1], nil
		}
	}
	return "", fmt.Errorf("saved-card quote payment_confirmation is missing or does not match the selected card and quote amount; purchase aborted")
}

func confirmPoolCard(in io.Reader, out io.Writer, yes, interactive bool, message, confirmation string) error {
	// --yes is explicit consent to this same charge; show it even without a prompt.
	if _, err := fmt.Fprintf(out, "Saved-card charge approval: %s\n", confirmation); err != nil {
		return err
	}
	return confirmAction(in, out, yes, interactive, message, confirmation)
}

func validatePoolConfirmation(phrase string) error {
	if strings.TrimSpace(phrase) == "" {
		return fmt.Errorf("Capacity quote is missing confirmation; hardened server required; purchase aborted")
	}
	if terminaltext.Clean(phrase) != phrase {
		return fmt.Errorf("Capacity quote confirmation contains unsafe terminal text; purchase aborted")
	}
	return nil
}

func confirmPoolPurchase(in io.Reader, out io.Writer, yes, interactive bool, message, phrase string) error {
	if err := validatePoolConfirmation(phrase); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Purchase approval: %s\n", phrase); err != nil {
		return err
	}
	return confirmAction(in, out, yes, interactive, message, phrase)
}
