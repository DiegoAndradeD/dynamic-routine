package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
)

func (m model) viewRules(w int) string {
	var b strings.Builder
	b.WriteString(bold(colorAmber).Render("CONTINGENCY RULES") + "\n\n")

	if len(m.state.Rules) == 0 {
		b.WriteString(lipgloss.NewStyle().Width(clampInt(w-4, 20, 110)).MarginLeft(2).Foreground(colorText).
			Render("Rules are your own \"when X happens, do Y\" notes for days that do not go to plan.") + "\n")
		b.WriteString("  " + keyHint("A", "write your first rule") + "\n\n")
		b.WriteString(sectionTitle("EXAMPLES (not saved)") + "\n")
		for _, rule := range routine.ExampleRules() {
			b.WriteString("  " + muted("When "+rule.Trigger+":") + "\n")
			b.WriteString(lipgloss.NewStyle().Width(clampInt(w-6, 20, 110)).MarginLeft(4).Foreground(colorMuted).Render(rule.Instruction) + "\n\n")
		}
		return b.String()
	}
	for i, rule := range m.state.Rules {
		selected := i == m.ruleCursor
		cursor, triggerStyle := "  ", bold(colorMuted)
		if selected {
			cursor, triggerStyle = bold(colorAmber).Render("▸ "), bold(colorAmber)
		}

		b.WriteString(cursor + triggerStyle.Render(rule.Trigger) + "\n")
		b.WriteString(lipgloss.NewStyle().Width(clampInt(w-4, 20, 110)).MarginLeft(4).Foreground(colorText).Render(rule.Instruction) + "\n\n")
	}
	return b.String()
}
