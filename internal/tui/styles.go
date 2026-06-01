package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	colorInk       = "#F8F5EF"
	colorSoft      = "#C8D3F5"
	colorMuted     = "#7A8299"
	colorDim       = "#535A6B"
	colorAccent    = "#D946EF"
	colorAccentHot = "#FF9F1C"
	colorCyan      = "#00E5C7"
	colorSuccess   = "#2EE59D"
	colorWarn      = "#FFD166"
	colorError     = "#FF4D6D"
	colorPanel     = "#414868"
	colorPanelHot  = "#D946EF"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorAccentHot))
	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorSoft))
	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))
	errorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorError))
	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorSuccess))
	warnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorWarn))
	accentStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorCyan))
	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))
	faintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorDim))
	fieldLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorCyan))
	sectionStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorAccentHot))
	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorInk))
	statusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorAccent))
	selectedPlanStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(colorAccentHot))
	selectedPlanDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(colorCyan))
	selectedStripeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(colorAccentHot))
	toggleOnStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorSuccess))
	togglePartialStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(colorWarn))
	toggleOffStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorMuted))
	logStepStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorCyan))
	logCommandStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorWarn))
	logDoneStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorSuccess))
	logErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(colorError))
	logPanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(colorAccentHot))
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color(colorPanelHot)).
			Padding(1, 2)
	runPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color(colorPanel)).
			Padding(0, 1)
)

func divider(width int) string {
	if width < 1 {
		width = 1
	}
	return faintStyle.Render(strings.Repeat("━", width))
}
