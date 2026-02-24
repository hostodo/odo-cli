package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hostodo/hostodo-cli/pkg/api"
)

// ViewMode represents the current view mode
type ViewMode int

const (
	ListMode ViewMode = iota
	DetailMode
)

// PowerStatusFunc fetches power status for an instance ID
type PowerStatusFunc func(instanceID string) (string, error)

// powerStatusMsg carries the result of a power status fetch
type powerStatusMsg struct {
	index  int
	status string
}

// TableModel represents the Bubble Tea model for the instances table
type TableModel struct {
	table            table.Model
	instances        []api.Instance
	selectedInstance int
	mode             ViewMode
	quitting         bool
	fetchPowerStatus PowerStatusFunc
	SSHHostname      string // Public: checked by caller after Run() to trigger SSH
}

// NewTableModel creates a new table model with instances
func NewTableModel(instances []api.Instance, fetchPower PowerStatusFunc) TableModel {
	columns := []table.Column{
		{Title: "ID", Width: 16},
		{Title: "Hostname", Width: 25},
		{Title: "IP Address", Width: 16},
		{Title: "Status", Width: 14},
		{Title: "RAM", Width: 10},
		{Title: "CPU", Width: 6},
		{Title: "Disk", Width: 8},
	}

	rows := make([]table.Row, len(instances))
	for i, instance := range instances {
		rows[i] = table.Row{
			instance.InstanceID,
			truncate(instance.Hostname, 25),
			instance.MainIP,
			instance.Status,
			fmt.Sprintf("%d MB", instance.RAM),
			fmt.Sprintf("%d", instance.VCPU),
			fmt.Sprintf("%d GB", instance.Disk),
		}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(min(len(instances)+2, 20)),
	)

	// Custom styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true).
		Foreground(primaryColor)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(highlightColor).
		Bold(true)

	t.SetStyles(s)

	return TableModel{
		table:            t,
		instances:        instances,
		mode:             ListMode,
		fetchPowerStatus: fetchPower,
	}
}

// Init initializes the table model
func (m TableModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m TableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case powerStatusMsg:
		if msg.index < len(m.instances) {
			m.instances[msg.index].PowerStatus = msg.status
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if m.mode == DetailMode {
				m.mode = ListMode
				return m, nil
			}
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if m.mode == ListMode {
				m.selectedInstance = m.table.Cursor()
				if m.selectedInstance < len(m.instances) {
					m.mode = DetailMode
					m.instances[m.selectedInstance].PowerStatus = "loading..."
					return m, m.fetchPowerStatusCmd(m.selectedInstance)
				}
			} else {
				m.mode = ListMode
				return m, nil
			}
		case "c":
			if m.mode == DetailMode {
				m.SSHHostname = m.instances[m.selectedInstance].Hostname
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	if m.mode == ListMode {
		m.table, cmd = m.table.Update(msg)
	}
	return m, cmd
}

func (m TableModel) fetchPowerStatusCmd(index int) tea.Cmd {
	if m.fetchPowerStatus == nil {
		return nil
	}
	instanceID := m.instances[index].InstanceID
	return func() tea.Msg {
		status, err := m.fetchPowerStatus(instanceID)
		if err != nil {
			status = "unknown"
		}
		return powerStatusMsg{index: index, status: status}
	}
}

// View renders the table or detail view
func (m TableModel) View() string {
	if m.quitting {
		if m.SSHHostname != "" {
			// Exiting for SSH — don't render table
			return ""
		}
		// Render the table one last time so it persists in scrollback
		var sb strings.Builder
		sb.WriteString(TitleStyle.Render("Hostodo Instances") + "\n\n")
		sb.WriteString(m.table.View() + "\n")
		return sb.String()
	}

	if m.mode == DetailMode {
		var sb strings.Builder
		sb.WriteString(FormatInstanceDetail(&m.instances[m.selectedInstance]))
		sb.WriteString("\n")
		sb.WriteString(HelpStyle.Render("[c] SSH Connect • Enter: back to list • q/Esc: quit"))
		sb.WriteString("\n")
		return sb.String()
	}

	// Show list view
	var sb strings.Builder
	title := TitleStyle.Render("Hostodo Instances")
	sb.WriteString(title + "\n\n")
	sb.WriteString(m.table.View() + "\n\n")
	help := HelpStyle.Render("↑/↓: Navigate • Enter: Details • q: Quit")
	sb.WriteString(help + "\n")

	return sb.String()
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
