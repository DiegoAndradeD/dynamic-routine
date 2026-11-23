package ui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		m.now = time.Time(msg)
		m.pollSync()
		return m, tick()
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.form != nil {
			m.handleFormKey(msg)
			return m, nil
		}
		return m.handleKey(msg.String())
	}
	return m, nil
}

func (m model) handleKey(key string) (tea.Model, tea.Cmd) {
	if m.confirmYes != nil {
		yes := m.confirmYes
		m.confirmYes, m.confirmText = nil, ""
		if key == "y" {
			yes(&m)
		}
		return m, nil
	}
	if m.help {
		m.help = false
		return m, nil
	}
	if m.welcome {
		if key == "q" {
			return m, tea.Quit
		}
		m.handleWelcomeKey(key)
		return m, nil
	}
	if m.picking {
		return m.handlePickerKey(key), nil
	}
	if m.trackPicking {
		m.handleTrackPickKey(key)
		return m, nil
	}

	switch key {
	case "q":
		return m, tea.Quit
	case "?":
		m.help = true
		return m, nil
	case "1", "2", "3", "4", "5":
		m.tab = tab(key[0] - '1')
		return m, nil
	case "r":
		if m.tab == tabRules {
			m.tab = tabDashboard
		} else {
			m.tab = tabRules
		}
		return m, nil
	case "p":
		m.tab = tabPlan
		return m, nil
	case "esc":
		m.tab = tabDashboard
		return m, nil
	case "t":
		if m.tab != tabPlan { // on the plan tab T adds a time track
			m.timerKey()
			return m, nil
		}
	case "l":
		m.logKey()
		return m, nil
	}

	switch m.tab {
	case tabDashboard:
		m.handleDashboardKey(key)
	case tabTime:
		m.handleTimeKey(key)
	case tabHistory:
		m.handleHistoryKey(key)
	case tabRules:
		m.handleRulesKey(key)
	case tabPlan:
		m.handlePlanKey(key)
	}
	return m, nil
}

// --- dashboard ---

func (m *model) handleDashboardKey(key string) {
	switch key {
	case "c", "enter":
		m.complete(routine.SessionFull)
	case "m":
		m.complete(routine.SessionMicro)
	case "s", "tab":
		if len(m.state.Queue) < 2 {
			m.flashError("There is only one activity in the queue.")
			return
		}
		m.picking = true
		m.pickCursor = 1
	case "u":
		m.mutate(func(s *routine.State) (string, error) {
			act, _, ok := s.UndoLast()
			if !ok {
				return "", refusal("Nothing to undo.")
			}
			return "Undone: " + act.Name + " is back on top of the queue", nil
		})
	}
}

func (m *model) complete(kind string) {
	m.mutate(func(s *routine.State) (string, error) {
		act, ok := s.Complete(kind, time.Now())
		if !ok {
			return "", refusal("The queue is empty. Press 5, then A to add an activity.")
		}
		label := "full"
		if kind == routine.SessionMicro {
			label = "micro"
		}
		return fmt.Sprintf("Done: %s (%s)", act.Name, label), nil
	})
}

func (m model) handlePickerKey(key string) model {
	n := len(m.state.Queue)
	switch key {
	case "esc", "s", "tab", "q":
		m.picking = false
	case "up", "k":
		if m.pickCursor > 0 {
			m.pickCursor--
		}
	case "down", "j":
		if m.pickCursor < n-1 {
			m.pickCursor++
		}
	case "home", "g":
		m.pickCursor = 0
	case "end", "G":
		m.pickCursor = n - 1
	case "enter":
		m.choose(m.pickCursor)
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if idx := int(key[0] - '1'); idx < n {
			m.choose(idx)
		}
	}
	return m
}

func (m *model) choose(index int) {
	m.picking = false
	if index == 0 {
		cur, _ := m.state.Current()
		m.flash("Kept on top: " + cur.Name)
		return
	}
	// The queue may have changed on another machine since the list was drawn:
	// pick by instance, not by position.
	want := m.state.Queue[index].InstanceID
	m.mutate(func(s *routine.State) (string, error) {
		for i, it := range s.Queue {
			if it.InstanceID == want {
				act, _ := s.Choose(i)
				return "Chosen: " + act.Name + " (the rest of the queue keeps its order)", nil
			}
		}
		return "", refusal("That item is no longer in the queue.")
	})
}

// --- timer and logging, available from every screen ---

// timerKey is "T": stop the running timer, or start one. On the Time tab it
// starts the selected track; elsewhere it asks which one if there are several.
func (m *model) timerKey() {
	tracks := m.state.Tracks
	if len(tracks) == 0 {
		m.flashError("No time tracks yet. Press 5, then T to add one.")
		return
	}
	switch {
	case m.tab == tabTime:
		t, _ := m.selectedTrack()
		m.toggleTimer(t.ID)
	case m.state.TimerRunning():
		m.toggleTimer(m.state.Timer.TrackID)
	case len(tracks) == 1:
		m.toggleTimer(tracks[0].ID)
	default:
		m.trackPicking = true
		m.trackPickCursor = 0
	}
}

func (m *model) handleTrackPickKey(key string) {
	n := len(m.state.Tracks)
	switch key {
	case "esc", "t", "q":
		m.trackPicking = false
	case "up", "k":
		m.trackPickCursor = clamp(m.trackPickCursor-1, n-1)
	case "down", "j":
		m.trackPickCursor = clamp(m.trackPickCursor+1, n-1)
	case "enter":
		m.trackPicking = false
		m.toggleTimer(m.state.Tracks[m.trackPickCursor].ID)
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if idx := int(key[0] - '1'); idx < n {
			m.trackPicking = false
			m.toggleTimer(m.state.Tracks[idx].ID)
		}
	}
}

// toggleTimer stops the timer if it is running on trackID, otherwise starts it
// on trackID (stopping any other first).
func (m *model) toggleTimer(trackID string) {
	now := time.Now()
	m.mutate(func(s *routine.State) (string, error) {
		track, ok := s.FindTrack(trackID)
		if !ok {
			return "", refusal("That time track no longer exists.")
		}
		if s.Timer != nil && s.Timer.TrackID == trackID {
			slice, _ := s.StopTimer(now)
			return fmt.Sprintf("Stopped %s: +%s this slice • today %s", track.Name, fmtDur(slice), todayProgress(s, track, now)), nil
		}
		prev, slice, err := s.StartTimer(trackID, now)
		if err != nil {
			return "", refusal(err.Error())
		}
		if prev.TrackID != "" {
			old, _ := s.FindTrack(prev.TrackID)
			return fmt.Sprintf("Switched from %s (+%s) to %s", old.Name, fmtDur(slice), track.Name), nil
		}
		return "Timer started: " + track.Name + " • today " + todayProgress(s, track, now), nil
	})
}

// todayProgress is "1h12/2h" (or just "1h12" when there is no daily goal).
func todayProgress(s *routine.State, t routine.Track, now time.Time) string {
	worked := fmtDur(s.WorkedOn(t.ID, now, now))
	if target := s.TargetOn(t.ID, now); target > 0 {
		return worked + "/" + fmtDur(target)
	}
	return worked
}

// logKey is "L": log time by hand, for the track and day selected in the Time
// tab, or the running / first track and today elsewhere.
func (m *model) logKey() {
	if m.tab == tabTime {
		if _, ok := m.selectedTrack(); ok {
			m.openLogForm(m.timeTrack, m.selectedDay())
			return
		}
	}
	idx := 0
	if m.state.Timer != nil {
		for i, t := range m.state.Tracks {
			if t.ID == m.state.Timer.TrackID {
				idx = i
			}
		}
	}
	m.openLogForm(idx, m.now)
}

// --- time tab ---

func (m *model) handleTimeKey(key string) {
	t, ok := m.selectedTrack()
	if !ok {
		return
	}
	items := m.state.ItemsOn(t.ID, m.selectedDay(), m.now)

	switch key {
	case "down", "j":
		switch {
		case m.entryCursor < len(items)-1:
			m.entryCursor++
		case m.dayCursor < 6:
			m.dayCursor++
			m.entryCursor = -1
		}
	case "up", "k":
		switch {
		case m.entryCursor >= 0:
			m.entryCursor--
		case m.dayCursor > 0:
			m.dayCursor--
			m.entryCursor = len(m.state.ItemsOn(t.ID, m.selectedDay(), m.now)) - 1
		}
	case "left", "[", "h":
		m.weekOffset--
		m.entryCursor = -1
	case "right", "]":
		if m.weekOffset < 0 {
			m.weekOffset++
			m.entryCursor = -1
		}
	case "g":
		m.weekOffset, m.dayCursor, m.entryCursor = 0, todayIndex(m.now), -1
	case "tab":
		m.timeTrack = (m.timeTrack + 1) % len(m.state.Tracks)
		m.entryCursor = -1
	case "shift+tab":
		m.timeTrack = (m.timeTrack + len(m.state.Tracks) - 1) % len(m.state.Tracks)
		m.entryCursor = -1
	case "a", "enter":
		m.openLogForm(m.timeTrack, m.selectedDay())
	case "d":
		if m.entryCursor < 0 || m.entryCursor >= len(items) {
			m.flashError("Select an entry first (↓ from the day), then press D.")
			return
		}
		it := items[m.entryCursor]
		if it.Running {
			m.flashError("This slice is still running: stop the timer first.")
			return
		}
		what := fmt.Sprintf("this %s entry on %s", fmtDur(it.Duration()), fmtDay(m.selectedDay()))
		m.askDelete(what, func(m *model) error {
			_, err := m.apply(func(s *routine.State) (string, error) {
				if !s.DeleteTimeItem(it.ID) {
					return "", refusal("That entry is already gone.")
				}
				return "", nil
			})
			m.entryCursor = -1
			return err
		})
	}
}

// --- history, rules ---

func (m *model) handleHistoryKey(key string) {
	switch key {
	case "up", "k":
		if m.histScroll > 0 {
			m.histScroll--
		}
	case "down", "j":
		if m.histScroll < m.historyMaxScroll() {
			m.histScroll++
		}
	case "pgup":
		m.histScroll = clamp(m.histScroll-m.historyRows(), m.historyMaxScroll())
	case "pgdown":
		m.histScroll = clamp(m.histScroll+m.historyRows(), m.historyMaxScroll())
	case "home", "g":
		m.histScroll = 0
	}
}

func (m *model) handleRulesKey(key string) {
	switch key {
	case "up", "k":
		if m.ruleCursor > 0 {
			m.ruleCursor--
		}
	case "down", "j":
		if m.ruleCursor < len(m.state.Rules)-1 {
			m.ruleCursor++
		}
	case "a":
		m.openRuleForm("")
	case "enter", "e":
		if m.ruleCursor < len(m.state.Rules) {
			m.openRuleForm(m.state.Rules[m.ruleCursor].ID)
		}
	case "d":
		if m.ruleCursor >= len(m.state.Rules) {
			return
		}
		r := m.state.Rules[m.ruleCursor]
		m.askDeleteFromPlan(fmt.Sprintf("the rule %q", r.Trigger), func(p *routine.Plan) {
			kept := p.Rules[:0]
			for _, x := range p.Rules {
				if x.ID != r.ID {
					kept = append(kept, x)
				}
			}
			p.Rules = kept
		})
	}
}
