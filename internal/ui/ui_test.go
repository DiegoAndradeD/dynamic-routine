package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
	"github.com/DiegoAndradeD/dynamic-routine/internal/store"
)

func newTestModel(t *testing.T) model {
	t.Helper()
	st := store.New(filepath.Join(t.TempDir(), "state.json"), nil)
	state, err := routine.NewState(routine.DefaultPlan())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Save(state); err != nil {
		t.Fatal(err)
	}
	m := newModel(st, state, false)
	m.width, m.height = 100, 40
	return m
}

func press(m model, keys ...string) model {
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "home":
			msg = tea.KeyMsg{Type: tea.KeyHome}
		case "end":
			msg = tea.KeyMsg{Type: tea.KeyEnd}
		case "delete":
			msg = tea.KeyMsg{Type: tea.KeyDelete}
		case "left":
			msg = tea.KeyMsg{Type: tea.KeyLeft}
		case "right":
			msg = tea.KeyMsg{Type: tea.KeyRight}
		case "backspace":
			msg = tea.KeyMsg{Type: tea.KeyBackspace}
		case "ctrl+u":
			msg = tea.KeyMsg{Type: tea.KeyCtrlU}
		case "ctrl+s":
			msg = tea.KeyMsg{Type: tea.KeyCtrlS}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		next, _ := m.Update(msg)
		m = next.(model)
	}
	return m
}

func TestChooseAsManyTimesAsWanted(t *testing.T) {
	m := newTestModel(t)
	before := len(m.state.Queue)
	third := m.state.Queue[2].ActivityID

	m = press(m, "s", "down", "enter") // cursor starts at 1, move to 2, choose
	if m.picking {
		t.Fatal("picker should close after choosing")
	}
	if m.state.Queue[0].ActivityID != third {
		t.Fatalf("top = %s, want %s", m.state.Queue[0].ActivityID, third)
	}
	// choose again and again: no limit, nothing logged
	for i := 0; i < 10; i++ {
		m = press(m, "s", "enter")
	}
	if len(m.state.Queue) != before || len(m.state.Logs) != 0 {
		t.Fatalf("queue/logs changed unexpectedly: %d %d", len(m.state.Queue), len(m.state.Logs))
	}
	// direct pick by number
	want := m.state.Queue[4].ActivityID
	m = press(m, "s", "5")
	if m.state.Queue[0].ActivityID != want {
		t.Fatal("number key should pick directly")
	}
}

func TestCompleteUndoAndPersistence(t *testing.T) {
	m := newTestModel(t)
	top := m.state.Queue[0].ActivityID
	m = press(m, "c")
	if len(m.state.Logs) != 1 || m.state.Queue[0].ActivityID == top && m.state.Queue[1].ActivityID == top {
		t.Fatal("complete did not rotate queue")
	}
	reloaded, err := m.store.Load()
	if err != nil || len(reloaded.Logs) != 1 {
		t.Fatalf("not persisted: %v %d", err, len(reloaded.Logs))
	}
	m = press(m, "u")
	if len(m.state.Logs) != 0 || m.state.Queue[0].ActivityID != top {
		t.Fatal("undo did not restore")
	}
}

// Another machine saves while this one is open: its change must show up, and
// our next action must not erase it.
func TestSyncFromAnotherMachine(t *testing.T) {
	m := newTestModel(t)
	other := store.New(m.store.Path(), nil)
	if _, err := other.Update(func(s *routine.State) error {
		s.Complete(routine.SessionFull, time.Now().Add(-time.Hour))
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(m.state.Logs) != 0 {
		t.Fatal("precondition: this model has not seen the other save")
	}

	m.now = time.Now().Add(time.Minute) // past the poll interval
	m.pollSync()
	if len(m.state.Logs) != 1 {
		t.Fatalf("logs = %d, want the other machine's completion", len(m.state.Logs))
	}

	// now the reverse: act without having polled
	other.Update(func(s *routine.State) error {
		s.Complete(routine.SessionMicro, time.Now().Add(-30*time.Minute))
		return nil
	})
	m = press(m, "c")
	if len(m.state.Logs) != 3 {
		t.Fatalf("logs = %d, want 3 (nothing lost)", len(m.state.Logs))
	}
}

func TestRefusalIsShownAsErrorAndStateRefreshed(t *testing.T) {
	m := newTestModel(t)
	m = press(m, "u") // nothing to undo
	if !m.flashIsErr || m.flashMsg != "Nothing to undo." {
		t.Fatalf("flash = %q err=%v", m.flashMsg, m.flashIsErr)
	}
}

func TestCategoryColourComesFromThePlan(t *testing.T) {
	m := newTestModel(t)
	m.state.Categories["body"] = "#112233"
	if got := string(m.catColor("body")); got != "#112233" {
		t.Fatalf("colour = %s", got)
	}
	if string(m.catColor("unknown")) == "" {
		t.Fatal("unknown categories need a fallback colour")
	}
}

func TestAddActivityFromThePlanTab(t *testing.T) {
	m := newTestModel(t)
	before := len(m.state.Activities)

	m = press(m, "5", "a")
	if m.form == nil {
		t.Fatal("A on the plan tab must open the form")
	}
	// name, category, times (default 3 -> 5), minutes (default 45 -> 30)
	m = press(m, "Guitar", "enter", "leisure", "enter", "backspace", "5", "enter", "backspace", "backspace", "30", "enter")
	if m.form != nil {
		t.Fatalf("form should close after saving, err=%q", m.form.err)
	}
	a, ok := m.state.Activities["guitar"]
	if !ok || a.TargetPerWeek != 5 || a.RecommendedMinutes != 30 || a.Category != "leisure" {
		t.Fatalf("saved activity = %+v ok=%v", a, ok)
	}
	if len(m.state.Activities) != before+1 {
		t.Fatal("activity count")
	}
	count := 0
	for _, q := range m.state.Queue {
		if q.ActivityID == "guitar" {
			count++
		}
	}
	if count != 5 {
		t.Fatalf("queue has %d guitar entries, want 5", count)
	}
	reloaded, _ := m.store.Load()
	if _, ok := reloaded.Activities["guitar"]; !ok {
		t.Fatal("not persisted")
	}
}

func TestFormShowsValidationErrorAndStaysOpen(t *testing.T) {
	m := newTestModel(t)
	m = press(m, "5", "a", "ctrl+s") // no name
	if m.form == nil || m.form.err == "" {
		t.Fatalf("expected the form to stay open with an error, form=%v", m.form)
	}
	if !strings.Contains(m.View(), m.form.err) {
		t.Fatal("the error must be visible")
	}
	m = press(m, "esc")
	if m.form != nil {
		t.Fatal("Esc cancels")
	}
}

func TestNumericFieldsIgnoreLetters(t *testing.T) {
	m := newTestModel(t)
	m = press(m, "5", "a", "tab", "tab", "backspace", "x7q")
	if got := m.form.fields[2].value; got != "7" {
		t.Fatalf("value = %q", got)
	}
}

func TestEditAndDeleteWithConfirmation(t *testing.T) {
	m := newTestModel(t)
	first := m.state.SortedActivities()[0]

	m = press(m, "5", "enter", "backspace", "!", "ctrl+s") // rename the first one
	if got := m.state.Activities[first.ID].Name; got != first.Name[:len(first.Name)-1]+"!" {
		t.Fatalf("name = %q", got)
	}

	m = press(m, "d", "x") // any key but y cancels
	if _, ok := m.state.Activities[first.ID]; !ok || m.confirmYes != nil {
		t.Fatal("delete must be cancelled")
	}
	m = press(m, "d")
	if m.confirmYes == nil || !strings.Contains(m.View(), "Delete") {
		t.Fatal("expected a visible confirmation")
	}
	m = press(m, "y")
	if _, ok := m.state.Activities[first.ID]; ok {
		t.Fatal("activity should be gone")
	}
	for _, q := range m.state.Queue {
		if q.ActivityID == first.ID {
			t.Fatal("deleted activity is still in the queue")
		}
	}
}

func TestTimerToggleAndSwitch(t *testing.T) {
	m := newTestModel(t)
	if len(m.state.Tracks) != 2 {
		t.Fatalf("expected the two example tracks, got %d", len(m.state.Tracks))
	}

	m = press(m, "t") // several tracks: asks which
	if !m.trackPicking {
		t.Fatal("T with several tracks should ask which one")
	}
	if !strings.Contains(m.View(), "START A TIMER") {
		t.Fatal("picker not shown")
	}
	m = press(m, "1")
	if m.state.Timer == nil || m.state.Timer.TrackID != m.state.Tracks[0].ID {
		t.Fatalf("timer = %+v", m.state.Timer)
	}
	reloaded, _ := m.store.Load()
	if reloaded.Timer == nil {
		t.Fatal("a running timer must be persisted, so a restart does not lose it")
	}
	if !strings.Contains(m.View(), m.state.Tracks[0].Name) {
		t.Fatal("the header should show the running timer")
	}

	m = press(m, "t") // running: T stops it
	if m.state.Timer != nil || len(m.state.TimeSessions) != 1 {
		t.Fatalf("timer=%v sessions=%d", m.state.Timer, len(m.state.TimeSessions))
	}

	// on the Time tab T starts the selected track, and switching stops the other
	m = press(m, "2", "t")
	first := m.state.Tracks[0].ID
	if m.state.Timer == nil || m.state.Timer.TrackID != first {
		t.Fatal("T on the Time tab starts the selected track")
	}
	m = press(m, "tab", "t")
	if m.state.Timer.TrackID != m.state.Tracks[1].ID || len(m.state.TimeSessions) != 2 {
		t.Fatalf("switching: timer=%+v sessions=%d", m.state.Timer, len(m.state.TimeSessions))
	}
}

func TestLogTimeAfterTheFact(t *testing.T) {
	m := newTestModel(t)
	id := m.state.Tracks[0].ID

	m = press(m, "l")
	if m.form == nil || m.form.kind != formLog {
		t.Fatal("L opens the log form")
	}
	// Track (first), Day=today -> yesterday, Time, Note
	m = press(m, "up", "ctrl+u", "yesterday", "enter", "1h30", "enter", "planning", "ctrl+s")
	if m.form != nil {
		t.Fatalf("form should close, err=%q", m.form.err)
	}
	yesterday := time.Now().AddDate(0, 0, -1)
	if got := m.state.WorkedOn(id, yesterday, time.Now()); got != 90*time.Minute {
		t.Fatalf("worked yesterday = %v", got)
	}
	if m.state.TimeEntries[0].Note != "planning" {
		t.Fatalf("entry = %+v", m.state.TimeEntries[0])
	}

	// mistakes are explained, and the form stays open
	for in, want := range map[string]string{"abc": "cannot read", "0": "more than zero"} {
		m = press(m, "l", "ctrl+u", in, "ctrl+s") // the form opens on the time field
		if m.form == nil || !strings.Contains(m.form.err, want) {
			t.Fatalf("input %q: form=%v", in, m.form)
		}
		m = press(m, "esc")
	}
	m = press(m, "l", "up", "ctrl+u", "2099-01-01", "enter", "30", "ctrl+s")
	if m.form == nil || !strings.Contains(m.form.err, "not happened") {
		t.Fatalf("future day must be refused: %+v", m.form)
	}
}

func TestTimeTabBrowseAndDelete(t *testing.T) {
	m := newTestModel(t)
	id := m.state.Tracks[0].ID
	now := time.Now()
	m.mutate(func(s *routine.State) (string, error) {
		_, err := s.LogTime(id, now, 45, "first", now)
		return "", err
	})

	m = press(m, "2", "g")
	if m.dayCursor != todayIndex(now) {
		t.Fatalf("G should jump to today, cursor=%d", m.dayCursor)
	}
	m = press(m, "down") // onto the entry (or the next day, if today is Sunday)
	if todayIndex(now) < 6 && m.entryCursor != 0 {
		t.Fatalf("entry cursor = %d", m.entryCursor)
	}
	if m.entryCursor == 0 {
		m = press(m, "d", "y")
		if len(m.state.TimeEntries) != 0 {
			t.Fatal("the entry should be deleted")
		}
	}

	m = press(m, "left")
	if m.weekOffset != -1 || !strings.Contains(m.View(), "last week") {
		t.Fatal("← browses to last week")
	}
	m = press(m, "right", "right")
	if m.weekOffset != 0 {
		t.Fatal("cannot browse into the future")
	}

	// log into last week from the Time tab: the form starts on that day
	m = press(m, "left", "a")
	if m.form == nil || m.form.fields[1].value == "today" {
		t.Fatalf("form day = %+v", m.form)
	}
}

func TestTrackFormAddEditDelete(t *testing.T) {
	m := newTestModel(t)
	n := len(m.state.Tracks)

	m = press(m, "5", "t")
	if m.form == nil || m.form.kind != formTrack {
		t.Fatal("T on the Plan tab adds a time track")
	}
	// name, category, goal is (right -> per week), goal, days
	m = press(m, "Studying", "enter", "focus", "enter", "right", "enter", "ctrl+u", "6h", "ctrl+s")
	if m.form != nil {
		t.Fatalf("err=%q", m.form.err)
	}
	tr, ok := m.state.FindTrack("studying")
	if !ok || tr.Period != routine.PeriodWeek || tr.TargetMinutes != 360 || len(m.state.Tracks) != n+1 {
		t.Fatalf("track = %+v ok=%v", tr, ok)
	}

	// a bad goal is explained
	m = press(m, "5")
	for i := 0; i < len(m.state.Activities)+n; i++ {
		m = press(m, "down")
	}
	m = press(m, "enter", "tab", "tab", "tab", "ctrl+u", "soon", "ctrl+s")
	if m.form == nil || !strings.Contains(m.form.err, "cannot read") {
		t.Fatalf("form = %+v", m.form)
	}
	m = press(m, "esc")

	// delete it (the confirmation names it)
	m = press(m, "d")
	if m.confirmYes == nil || !strings.Contains(m.View(), "time track") {
		t.Fatal("expected a confirmation")
	}
	m = press(m, "y")
	if _, ok := m.state.FindTrack("studying"); ok {
		t.Fatal("track should be gone")
	}
}

func TestSettingsForm(t *testing.T) {
	m := newTestModel(t)
	m = press(m, "5")
	for i := 0; i < len(m.state.Activities)+len(m.state.Tracks); i++ {
		m = press(m, "down")
	}
	m = press(m, "enter") // the settings row
	if m.form == nil || m.form.kind != formSettings {
		t.Fatal("expected the settings form")
	}
	m = press(m, "ctrl+u", "35", "ctrl+s")
	if m.form != nil || m.state.MicroMinutes != 35 {
		t.Fatalf("micro = %d", m.state.MicroMinutes)
	}
}

func TestRulesAddEditDelete(t *testing.T) {
	m := newTestModel(t)
	n := len(m.state.Rules)
	m = press(m, "4", "a", "I slept badly", "tab", "Only light exercise", "ctrl+s")
	if len(m.state.Rules) != n+1 || m.state.Rules[n].Trigger != "I slept badly" {
		t.Fatalf("rules = %+v", m.state.Rules)
	}
	m = press(m, "d", "y")
	if len(m.state.Rules) != n {
		t.Fatalf("rules after delete = %d", len(m.state.Rules))
	}
}

func TestWelcomeStartEmpty(t *testing.T) {
	m := newTestModel(t)
	m.welcome = true
	if !strings.Contains(m.View(), "WELCOME") {
		t.Fatal("welcome screen not shown")
	}
	m = press(m, "n")
	if m.welcome || len(m.state.Activities) != 0 || len(m.state.Queue) != 0 || len(m.state.Tracks) != 0 {
		t.Fatalf("welcome=%v activities=%d tracks=%d", m.welcome, len(m.state.Activities), len(m.state.Tracks))
	}
	if m.form == nil || m.tab != tabPlan {
		t.Fatal("starting empty should open the add-activity form")
	}
	// the empty app still renders everywhere
	m = press(m, "esc", "esc")
	for _, k := range []string{"1", "2", "3", "4", "5"} {
		_ = press(m, k).View()
	}
	if !strings.Contains(press(m, "1").View(), "NOTHING IN THE QUEUE") {
		t.Fatal("empty dashboard should say how to start")
	}
	if got := press(m, "t").flashMsg; !strings.Contains(got, "No time tracks") {
		t.Fatalf("T with no tracks: %q", got)
	}
}

func TestWelcomeKeepExample(t *testing.T) {
	m := newTestModel(t)
	n := len(m.state.Activities)
	m.welcome = true
	m = press(m, "e")
	if m.welcome || len(m.state.Activities) != n {
		t.Fatal("keeping the example must change nothing")
	}
}

func TestHistoryMergesActivitiesAndTime(t *testing.T) {
	m := newTestModel(t)
	id := m.state.Tracks[0].ID
	now := time.Now()
	m = press(m, "c")
	m.mutate(func(s *routine.State) (string, error) {
		_, err := s.LogTime(id, now, 90, "deep work", now)
		return "", err
	})
	out := press(m, "3").View()
	for _, want := range []string{"1 done", "1h30 tracked", "logged by hand", "deep work", "TODAY"} {
		if !strings.Contains(out, want) {
			t.Errorf("history is missing %q:\n%s", want, out)
		}
	}
}

// The layout uses the whole terminal and never overflows it.
func TestLayoutFillsTheTerminal(t *testing.T) {
	m := newTestModel(t)
	m.state.Timer = &routine.RunningTimer{TrackID: m.state.Tracks[0].ID, Since: time.Now().Add(-95 * time.Minute)}
	m = press(m, "c", "m")

	for _, size := range [][2]int{{200, 50}, {120, 30}, {100, 24}, {80, 24}, {60, 20}, {50, 12}} {
		m.width, m.height = size[0], size[1]
		for _, k := range []string{"1", "2", "3", "4", "5"} {
			out := press(m, k).View()
			lines := strings.Split(out, "\n")
			if len(lines) > size[1] {
				t.Fatalf("%dx%d tab %s: %d lines overflow the terminal", size[0], size[1], k, len(lines))
			}
			for _, l := range lines {
				if lipgloss.Width(l) > size[0] {
					t.Fatalf("%dx%d tab %s: line wider than the terminal (%d): %q", size[0], size[1], k, lipgloss.Width(l), l)
				}
			}
		}
	}

	m.width, m.height = 160, 40
	if lines := strings.Count(m.View(), "\n") + 1; lines != 40 {
		t.Fatalf("a roomy terminal should be filled, got %d lines", lines)
	}
}

func TestViewsRender(t *testing.T) {
	m := newTestModel(t)
	m.state.Timer = &routine.RunningTimer{TrackID: m.state.Tracks[0].ID, Since: time.Now().Add(-95 * time.Minute)}
	m = press(m, "c", "m")

	screens := map[string]model{
		"dashboard": m,
		"picker":    press(m, "s", "down"),
		"time":      press(m, "2"),
		"history":   press(m, "3"),
		"rules":     press(m, "4"),
		"plan":      press(m, "5", "down"),
		"form":      press(m, "5", "a", "Gui"),
		"log form":  press(m, "l"),
	}
	welcome := m
	welcome.welcome = true
	screens["welcome"] = welcome
	for name, sm := range screens {
		out := sm.View()
		if strings.TrimSpace(out) == "" {
			t.Fatalf("%s rendered empty", name)
		}
		t.Logf("\n===== %s =====\n%s", name, out)
	}
}

func TestEmptyRulesTabShowsExamplesButSavesNothing(t *testing.T) {
	m := newTestModel(t)
	if len(m.state.Rules) != 0 {
		t.Fatal("no rules by default")
	}
	out := press(m, "4").View()
	for _, want := range []string{"EXAMPLES (not saved)", "write your first rule"} {
		if !strings.Contains(out, want) {
			t.Errorf("empty rules tab missing %q", want)
		}
	}
	// d / enter on nothing must not crash or save anything
	m = press(m, "4", "d", "enter", "y")
	if len(m.state.Rules) != 0 || m.form != nil {
		t.Fatal("nothing to edit or delete")
	}
}

func TestFormTextCursor(t *testing.T) {
	m := newTestModel(t)
	m = press(m, "5", "a", "Helo", "left", "l") // Hel|o -> Hell|o
	if got := m.form.fields[0].value; got != "Hello" {
		t.Fatalf("insert in the middle: %q", got)
	}
	m = press(m, "home", "X", "end", "!")
	if got := m.form.fields[0].value; got != "XHello!" {
		t.Fatalf("home/end: %q", got)
	}
	m = press(m, "home", "right", "delete", "backspace") // X|Hello! -> delete H, backspace X
	if got := m.form.fields[0].value; got != "ello!" {
		t.Fatalf("delete/backspace: %q", got)
	}
	m = press(m, "backspace") // at the start: nothing to delete
	if got := m.form.fields[0].value; got != "ello!" {
		t.Fatalf("backspace at start: %q", got)
	}
	if !strings.Contains(m.View(), "▏ello!") {
		t.Fatal("the cursor should be drawn where it is")
	}
}

func TestHelpOverlay(t *testing.T) {
	m := newTestModel(t)
	m = press(m, "?")
	out := m.View()
	for _, want := range []string{"KEYS", "DASHBOARD", "TIME TAB", "FORMS", "any key"} {
		if !strings.Contains(out, want) {
			t.Errorf("help is missing %q", want)
		}
	}
	m = press(m, "x") // any key closes it, and is not acted on
	if m.help || m.tab != tabDashboard {
		t.Fatal("help should close")
	}
}
