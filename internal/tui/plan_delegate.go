package tui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type planDelegate struct{}

func (d planDelegate) Height() int {
	return 2
}

func (d planDelegate) Spacing() int {
	return 0
}

func (d planDelegate) Update(tea.Msg, *list.Model) tea.Cmd {
	return nil
}

func (d planDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	p, ok := item.(planItem)
	if !ok {
		return
	}

	width := m.Width()
	if width <= 0 {
		return
	}

	selected := index == m.Index() && m.FilterState() != list.Filtering
	depth, state, name := splitPlanTitle(p.title)
	selector := "  "
	titleStyle := valueStyle
	descStyle := dimStyle
	toggleStyle := toggleOffStyle
	if selected {
		selector = selectedStripeStyle.Render("▌") + " "
		titleStyle = selectedPlanStyle
		descStyle = selectedPlanDescStyle
	}
	toggle := "○"
	switch state {
	case toggleOn:
		toggle = "●"
		toggleStyle = toggleOnStyle
	case togglePartial:
		toggle = "◐"
		toggleStyle = togglePartialStyle
	}

	titlePrefix := selector + depth
	availableTitle := width - ansi.StringWidth(titlePrefix) - ansi.StringWidth(toggle) - 1
	if availableTitle < 1 {
		availableTitle = 1
	}
	title := titleStyle.Render(ansi.Truncate(name, availableTitle, "…"))
	titleLine := titlePrefix + toggleStyle.Render(toggle) + " " + title

	descPrefix := "  " + depth + "    "
	availableDesc := width - ansi.StringWidth(descPrefix)
	if availableDesc < 1 {
		availableDesc = 1
	}
	desc := ansi.Truncate(p.desc, availableDesc, "…")
	descLine := descPrefix + descStyle.Render(desc)

	fmt.Fprintf(w, "%s\n%s", titleLine, descLine) //nolint:errcheck
}

type toggleState int

const (
	toggleOff toggleState = iota
	toggleOn
	togglePartial
)

func splitPlanTitle(title string) (depth string, state toggleState, name string) {
	depthLen := len(title) - len(strings.TrimLeft(title, " "))
	depth = title[:depthLen]
	rest := strings.TrimLeft(title, " ")
	if len(rest) >= 3 {
		switch rest[:3] {
		case "[x]":
			return depth, toggleOn, strings.TrimSpace(rest[3:])
		case "[-]":
			return depth, togglePartial, strings.TrimSpace(rest[3:])
		case "[ ]":
			return depth, toggleOff, strings.TrimSpace(rest[3:])
		}
	}
	return depth, toggleOff, strings.TrimSpace(rest)
}

var _ list.ItemDelegate = planDelegate{}
