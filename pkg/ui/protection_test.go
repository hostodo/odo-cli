package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hostodo/odo-cli/v2/pkg/api"
)

func TestProtectionVisibility(t *testing.T) {
	instances := []api.Instance{{InstanceID: "ins::abc", ProtectionEnabled: true}}
	json, err := FormatInstancesJSON(instances)
	if err != nil || !strings.Contains(json, `"protection_enabled": true`) {
		t.Fatalf("missing JSON protection: %s %v", json, err)
	}
	for name, output := range map[string]string{
		"simple":  FormatInstancesSimpleTable(instances),
		"details": FormatInstancesDetailedTable(instances),
		"status":  FormatInstanceDetail(&instances[0]),
		"tui":     NewTableModel(instances, nil).View(),
	} {
		if !strings.Contains(strings.ToLower(output), "protect") || !strings.Contains(output, "true") {
			t.Errorf("%s missing state: %s", name, output)
		}
	}
}

func TestProtectedTUIBlocksOnlyRenameAndReinstall(t *testing.T) {
	for key, action := range map[rune]string{'n': "", 'i': "", 's': "start", 'S': "stop", 'x': "force-stop", 'r': "restart", 'R': "force-restart"} {
		m := NewTableModel([]api.Instance{{ProtectionEnabled: true}}, nil)
		m.mode = DetailMode
		updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		got := updated.(TableModel)
		if got.APIAction != action || got.mode == RenameMode {
			t.Errorf("key %c: action=%s mode=%v", key, got.APIAction, got.mode)
		}
	}
}
