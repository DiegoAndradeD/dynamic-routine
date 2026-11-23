package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
)

// The plan tab is one list with three sections: activities, time tracks and the
// settings row. planCursor runs through all of them.

func (m *model) handlePlanKey(key string) {
	acts := m.state.SortedActivities()
	tracks := m.state.Tracks
	last := len(acts) + len(tracks) // the settings row

	switch key {
	case "up", "k":
		if m.planCursor > 0 {
			m.planCursor--
		}
	case "down", "j":
		if m.planCursor < last {
			m.planCursor++
		}
	case "a":
		m.openActivityForm("")
	case "t":
		m.openTrackForm("")
	case "enter", "e":
		switch c := m.planCursor; {
		case c < len(acts):
			m.openActivityForm(acts[c].ID)
		case c < last:
			m.openTrackForm(tracks[c-len(acts)].ID)
		default:
			m.openSettingsForm()
		}
	case "d":
		switch c := m.planCursor; {
		case c < len(acts):
			a := acts[c]
			m.askDeleteFromPlan(fmt.Sprintf("%q (it leaves the queue, your history stays)", a.Name), func(p *routine.Plan) {
				kept := p.Activities[:0]
				for _, x := range p.Activities {
					if x.ID != a.ID {
						kept = append(kept, x)
					}
				}
				p.Activities = kept
			})
		case c < last:
			t := tracks[c-len(acts)]
			m.askDeleteFromPlan(fmt.Sprintf("the time track %q (its history stays)", t.Name), func(p *routine.Plan) {
				kept := p.Tracks[:0]
				for _, x := range p.Tracks {
					if x.ID != t.ID {
						kept = append(kept, x)
					}
				}
				p.Tracks = kept
			})
		}
	}
}

func (m model) viewPlan(w int) string {
	acts := m.state.SortedActivities()
	tracks := m.state.Tracks

	nameW := 10
	for _, a := range acts {
		nameW = max(nameW, len([]rune(a.Name)))
	}
	for _, t := range tracks {
		nameW = max(nameW, len([]rune(t.Name)))
	}
	nameW = clampInt(nameW, 10, clampInt(w/3, 12, 36))

	var rows []string // one string per list row, including the section titles
	cursorRow := 0
	add := func(selected bool, row string) {
		if selected {
			cursorRow = len(rows)
		}
		rows = append(rows, row)
	}

	rows = append(rows, sectionTitle("ACTIVITIES")+muted(clip("  done in a sitting and ticked off; they rotate through the queue", w-10)))
	if len(acts) == 0 {
		rows = append(rows, muted("  None yet. Press A to add your first one."))
	}
	for i, a := range acts {
		plain := clip(fmt.Sprintf("● %s  %s  %2d×/week  %4dm", pad(a.Name, nameW), pad(strings.ToUpper(a.Category), 9),
			a.TargetPerWeek, a.RecommendedMinutes), w-4)
		add(i == m.planCursor, m.planRow(i == m.planCursor, plain, m.catColor(a.Category), w))
	}

	rows = append(rows, "", sectionTitle("TIME TRACKS")+muted(clip("  time you spend, by timer or by logging it, with a goal per day or per week", w-13)))
	if len(tracks) == 0 {
		rows = append(rows, muted("  None yet. Press T to add one (work, studying, practice...)."))
	}
	for i, t := range tracks {
		idx := len(acts) + i
		plain := clip(fmt.Sprintf("◆ %s  %s  %s", pad(t.Name, nameW), pad(strings.ToUpper(t.Category), 9), goalText(t)), w-4)
		add(idx == m.planCursor, m.planRow(idx == m.planCursor, plain, m.catColor(t.Category), w))
	}

	idx := len(acts) + len(tracks)
	rows = append(rows, "", sectionTitle("SETTINGS"))
	add(m.planCursor == idx, m.planRow(m.planCursor == idx, fmt.Sprintf("  micro block: %dm", m.state.MicroMinutes), colorMuted, w))

	// scroll so the cursor stays visible
	avail := m.bodyHeight() - 2
	start, end := 0, len(rows)
	if avail > 5 && len(rows) > avail {
		start, end = window(len(rows), cursorRow, avail)
	}

	var b strings.Builder
	b.WriteString(bold(colorAmber).Render("PLAN") + muted(clip("  everything here is yours to change", w-6)) + "\n\n")
	b.WriteString(strings.Join(rows[start:end], "\n") + "\n")
	return b.String()
}

func (m model) planRow(selected bool, plain string, dot lipgloss.Color, w int) string {
	if selected {
		return lipgloss.NewStyle().Bold(true).Foreground(colorAmber).Background(colorSelBg).Width(w).Render("▸ " + plain)
	}
	row := "  " + plain
	row = strings.Replace(row, "●", fg(dot).Render("●"), 1)
	return strings.Replace(row, "◆", fg(dot).Render("◆"), 1)
}

func (m model) viewWelcome(w int) string {
	var b strings.Builder
	b.WriteString(bold(colorAmber).Render("WELCOME") + "\n\n")
	text := "This app helps you keep a routine that bends instead of breaking. It handles two kinds of things:\n\n" +
		"  ●  ACTIVITIES   things you do in a sitting and tick off (a workout, reading). They rotate through a " +
		"queue, so what you did least comes up first.\n\n" +
		"  ◆  TIME TRACKS  things you spend time on (work, studying). Run a timer or log the time afterwards, " +
		"and measure it against a goal per day or per week.\n\n" +
		"It starts with an example of each so you can see how it works. Everything can be changed from the " +
		"Plan tab (key 5). The Rules tab (key 4) is yours to fill with \"when X happens, do Y\" notes."
	b.WriteString(lipgloss.NewStyle().Width(clampInt(w-2, 40, 100)).Foreground(colorText).Render(text) + "\n\n")
	b.WriteString("  " + keyHint("E", "keep the example, I will edit it") + "\n")
	b.WriteString("  " + keyHint("N", "start empty and add my own") + "\n")
	return b.String()
}

func (m *model) handleWelcomeKey(key string) {
	switch key {
	case "e", "enter", "esc":
		m.welcome = false
		m.flash("Example loaded. Press 5 to make it yours.")
	case "n":
		m.welcome = false
		err := m.editPlan(func(p *routine.Plan) error { p.Activities, p.Tracks, p.Rules = nil, nil, nil; return nil })
		if err != nil {
			m.flashError(err.Error())
			return
		}
		m.tab = tabPlan
		m.openActivityForm("")
	}
}
