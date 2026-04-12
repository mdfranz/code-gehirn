package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#5C5C5C")).
			Padding(0, 1)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EE6FF8")).
				Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F5F"))

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#444444"))

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#EE6FF8"))

	summaryTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FAFAFA")).
				MarginBottom(1)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#222222")).
			Height(1)

	statusTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
)
