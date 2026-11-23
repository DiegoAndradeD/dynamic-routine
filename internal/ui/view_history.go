package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
)

const historyDays = 120

// historyEvent is one line of the timeline: an activity done, or time added.
type historyEvent struct {
	at   time.Time
	text string
}

// historyLines builds the timeline, newest day first. Each day gets a header
// with its totals; under it, what was done and the time that was spent.
func (m model) historyLines() []string {
	cutoff := routine.DayKey(m.now.AddDate(0, 0, -historyDays))
	days := map[string][]historyEvent{}
	done := map[string]int{}
	tracked := map[string]time.Duration{}

	for _, l := range m.state.Logs {
		key := routine.DayKey(l.CompletedAt)
		if key < cutoff {
			continue
		}
		act, ok := m.state.Activities[l.ActivityID]
		name := act.Name
		if !ok {
			name = l.ActivityID
		}
		kind := muted("full")
		if l.Type == routine.SessionMicro {
			kind = fg(colorAmber).Render("micro")
		}
		days[key] = append(days[key], historyEvent{l.CompletedAt, fmt.Sprintf("%s  %s %s  %s",
			muted(l.CompletedAt.Format("15:04")), fg(m.catColor(act.Category)).Render("●"), pad(name, 28), kind)})
		done[key]++
	}

	trackOf := func(id string) (string, string) {
		t, ok := m.state.FindTrack(id)
		if !ok {
			return id, "general"
		}
		return t.Name, t.Category
	}
	for _, s := range m.state.TimeSessions {
		key := routine.DayKey(s.Start)
		if key < cutoff {
			continue
		}
		name, cat := trackOf(s.TrackID)
		d := s.End.Sub(s.Start)
		days[key] = append(days[key], historyEvent{s.Start, fmt.Sprintf("%s  %s %s  %s  %s",
			muted(s.Start.Format("15:04")), fg(m.catColor(cat)).Render("◆"), pad(name, 28), fmtDur(d), muted("timer until "+s.End.Format("15:04")))})
		tracked[key] += d
	}
	for _, e := range m.state.TimeEntries {
		if e.Date < cutoff {
			continue
		}
		name, cat := trackOf(e.TrackID)
		d := time.Duration(e.Minutes) * time.Minute
		note := ""
		if e.Note != "" {
			note = muted("  “" + e.Note + "”")
		}
		days[e.Date] = append(days[e.Date], historyEvent{e.LoggedAt, fmt.Sprintf("%s  %s %s  %s  %s%s",
			muted("     "), fg(m.catColor(cat)).Render("◆"), pad(name, 28), fmtDur(d), muted("logged by hand"), note)})
		tracked[e.Date] += d
	}

	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	var lines []string
	for i, key := range keys {
		if i > 0 {
			lines = append(lines, "")
		}
		day, _ := time.ParseInLocation("2006-01-02", key, m.now.Location())
		var totals []string
		if n := done[key]; n > 0 {
			totals = append(totals, fmt.Sprintf("%d done", n))
		}
		if d := tracked[key]; d > 0 {
			totals = append(totals, fmtDur(d)+" tracked")
		}
		lines = append(lines, bold(colorMuted).Render(m.dayLabel(day))+muted("  "+strings.Join(totals, " • ")))

		evs := days[key]
		sort.SliceStable(evs, func(a, b int) bool { return evs[a].at.After(evs[b].at) })
		for _, ev := range evs {
			lines = append(lines, "  "+ev.text)
		}
	}
	return lines
}

func (m model) dayLabel(t time.Time) string {
	switch {
	case sameDay(t, m.now):
		return "TODAY"
	case sameDay(t, m.now.AddDate(0, 0, -1)):
		return "YESTERDAY"
	}
	return strings.ToUpper(fmtDate(t))
}

func (m model) historyRows() int { return m.listRows(4) }

func (m model) historyMaxScroll() int {
	if max := len(m.historyLines()) - m.historyRows(); max > 0 {
		return max
	}
	return 0
}

func (m model) viewHistory(w int) string {
	var b strings.Builder
	b.WriteString(bold(colorAmber).Render("HISTORY") +
		muted(fmt.Sprintf("  %d activities done • %d time entries", len(m.state.Logs), len(m.state.TimeSessions)+len(m.state.TimeEntries))) + "\n\n")

	lines := m.historyLines()
	if len(lines) == 0 {
		b.WriteString(muted("Nothing yet. Complete an activity or log some time and it shows up here.") + "\n")
		return b.String()
	}

	start := m.histScroll
	if max := m.historyMaxScroll(); start > max {
		start = max
	}
	end := start + m.historyRows()
	if end > len(lines) {
		end = len(lines)
	}
	if start > 0 {
		b.WriteString(muted("  ↑ newer") + "\n")
	}
	b.WriteString(strings.Join(lines[start:end], "\n") + "\n")
	if end < len(lines) {
		b.WriteString(muted("  ↓ older") + "\n")
	}
	return b.String()
}
