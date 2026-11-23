package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// listRows is how many list rows fit below the header and above the footer.
func (m model) listRows(taken int) int {
	if m.bodyH == 0 {
		return 15
	}
	if rows := m.bodyH - taken; rows >= 5 {
		return rows
	}
	return 5
}

// window returns the [start,end) slice of a list of n rows that keeps cursor
// visible inside rows lines.
func window(n, cursor, rows int) (int, int) {
	if n <= rows {
		return 0, n
	}
	start := cursor - rows/2
	if start < 0 {
		start = 0
	}
	if start > n-rows {
		start = n - rows
	}
	return start, start + rows
}

func (m model) viewPicker(w int) string {
	var b strings.Builder
	b.WriteString(bold(colorAmber).Render("CHOOSE WHAT TO DO NOW") + "\n")
	b.WriteString(muted("The queue order (priority) still holds for the rest. The chosen one moves to the top,\nand nothing is logged as a miss for the ones left for later.") + "\n\n")

	q := m.state.Queue
	stats := m.state.WeeklyStats(m.now)
	start, end := window(len(q), m.pickCursor, m.listRows(6))

	if start > 0 {
		b.WriteString(muted(fmt.Sprintf("    ↑ %d above", start)) + "\n")
	}
	for i := start; i < end; i++ {
		act := m.state.Activities[q[i].ActivityID]
		st := stats[act.ID]

		tag := ""
		if i == 0 {
			tag = " (current)"
		}
		week := fmt.Sprintf("%d/%dx wk", st.Total, act.TargetPerWeek)
		plain := fmt.Sprintf("%2d  ● %s %4dm  %s%s", i+1, pad(act.Name, 24), act.RecommendedMinutes, pad(week, 9), tag)

		if i == m.pickCursor {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorAmber).Background(colorSelBg).
				Render("▸ "+plain) + "\n")
			continue
		}
		dot := fg(m.catColor(act.Category)).Render("●")
		line := "  " + strings.Replace(plain, "●", dot, 1)
		if st.Total >= act.TargetPerWeek {
			line = strings.Replace(line, week, fg(colorEmerald).Render(week), 1)
		}
		b.WriteString(line + "\n")
	}
	if end < len(q) {
		b.WriteString(muted(fmt.Sprintf("    ↓ %d below", len(q)-end)) + "\n")
	}
	return b.String()
}
