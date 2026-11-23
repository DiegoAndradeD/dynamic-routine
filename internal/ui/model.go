// Package ui is the Bubble Tea terminal interface.
package ui

import (
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
	"github.com/DiegoAndradeD/dynamic-routine/internal/store"
)

type tab int

const (
	tabDashboard tab = iota
	tabTime
	tabHistory
	tabRules
	tabPlan
)

var tabNames = []string{"Dashboard", "Time", "History", "Rules", "Plan"}

const (
	flashDuration     = 5 * time.Second
	syncCheckEvery    = 2 * time.Second  // how often to look for changes made by another machine
	syncRetryAfter    = 15 * time.Second // wait before retrying a reload that failed
	conflictCheckEach = 30 * time.Second
)

type tickMsg time.Time

type model struct {
	state routine.State
	store *store.Store

	tab tab

	picking         bool // "choose what to do" overlay on the dashboard
	pickCursor      int
	trackPicking    bool // "which timer to start" overlay
	trackPickCursor int

	ruleCursor int
	planCursor int // activities, then time tracks, then the settings row
	histScroll int

	// Time tab: which track, which week (0 = this one, -1 = last...), and the
	// cursor inside it (entryCursor -1 = on the day itself).
	timeTrack   int
	weekOffset  int
	dayCursor   int // Monday = 0
	entryCursor int

	form        *form  // open editor, if any
	welcome     bool   // first-run screen
	help        bool   // key reference overlay
	confirmText string // pending yes/no question, if any
	confirmYes  func(*model)

	width, height int
	bodyH         int // lines available to the current screen
	now           time.Time

	flashMsg   string
	flashIsErr bool
	flashUntil time.Time

	nextSyncCheck     time.Time
	nextConflictCheck time.Time
	conflicts         int // sync conflict copies sitting next to the state file
}

// Run loads the state from the store and runs the TUI until the user quits.
func Run(st *store.Store, firstRun bool) error {
	state, err := st.Load()
	if err != nil {
		return err
	}
	_, err = tea.NewProgram(newModel(st, state, firstRun), tea.WithAltScreen()).Run()
	return err
}

func newModel(st *store.Store, state routine.State, firstRun bool) model {
	now := time.Now()
	return model{
		state:       state,
		store:       st,
		now:         now,
		dayCursor:   todayIndex(now),
		entryCursor: -1,
		conflicts:   len(st.Conflicts()),
		welcome:     firstRun,
	}
}

// todayIndex maps a date to its position in the Monday-first week.
func todayIndex(t time.Time) int { return (int(t.Weekday()) + 6) % 7 }

func (m model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *model) flash(msg string) {
	m.flashMsg, m.flashIsErr = msg, false
	m.flashUntil = time.Now().Add(flashDuration)
}

func (m *model) flashError(msg string) {
	m.flashMsg, m.flashIsErr = msg, true
	m.flashUntil = time.Now().Add(flashDuration)
}

// refusal is returned by a mutation that does not apply to the current state
// ("nothing to undo", "that day has not happened yet"). The state is refreshed,
// but it is shown as a notice instead of an I/O failure.
type refusal string

func (r refusal) Error() string { return string(r) }

// apply runs fn on the freshest copy of the state on disk and saves it. Going
// through the store (instead of saving our in-memory copy) is what keeps changes
// another machine saved meanwhile. It returns fn's message, or its refusal / the
// I/O error as an error.
func (m *model) apply(fn func(*routine.State) (string, error)) (string, error) {
	var msg string
	st, err := m.store.Update(func(s *routine.State) error {
		var err error
		msg, err = fn(s)
		return err
	})

	var r refusal
	switch {
	case err == nil:
		m.adopt(st)
		return msg, nil
	case errors.As(err, &r):
		m.adopt(st)
		return "", errors.New(string(r))
	default:
		return "", fmt.Errorf("not saved: %w", err)
	}
}

// mutate is apply for actions that report to the flash line.
func (m *model) mutate(fn func(*routine.State) (string, error)) {
	if msg, err := m.apply(fn); err != nil {
		m.flashError(err.Error())
	} else {
		m.flash(msg)
	}
}

// adopt replaces the in-memory state and keeps the cursors inside it.
func (m *model) adopt(st routine.State) {
	m.state = st
	m.pickCursor = clamp(m.pickCursor, len(st.Queue)-1)
	m.ruleCursor = clamp(m.ruleCursor, len(st.Rules)-1)
	m.planCursor = clamp(m.planCursor, len(st.Activities)+len(st.Tracks)) // last row is the settings row
	m.timeTrack = clamp(m.timeTrack, len(st.Tracks)-1)
	m.trackPickCursor = clamp(m.trackPickCursor, len(st.Tracks)-1)
}

func clamp(v, max int) int {
	if v > max {
		v = max
	}
	if v < 0 {
		v = 0
	}
	return v
}

// pollSync picks up what the other machine saved: the sync tool replaces the
// file, we notice and reload it.
func (m *model) pollSync() {
	if m.now.Before(m.nextSyncCheck) {
		return
	}
	m.nextSyncCheck = m.now.Add(syncCheckEvery)

	if m.store.Changed() {
		st, err := m.store.Load()
		if err != nil {
			m.flashError(fmt.Sprintf("Could not reload %s: %v", m.store.Path(), err))
			m.nextSyncCheck = m.now.Add(syncRetryAfter)
			return
		}
		m.adopt(st)
		m.flash("Updated from the state file (changed on another machine)")
	}

	if !m.now.Before(m.nextConflictCheck) {
		m.nextConflictCheck = m.now.Add(conflictCheckEach)
		m.conflicts = len(m.store.Conflicts())
	}
}

// contentWidth is the whole terminal: the layouts adapt to it.
func (m model) contentWidth() int {
	switch w := m.width; {
	case w == 0:
		return 80
	case w < 50:
		return 50
	default:
		return w
	}
}

// bodyHeight is how many lines the current screen may use (0 = unknown).
func (m model) bodyHeight() int { return m.bodyH }

// catColor is the colour of a category, as set in the plan.
func (m model) catColor(cat string) lipgloss.Color {
	if hex, ok := m.state.Categories[cat]; ok {
		return lipgloss.Color(hex)
	}
	return colorCyan
}

// monday is the first day of the week shown in the Time tab.
func (m model) monday() time.Time {
	return routine.WeekStart(m.now).AddDate(0, 0, 7*m.weekOffset)
}

func (m model) selectedDay() time.Time { return m.monday().AddDate(0, 0, m.dayCursor) }

// selectedTrack is the track shown in the Time tab.
func (m model) selectedTrack() (routine.Track, bool) {
	if m.timeTrack < 0 || m.timeTrack >= len(m.state.Tracks) {
		return routine.Track{}, false
	}
	return m.state.Tracks[m.timeTrack], true
}
