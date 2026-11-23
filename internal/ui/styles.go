package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorAmber   = lipgloss.Color("#F59E0B")
	colorEmerald = lipgloss.Color("#10B981")
	colorRose    = lipgloss.Color("#F43F5E")
	colorPurple  = lipgloss.Color("#A855F7")
	colorCyan    = lipgloss.Color("#06B6D4")
	colorMuted   = lipgloss.Color("#737373")
	colorText    = lipgloss.Color("#E5E5E5")
	colorBorder  = lipgloss.Color("#3F3F46")
	colorSelBg   = lipgloss.Color("#27272A")
)

func fg(c lipgloss.Color) lipgloss.Style { return lipgloss.NewStyle().Foreground(c) }

func bold(c lipgloss.Color) lipgloss.Style { return lipgloss.NewStyle().Bold(true).Foreground(c) }

func muted(s string) string { return fg(colorMuted).Render(s) }

func sectionTitle(s string) string { return bold(colorMuted).Render(s) }

func keyHint(key, label string) string {
	return bold(colorAmber).Render("["+key+"]") + " " + label
}

func box(w int, border lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(w - 2)
}

func badge(text string, bg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#0A0A0A")).
		Background(bg).
		Bold(true).
		Padding(0, 1).
		Render(text)
}

// progressBar draws ratio (0..1, clamped) in width cells.
func progressBar(ratio float64, width int, color lipgloss.Color) string {
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	return fg(color).Render(strings.Repeat("█", filled)) +
		fg(lipgloss.Color("#3F3F46")).Render(strings.Repeat("░", width-filled))
}
