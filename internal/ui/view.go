package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View lays the screen out in three bands: a header, a body that takes all the
// space that is left, and a footer with the messages and the keys.
func (m model) View() string {
	w := m.contentWidth()
	header := m.renderHeader(w)
	footer := m.renderFooter(w)

	if m.height > 0 {
		m.bodyH = m.height - strings.Count(header, "\n") - strings.Count(footer, "\n") - 2
		if m.bodyH < 3 {
			m.bodyH = 3
		}
	}
	body := fitHeight(m.renderBody(w), m.bodyH)
	return header + "\n" + body + "\n" + footer
}

func (m model) renderBody(w int) string {
	switch {
	case m.welcome:
		return m.viewWelcome(w)
	case m.help:
		return m.viewHelp(w)
	case m.form != nil:
		return m.viewForm(w)
	case m.picking:
		return m.viewPicker(w)
	case m.trackPicking:
		return m.viewTrackPicker(w)
	}
	switch m.tab {
	case tabTime:
		return m.viewTime(w)
	case tabHistory:
		return m.viewHistory(w)
	case tabRules:
		return m.viewRules(w)
	case tabPlan:
		return m.viewPlan(w)
	}
	return m.viewDashboard(w)
}

func (m model) renderHeader(w int) string {
	title := bold(lipgloss.Color("#FAFAFA")).Render("⚡ DYNAMIC ROUTINE")
	clock := muted(fmt.Sprintf("%s · %s", fmtDate(m.now), m.now.Format("15:04")))

	timer := ""
	if t := m.state.Timer; t != nil {
		track, _ := m.state.FindTrack(t.TrackID)
		timer = bold(colorAmber).Render(fmt.Sprintf("● %s %s", track.Name, fmtClock(m.now.Sub(t.Since))))
	}
	gap := w - lipgloss.Width(title) - lipgloss.Width(clock) - lipgloss.Width(timer)
	if timer != "" {
		gap -= 4
	}
	if gap < 2 {
		gap = 2
	}
	top := title
	if timer != "" {
		top += "  " + timer + "  "
	}
	top += strings.Repeat(" ", gap) + clock

	// Names for every tab if they fit; otherwise only the active one gets its name.
	full := 0
	for i, name := range tabNames {
		full += len(name) + 5
		_ = i
	}
	compact := full > w
	var tabs []string
	for i, name := range tabNames {
		active := tab(i) == m.tab && !m.picking && !m.trackPicking && !m.welcome
		label := fmt.Sprintf(" %d %s ", i+1, name)
		if compact && !active {
			label = fmt.Sprintf(" %d ", i+1)
		}
		if active {
			tabs = append(tabs, lipgloss.NewStyle().Bold(true).
				Foreground(lipgloss.Color("#0A0A0A")).Background(colorAmber).Render(label))
		} else {
			tabs = append(tabs, muted(label))
		}
	}
	return top + "\n" + strings.Join(tabs, " ") + "\n" + m.renderStatusLine(w)
}

// renderStatusLine shows where the state lives, whether it is encrypted, and a
// warning when the sync tool left conflict copies behind. It never wraps: a long
// path is shortened from the left.
func (m model) renderStatusLine(w int) string {
	mode := " (encrypted)"
	if !m.store.Encrypted() {
		mode = " (NOT encrypted)"
	}
	warning := ""
	if m.conflicts > 0 {
		warning = fmt.Sprintf("⚠ %d sync conflict copies: run `routine merge`", m.conflicts)
	}

	path := m.store.Path()
	room := w - len([]rune(mode)) - len([]rune(warning)) - 2
	if r := []rune(path); room > 8 && len(r) > room {
		path = "…" + string(r[len(r)-room+1:])
	} else if room <= 8 {
		path = ""
	}
	line := muted(path + mode)
	if warning != "" {
		line += "  " + bold(colorRose).Render(warning)
	}
	return line
}

// renderFooter is one line for messages (or the pending question) and the key hints.
func (m model) renderFooter(w int) string {
	msg := ""
	switch {
	case m.confirmYes != nil:
		msg = bold(colorRose).Render(m.confirmText + "[Y] yes, any other key cancels")
	case m.flashMsg != "" && m.now.Before(m.flashUntil):
		if m.flashIsErr {
			msg = bold(colorRose).Render("✖ " + m.flashMsg)
		} else {
			msg = bold(colorEmerald).Render("✔ " + m.flashMsg)
		}
	}
	msg = lipgloss.NewStyle().Width(w).Render(msg)
	return msg + "\n" + lipgloss.NewStyle().Width(w).Render(strings.Join(m.hints(), muted(" • ")))
}

func (m model) hints() []string {
	quit := keyHint("Q", "quit")
	switch {
	case m.welcome:
		return []string{keyHint("E", "keep the example"), keyHint("N", "start empty"), quit}
	case m.help:
		return []string{keyHint("any key", "close help")}
	case m.form != nil:
		return []string{keyHint("Tab", "next field"), keyHint("Enter", "next/save"), keyHint("Ctrl+S", "save"), keyHint("Esc", "cancel")}
	case m.picking:
		return []string{keyHint("↑/↓", "move"), keyHint("Enter", "choose"), keyHint("1-9", "pick directly"), keyHint("Esc", "cancel")}
	case m.trackPicking:
		return []string{keyHint("↑/↓", "move"), keyHint("Enter", "start"), keyHint("1-9", "pick directly"), keyHint("Esc", "cancel")}
	}

	switch m.tab {
	case tabTime:
		return []string{
			keyHint("↑/↓", "day/entry"), keyHint("←/→", "week"), keyHint("Tab", "track"), m.timerHint(),
			keyHint("A", "log time"), keyHint("D", "delete"), keyHint("G", "today"), keyHint("?", "help"), quit,
		}
	case tabHistory:
		return []string{keyHint("↑/↓", "scroll"), keyHint("PgUp/PgDn", "page"), m.timerHint(), keyHint("Esc", "dashboard"), quit}
	case tabRules:
		return []string{keyHint("↑/↓", "move"), keyHint("A", "add"), keyHint("Enter", "edit"), keyHint("D", "delete"), keyHint("Esc", "dashboard"), quit}
	case tabPlan:
		return []string{
			keyHint("↑/↓", "move"), keyHint("A", "add activity"), keyHint("T", "add time track"),
			keyHint("Enter", "edit"), keyHint("D", "delete"), keyHint("?", "help"), quit,
		}
	}
	return []string{
		keyHint("C/Enter", "complete"), keyHint("M", fmt.Sprintf("micro (%dm)", m.state.MicroMinutes)),
		keyHint("S", "choose"), keyHint("U", "undo"), m.timerHint(), keyHint("L", "log time"),
		keyHint("2-5", "tabs"), keyHint("?", "help"), quit,
	}
}

func (m model) timerHint() string {
	if m.state.TimerRunning() {
		return keyHint("T", "stop timer")
	}
	return keyHint("T", "start timer")
}

// shortClock formats a time of day.
func shortClock(t time.Time) string { return t.Format("15:04") }
