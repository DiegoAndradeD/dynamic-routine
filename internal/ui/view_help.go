package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpRow struct{ keys, what string }

// helpSections is the key reference shown by "?". Keep it in step with update.go.
var helpSections = []struct {
	title string
	rows  []helpRow
}{
	{"EVERYWHERE", []helpRow{
		{"1-5", "Dashboard, Time, History, Rules, Plan"},
		{"T", "start / stop a timer (on the Time tab: for the selected track)"},
		{"L", "log time you already spent (\"1h30 yesterday\")"},
		{"?", "this help"},
		{"Esc", "back to the dashboard"},
		{"Q  /  Ctrl+C", "quit"},
	}},
	{"DASHBOARD", []helpRow{
		{"C  /  Enter", "complete the activity on top; it goes to the back of the queue"},
		{"M", "micro block: counts as done, with a short session"},
		{"S  /  Tab", "choose another activity from the queue (1-9 picks directly)"},
		{"U", "undo the last completion"},
	}},
	{"TIME TAB", []helpRow{
		{"↑ ↓", "move through the days of the week, then through a day's entries"},
		{"← →", "previous / next week"},
		{"Tab  /  Shift+Tab", "next / previous time track"},
		{"A  /  Enter", "log time on the selected day"},
		{"D", "delete the selected entry (move onto it with ↓ first)"},
		{"G", "jump to today"},
	}},
	{"PLAN AND RULES TABS", []helpRow{
		{"A", "add an activity (Plan) or a rule (Rules)"},
		{"T", "add a time track (Plan tab)"},
		{"Enter  /  E", "edit the selected row"},
		{"D", "delete the selected row (asks first)"},
	}},
	{"FORMS", []helpRow{
		{"Tab  /  ↑ ↓", "next / previous field"},
		{"Enter", "next field; on the last one, save"},
		{"Ctrl+S", "save"},
		{"← → Home End", "move the cursor (or change a choice)"},
		{"Ctrl+U", "clear the field"},
		{"Esc", "cancel"},
	}},
}

func (m model) viewHelp(w int) string {
	var b strings.Builder
	b.WriteString(bold(colorAmber).Render("KEYS") + "\n")
	keyW := 20
	descW := clampInt(w-keyW-6, 20, 100)
	for _, sec := range helpSections {
		b.WriteString("\n" + sectionTitle(sec.title) + "\n")
		for _, r := range sec.rows {
			b.WriteString("  " + bold(colorAmber).Render(pad(r.keys, keyW)) +
				lipgloss.NewStyle().Width(descW).Foreground(colorText).Render(r.what) + "\n")
		}
	}
	return b.String()
}
