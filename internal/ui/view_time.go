package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
)

func (m model) viewTime(w int) string {
	if len(m.state.Tracks) == 0 {
		return bold(colorAmber).Render("TIME") + "\n\n" +
			muted("  Time tracks are for things you spend time on instead of ticking off: work, studying, practice...\n"+
				"  You can run a timer or log time afterwards (\"I spent 1h30 on this yesterday\").\n\n"+
				"  Press 5, then T to add your first one.") + "\n"
	}

	if w < 90 {
		return m.trackStrip(w) + "\n" + m.viewTrackDetail(w)
	}
	lw := clampInt(w/4, 26, 36)
	rw := w - lw - 2
	return lipgloss.JoinHorizontal(lipgloss.Top, block(m.trackList(lw), lw), "  ", block(m.viewTrackDetail(rw), rw))
}

// trackList is the left pane: every track with today's progress.
func (m model) trackList(w int) string {
	var b strings.Builder
	b.WriteString(bold(colorAmber).Render("TRACKS") + muted("  (Tab)") + "\n\n")
	for i, t := range m.state.Tracks {
		p := m.progressOf(t)
		dot := fg(m.catColor(t.Category)).Render("◆")
		if p.running {
			dot = bold(colorAmber).Render("●")
		}
		name := pad(t.Name, w-4)
		sub := muted("today ") + fmtDur(p.today)
		switch {
		case t.Period == routine.PeriodWeek && p.target > 0:
			sub = muted("today ") + fmtDur(p.today) + muted(" • week ") + fmtDur(p.worked) + muted("/") + fmtDur(p.target)
		case p.target > 0:
			sub = muted("today ") + p.label
		}
		if i == m.timeTrack {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorAmber).Background(colorSelBg).Width(w).Render("▸ "+name) + "\n")
		} else {
			b.WriteString("  " + dot + " " + pad(t.Name, w-4) + "\n")
		}
		b.WriteString("    " + sub + "\n\n")
	}
	return b.String()
}

// trackStrip is the one-line track selector for narrow terminals.
func (m model) trackStrip(w int) string {
	var parts []string
	for i, t := range m.state.Tracks {
		if i == m.timeTrack {
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0A0A0A")).Background(colorAmber).Render(" "+t.Name+" "))
		} else {
			parts = append(parts, muted(" "+t.Name+" "))
		}
	}
	return strings.Join(parts, " ") + muted("  (Tab)") + "\n"
}

func goalText(t routine.Track) string {
	if t.TargetMinutes <= 0 {
		return "no goal, just tracking"
	}
	d := fmtDur(time.Duration(t.TargetMinutes) * time.Minute)
	if t.Period == routine.PeriodWeek {
		return d + " per week"
	}
	days := "every day"
	if len(t.Days) > 0 {
		days = strings.Join(t.Days, " ")
	}
	return d + " per day on " + days
}

func (m model) viewTrackDetail(w int) string {
	t, ok := m.selectedTrack()
	if !ok {
		return ""
	}
	week := m.state.Week(t.ID, m.monday(), m.now)
	monday, sunday := week.Days[0].Date, week.Days[6].Date

	when := "this week"
	switch {
	case m.weekOffset == -1:
		when = "last week"
	case m.weekOffset < -1:
		when = fmt.Sprintf("%d weeks ago", -m.weekOffset)
	}

	var b strings.Builder
	b.WriteString(bold(m.catColor(t.Category)).Render("◆ "+strings.ToUpper(t.Name)) + muted("  "+goalText(t)) + "\n")
	b.WriteString(muted(fmt.Sprintf("week %d–%d %s (%s)", monday.Day(), sunday.Day(), monthShort[sunday.Month()], when)) +
		muted("   ←/→ to browse weeks") + "\n\n")

	barW := clampInt(w-52, 6, 50)
	for i, d := range week.Days {
		b.WriteString(m.dayRow(t, i, d, barW) + "\n")
	}

	b.WriteString("\n")
	switch {
	case t.Period == routine.PeriodWeek && week.Target > 0:
		fmt.Fprintf(&b, "  Week  %s / %s    %s\n", bold(lipgloss.Color("#FFFFFF")).Render(fmtDur(week.Worked)), fmtDur(week.Target),
			muted(fmtLeft(week.Remaining())+" left"))
	case t.Period == routine.PeriodDay && week.Target > 0:
		fmt.Fprintf(&b, "  Week  %s / %s    Balance through yesterday  %s\n", bold(lipgloss.Color("#FFFFFF")).Render(fmtDur(week.Worked)),
			fmtDur(week.Target), balanceText(week.Balance))
		b.WriteString("  " + muted("Today's pending goal is not a debt until the day ends.") + "\n")
	default:
		fmt.Fprintf(&b, "  Week  %s\n", bold(lipgloss.Color("#FFFFFF")).Render(fmtDur(week.Worked)))
	}

	b.WriteString("\n" + m.dayEntries(t) + "\n")
	return b.String()
}

func (m model) dayRow(t routine.Track, i int, d routine.TrackDay, barW int) string {
	selected := i == m.dayCursor

	var status string
	barColor := colorAmber
	switch {
	case t.Period == routine.PeriodWeek || t.TargetMinutes == 0:
		if d.Worked == 0 && d.IsFuture {
			status = ""
		} else if d.Worked > 0 {
			status = fg(colorText).Render(fmtDur(d.Worked))
		}
	case d.Target == 0 && d.Worked == 0:
		status = muted("day off")
	case d.Done():
		status, barColor = fg(colorEmerald).Render("✔"), colorEmerald
		if extra := d.Worked - d.Target; extra > 0 {
			status += " " + fg(colorEmerald).Render(fmtSigned(extra))
		}
	case d.IsToday:
		status = fg(colorAmber).Render("● today, " + fmtLeft(d.Target-d.Worked) + " left")
	case d.IsFuture:
		status = ""
	default:
		status, barColor = fg(colorRose).Render(fmtSigned(d.Worked-d.Target)), colorRose
	}

	ratio := 0.0
	switch {
	case d.Target > 0:
		ratio = float64(d.Worked) / float64(d.Target)
	case d.Worked > 0 && t.Period == routine.PeriodDay:
		ratio = 1
	case d.Worked > 0 && t.TargetMinutes > 0:
		// weekly goal: a day's bar is its share of the weekly target
		ratio = float64(d.Worked) / float64(time.Duration(t.TargetMinutes)*time.Minute)
	}

	cursor, label := "  ", muted(fmtDay(d.Date))
	if selected {
		cursor = bold(colorAmber).Render("▸ ")
		label = bold(colorAmber).Render(fmtDay(d.Date))
		if m.entryCursor >= 0 {
			cursor = muted("▸ ")
		}
	}
	times := fmt.Sprintf("%5s", fmtDur(d.Worked))
	if d.Target > 0 {
		times += fmt.Sprintf(" / %-5s", fmtDur(d.Target))
	}
	if d.IsFuture {
		times = muted(times)
	}
	return fmt.Sprintf("%s%s  %s  %-14s %s", cursor, label, progressBar(ratio, barW, barColor), times, status)
}

// dayEntries lists what makes up the selected day, with a cursor for deleting.
func (m model) dayEntries(t routine.Track) string {
	day := m.selectedDay()
	var b strings.Builder
	b.WriteString(sectionTitle(fmt.Sprintf("%s · %s", strings.ToUpper(t.Name), strings.ToUpper(fmtDay(day)))) +
		muted("   Enter/A log time here") + "\n")

	items := m.state.ItemsOn(t.ID, day, m.now)
	if len(items) == 0 {
		b.WriteString(muted("  Nothing logged this day. Ran no timer? Press A and say how long you spent.") + "\n")
		return b.String()
	}
	for i, it := range items {
		cursor := "  "
		if i == m.entryCursor {
			cursor = bold(colorAmber).Render("▸ ")
		}
		var what string
		if it.Manual {
			what = fmt.Sprintf("%s  %s", muted("logged by hand"), fmtDur(it.Duration()))
			if it.Note != "" {
				what += muted("  “" + it.Note + "”")
			}
		} else {
			end := it.End.Format("15:04")
			note := ""
			if it.Running {
				end, note = "now", fg(colorAmber).Render("  ● running")
			}
			what = fmt.Sprintf("%s → %s  %s%s", it.Start.Format("15:04"), end, fmtDur(it.Duration()), note)
		}
		b.WriteString(cursor + what + "\n")
	}
	return b.String()
}

func (m model) viewTrackPicker(w int) string {
	var b strings.Builder
	b.WriteString(bold(colorAmber).Render("START A TIMER FOR...") + "\n\n")
	for i, t := range m.state.Tracks {
		p := m.progressOf(t)
		plain := fmt.Sprintf("%2d  ◆ %s  %s", i+1, pad(t.Name, clampInt(w/3, 12, 30)), p.label)
		if i == m.trackPickCursor {
			b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorAmber).Background(colorSelBg).Render("▸ "+plain) + "\n")
			continue
		}
		b.WriteString("  " + strings.Replace(plain, "◆", fg(m.catColor(t.Category)).Render("◆"), 1) + "\n")
	}
	return b.String()
}
