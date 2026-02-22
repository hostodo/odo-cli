package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	primaryColor   = lipgloss.Color("#00D9FF")
	successColor   = lipgloss.Color("#00FF9F")
	warningColor   = lipgloss.Color("#FFD700")
	dangerColor    = lipgloss.Color("#FF6B6B")
	mutedColor     = lipgloss.Color("#6B7280")
	highlightColor = lipgloss.Color("#8B5CF6")

	// Status colors
	runningColor      = successColor
	stoppedColor      = dangerColor
	provisioningColor = warningColor
	suspendedColor    = lipgloss.Color("#FF8C00")

	// Table styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(mutedColor)

	CellStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(highlightColor).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)

	// Status badge styles
	StatusRunningStyle = lipgloss.NewStyle().
				Foreground(runningColor).
				Bold(true)

	StatusStoppedStyle = lipgloss.NewStyle().
				Foreground(stoppedColor).
				Bold(true)

	StatusProvisioningStyle = lipgloss.NewStyle().
				Foreground(provisioningColor).
				Bold(true)

	StatusSuspendedStyle = lipgloss.NewStyle().
				Foreground(suspendedColor).
				Bold(true)

	// Title and text styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(dangerColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(successColor).
			Bold(true)

	// Help text style
	HelpStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	// Border styles
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(1, 2)
)

// GetStatusStyle returns the appropriate style for a status
func GetStatusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return StatusRunningStyle
	case "stopped":
		return StatusStoppedStyle
	case "provisioning", "PROVISIONING":
		return StatusProvisioningStyle
	case "suspended", "SUSPENDED":
		return StatusSuspendedStyle
	default:
		return lipgloss.NewStyle().Foreground(mutedColor)
	}
}

// GetPowerStatusBadge returns a styled power status badge
func GetPowerStatusBadge(status string) string {
	style := GetStatusStyle(status)
	var icon string
	switch status {
	case "running":
		icon = "●"
	case "stopped":
		icon = "○"
	default:
		icon = "◐"
	}
	return style.Render(icon + " " + status)
}
