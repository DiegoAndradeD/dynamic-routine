package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	weekdayShort = [...]string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}
	monthShort   = [...]string{"", "jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}
)

// fmtDur renders a duration as "2h05" or "45m".
func fmtDur(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	mins := int(d / time.Minute)
	if mins < 60 {
		return fmt.Sprintf("%dm", mins)
	}
	return fmt.Sprintf("%dh%02d", mins/60, mins%60)
}

func fmtSigned(d time.Duration) string {
	if d < 0 {
		return "-" + fmtDur(-d)
	}
	if d > 0 {
		return "+" + fmtDur(d)
	}
	return "0m"
}

func fmtClock(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int(d / time.Second)
	return fmt.Sprintf("%02d:%02d:%02d", s/3600, (s/60)%60, s%60)
}

func fmtDay(t time.Time) string {
	return fmt.Sprintf("%s %02d", weekdayShort[t.Weekday()], t.Day())
}

func fmtDate(t time.Time) string {
	return fmt.Sprintf("%s, %d %s", weekdayShort[t.Weekday()], t.Day(), monthShort[t.Month()])
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// pad right-pads s with spaces to w visible cells (and truncates if longer).
func pad(s string, w int) string {
	r := []rune(s)
	if len(r) > w {
		return string(r[:w-1]) + "…"
	}
	if gap := w - lipgloss.Width(s); gap > 0 {
		return s + fmt.Sprintf("%*s", gap, "")
	}
	return s
}

// fmtLeft renders time still missing, rounded up to the minute so that
// worked + left always adds up to the target on screen.
func fmtLeft(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}
	return fmtDur((d + time.Minute - 1) / time.Minute * time.Minute)
}

// fmtInput renders minutes the way the user would type them: "", "45", "2h", "1h30".
func fmtInput(minutes int) string {
	switch {
	case minutes <= 0:
		return ""
	case minutes < 60:
		return fmt.Sprintf("%d", minutes)
	case minutes%60 == 0:
		return fmt.Sprintf("%dh", minutes/60)
	}
	return fmt.Sprintf("%dh%02d", minutes/60, minutes%60)
}

// parseDay reads "today", "yesterday" or YYYY-MM-DD.
func parseDay(s string, now time.Time) (time.Time, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "today":
		return now, nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	}
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot read %q as a day (use today, yesterday or YYYY-MM-DD)", s)
	}
	return t, nil
}

// dayWord is how a form shows a date: today, yesterday or YYYY-MM-DD.
func dayWord(t, now time.Time) string {
	switch {
	case sameDay(t, now):
		return "today"
	case sameDay(t, now.AddDate(0, 0, -1)):
		return "yesterday"
	}
	return t.Format("2006-01-02")
}

// fitHeight cuts or pads s to exactly h lines (h <= 0 leaves it alone).
func fitHeight(s string, h int) string {
	if h <= 0 {
		return s
	}
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func clampInt(v, lo, hi int) int {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}

// block pads s to a fixed width, so columns can be joined side by side.
func block(s string, w int) string {
	return lipgloss.NewStyle().Width(w).Render(strings.TrimRight(s, "\n"))
}

// clip shortens plain text to w cells, ending in an ellipsis.
func clip(s string, w int) string {
	if w < 1 {
		return ""
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}
