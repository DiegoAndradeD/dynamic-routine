package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
)

const wideLayout = 100 // below this width the dashboard is a single column

func (m model) viewDashboard(w int) string {
	if w < wideLayout {
		var b strings.Builder
		b.WriteString(m.renderNowCard(w) + "\n")
		b.WriteString(m.renderTracks(w) + "\n")
		b.WriteString(m.renderChips(w))
		b.WriteString(m.renderScoreboard(w))
		return b.String()
	}

	lw := w * 11 / 20
	rw := w - lw - 2
	nowCard := m.renderNowCard(lw)
	left := nowCard + "\n" + m.renderUpNext(lw, m.bodyHeight()-strings.Count(nowCard, "\n")-3)
	right := m.renderTracks(rw) + "\n" + m.renderScoreboard(rw)
	return lipgloss.JoinHorizontal(lipgloss.Top, block(left, lw), "  ", block(right, rw))
}

// --- what to do now ---

func (m model) renderNowCard(w int) string {
	act, ok := m.state.Current()
	if !ok {
		return box(w, colorBorder).Render(
			bold(colorAmber).Render("NOTHING IN THE QUEUE YET") + "\n" +
				muted("Press 5, then A to add your first activity (or T for a time track)."))
	}
	color := m.catColor(act.Category)
	stat := m.state.WeeklyStats(m.now)[act.ID]

	ratio := 0.0
	if act.TargetPerWeek > 0 {
		ratio = float64(stat.Total) / float64(act.TargetPerWeek)
	}
	barColor := colorAmber
	if stat.Total >= act.TargetPerWeek {
		barColor = colorEmerald
	}

	head := bold(color).Render("● NOW") + "  " + badge(strings.ToUpper(act.Category), color) +
		muted(fmt.Sprintf("  1 of %d in queue", len(m.state.Queue)))
	name := bold(lipgloss.Color("#FFFFFF")).Render(act.Name)
	detail := fmt.Sprintf("%s  •  %s %s %d/%d",
		muted(fmt.Sprintf("%d min recommended • micro %dm", act.RecommendedMinutes, m.state.MicroMinutes)),
		muted("week"), progressBar(ratio, clampInt(w/6, 8, 24), barColor), stat.Total, act.TargetPerWeek)

	return box(w, color).Render(head + "\n" + name + "\n" + detail)
}

// renderUpNext is the vertical list of what comes after the current activity.
func (m model) renderUpNext(w, maxLines int) string {
	q := m.state.Queue
	var b strings.Builder
	b.WriteString(sectionTitle("UP NEXT") + muted("  (S to pick another)") + "\n")
	if len(q) < 2 {
		b.WriteString(muted("  Nothing else in the queue.") + "\n")
		return b.String()
	}
	if maxLines <= 0 {
		maxLines = 12
	}
	maxLines = clampInt(maxLines, 3, 40)

	stats := m.state.WeeklyStats(m.now)
	nameW := clampInt(w-34, 12, 40)
	shown := 0
	for i := 1; i < len(q); i++ {
		if shown >= maxLines-1 && len(q)-i > 1 {
			break
		}
		act := m.state.Activities[q[i].ActivityID]
		st := stats[act.ID]
		wk := fmt.Sprintf("%d/%d wk", st.Total, act.TargetPerWeek)
		if st.Total >= act.TargetPerWeek {
			wk = fg(colorEmerald).Render(wk)
		} else {
			wk = muted(wk)
		}
		fmt.Fprintf(&b, "  %s %s %s  %s  %s\n", muted(fmt.Sprintf("%2d", i+1)), fg(m.catColor(act.Category)).Render("●"),
			pad(act.Name, nameW), muted(fmt.Sprintf("%3dm", act.RecommendedMinutes)), wk)
		shown++
	}
	if rest := len(q) - 1 - shown; rest > 0 {
		b.WriteString(muted(fmt.Sprintf("     … +%d more", rest)) + "\n")
	}
	return b.String()
}

// renderChips is the compact one-line-per-row version of the queue preview.
func (m model) renderChips(w int) string {
	q := m.state.Queue
	var b strings.Builder
	b.WriteString(sectionTitle("UP NEXT") + muted("  (S to pick another)") + "\n")
	if len(q) < 2 {
		b.WriteString(muted("  Nothing else in the queue.") + "\n")
		return b.String()
	}

	const preview = 8
	line := "  "
	for i := 1; i < len(q) && i <= preview; i++ {
		act := m.state.Activities[q[i].ActivityID]
		chip := muted(fmt.Sprintf("%d ", i+1)) + fg(m.catColor(act.Category)).Render("●") + " " +
			fg(colorText).Render(act.Name)
		if lipgloss.Width(line)+lipgloss.Width(chip)+5 > w && strings.TrimSpace(line) != "" {
			b.WriteString(line + "\n")
			line = "  "
		}
		if strings.TrimSpace(line) != "" {
			line += muted("  ›  ")
		}
		line += chip
	}
	if extra := len(q) - 1 - preview; extra > 0 {
		line += muted(fmt.Sprintf("  … +%d", extra))
	}
	b.WriteString(line + "\n")
	return b.String()
}

// --- time tracks ---

func (m model) renderTracks(w int) string {
	var b strings.Builder
	b.WriteString(sectionTitle("TIME") + muted("  (T timer • L log time)") + "\n")
	tracks := m.state.Tracks
	if len(tracks) == 0 {
		b.WriteString(muted("  No time tracks yet. Press 5, then T to add one (work, study...).") + "\n")
		return b.String()
	}
	compact := len(tracks) > 3 || (m.bodyHeight() > 0 && len(tracks)*5 > m.bodyHeight()/2)
	for _, t := range tracks {
		if compact {
			b.WriteString(m.trackLine(t, w) + "\n")
		} else {
			b.WriteString(m.trackCard(t, w) + "\n")
		}
	}
	return b.String()
}

// trackProgress is how far a track is toward its goal, and what to print for it.
type trackProgress struct {
	worked, target time.Duration // for the period the goal is about
	today          time.Duration
	ratio          float64
	label          string // "1h12 / 2h" or "this week 3h / 4h"
	running        bool
	met            bool
	dayOff         bool
}

func (m model) progressOf(t routine.Track) trackProgress {
	now := m.now
	p := trackProgress{today: m.state.WorkedOn(t.ID, now, now)}
	p.running = m.state.Timer != nil && m.state.Timer.TrackID == t.ID

	if t.Period == routine.PeriodWeek {
		wk := m.state.Week(t.ID, routine.WeekStart(now), now)
		p.worked, p.target = wk.Worked, wk.Target
		p.label = "week " + fmtDur(wk.Worked)
	} else {
		p.worked, p.target = p.today, m.state.TargetOn(t.ID, now)
		p.label = fmtDur(p.worked)
		p.dayOff = t.TargetMinutes > 0 && p.target == 0
	}
	if p.target > 0 {
		p.ratio = float64(p.worked) / float64(p.target)
		p.met = p.worked >= p.target
		p.label += " / " + fmtDur(p.target)
	}
	return p
}

func (p trackProgress) badge() string {
	switch {
	case p.running:
		return badge("● RUNNING", colorAmber)
	case p.met:
		return badge("✔ GOAL MET", colorEmerald)
	case p.dayOff:
		return badge("DAY OFF", colorCyan)
	case p.today == 0:
		return badge("NOT STARTED", colorMuted)
	}
	return badge("PAUSED", colorMuted)
}

func (p trackProgress) barColor() lipgloss.Color {
	if p.met {
		return colorEmerald
	}
	return colorAmber
}

// trackLine is a track on one line, for when there are many.
func (m model) trackLine(t routine.Track, w int) string {
	p := m.progressOf(t)
	nameW := clampInt(w/4, 10, 24)
	barW := clampInt(w-nameW-34, 6, 40)
	dot := fg(m.catColor(t.Category)).Render("◆")
	if p.running {
		dot = bold(colorAmber).Render("●")
	}
	return fmt.Sprintf("  %s %s  %s  %s", dot, pad(t.Name, nameW), progressBar(p.ratio, barW, p.barColor()), p.label)
}

// trackCard is a track as a boxed card with a bar and a line of details.
func (m model) trackCard(t routine.Track, w int) string {
	p := m.progressOf(t)
	border := colorBorder
	switch {
	case p.running:
		border = colorAmber
	case p.met:
		border = colorEmerald
	}

	head := fg(m.catColor(t.Category)).Render("◆") + " " + bold(lipgloss.Color("#FAFAFA")).Render(t.Name) + "  " + p.badge()
	clock := bold(lipgloss.Color("#FFFFFF")).Render(fmtClock(p.today))
	barW := clampInt(w-40, 8, 40)
	line := clock + muted("  ") + progressBar(p.ratio, barW, p.barColor()) + fmt.Sprintf(" %3d%%  ", int(p.ratio*100)) + p.label
	if p.target == 0 && !p.dayOff {
		line = clock + muted("  today, no goal set")
	}

	var info []string
	week := m.state.Week(t.ID, routine.WeekStart(m.now), m.now)
	switch {
	case t.Period == routine.PeriodWeek && week.Target > 0:
		info = append(info, muted(fmt.Sprintf("%s left this week • %s today", fmtLeft(week.Remaining()), fmtDur(p.today))))
	case t.Period == routine.PeriodDay:
		if left := m.state.RemainingToday(t.ID, m.now); left > 0 {
			end := m.now.Add(left)
			eta := end.Format("15:04")
			if !sameDay(end, m.now) {
				eta += " (+1d)"
			}
			info = append(info, muted(fmt.Sprintf("%s left • would finish ~%s if you continue now", fmtLeft(left), eta)))
		}
		if t.TargetMinutes > 0 {
			info = append(info, muted("week balance ")+balanceText(week.Balance))
		}
	}
	body := head + "\n" + line
	if len(info) > 0 {
		body += "\n" + strings.Join(info, muted(" • "))
	}
	return box(w, border).Render(body)
}

// balanceText colours a signed duration: green surplus, red deficit.
func balanceText(d time.Duration) string {
	switch {
	case d < 0:
		return bold(colorRose).Render(fmtSigned(d))
	case d > 0:
		return bold(colorEmerald).Render(fmtSigned(d))
	}
	return muted("0m")
}

// --- weekly scoreboard ---

func (m model) renderScoreboard(w int) string {
	var b strings.Builder
	b.WriteString("\n" + sectionTitle("THIS WEEK (MON - SUN)") + "\n")

	acts := m.state.SortedActivities()
	nameW := 8
	for _, a := range acts {
		if n := len([]rune(a.Name)); n > nameW {
			nameW = n
		}
	}
	for _, t := range m.state.Tracks {
		if n := len([]rune(t.Name)); n > nameW {
			nameW = n
		}
	}
	nameW = clampInt(nameW, 8, clampInt(w/3, 12, 28))
	barW := clampInt(w-nameW-22, 8, 36)

	// Time tracks first: they are the commitments that are not optional.
	for _, t := range m.state.Tracks {
		wk := m.state.Week(t.ID, routine.WeekStart(m.now), m.now)
		ratio, color := 0.0, colorAmber
		label := fmtDur(wk.Worked)
		if wk.Target > 0 {
			ratio = float64(wk.Worked) / float64(wk.Target)
			label += "/" + fmtDur(wk.Target)
			if wk.Worked >= wk.Target {
				color = colorEmerald
			}
		}
		fmt.Fprintf(&b, "  %s %s  %s  %s\n", fg(m.catColor(t.Category)).Render("◆"), pad(t.Name, nameW), progressBar(ratio, barW, color), label)
	}

	stats := m.state.WeeklyStats(m.now)
	for _, a := range acts {
		st := stats[a.ID]
		ratio := 0.0
		if a.TargetPerWeek > 0 {
			ratio = float64(st.Total) / float64(a.TargetPerWeek)
		}
		barColor, countStyle := colorAmber, fg(colorText)
		if st.Total >= a.TargetPerWeek {
			barColor, countStyle = colorEmerald, bold(colorEmerald)
		}
		micro := ""
		if st.Micro > 0 {
			micro = " " + fg(colorAmber).Render(fmt.Sprintf("(%d micro)", st.Micro))
		}
		fmt.Fprintf(&b, "  %s %s  %s  %s%s\n",
			fg(m.catColor(a.Category)).Render("●"), pad(a.Name, nameW),
			progressBar(ratio, barW, barColor), countStyle.Render(fmt.Sprintf("%d/%dx", st.Total, a.TargetPerWeek)), micro)
	}
	if len(acts) == 0 && len(m.state.Tracks) == 0 {
		b.WriteString(muted("  Nothing to count yet.") + "\n")
	}
	return b.String()
}
