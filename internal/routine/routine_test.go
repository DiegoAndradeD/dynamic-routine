package routine

import (
	"testing"
	"time"
)

var (
	loc = time.UTC
	// Monday 19 Oct 2026
	monday = time.Date(2026, 10, 19, 0, 0, 0, 0, loc)
)

func at(day, hour, min int) time.Time {
	return time.Date(2026, 10, day, hour, min, 0, 0, loc)
}

func queueIDs(s State) []string {
	out := make([]string, len(s.Queue))
	for i, q := range s.Queue {
		out[i] = q.ActivityID
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func smallState() State {
	s := State{
		Activities: map[string]Activity{
			"a": {ID: "a", Name: "A"}, "b": {ID: "b", Name: "B"}, "c": {ID: "c", Name: "C"},
		},
		Queue: []QueueItem{{"q1", "a"}, {"q2", "b"}, {"q3", "c"}},
	}
	s.Normalize()
	return s
}

func TestCompleteRotatesAndLogs(t *testing.T) {
	s := smallState()
	act, ok := s.Complete(SessionFull, at(19, 10, 0))
	if !ok || act.ID != "a" {
		t.Fatalf("complete = %v %v", act, ok)
	}
	if got := queueIDs(s); !equal(got, []string{"b", "c", "a"}) {
		t.Fatalf("queue = %v", got)
	}
	if len(s.Logs) != 1 || s.Logs[0].ActivityID != "a" {
		t.Fatalf("logs = %+v", s.Logs)
	}
}

func TestChooseKeepsOrderOfTheRest(t *testing.T) {
	s := smallState()
	if _, ok := s.Choose(2); !ok {
		t.Fatal("choose failed")
	}
	if got := queueIDs(s); !equal(got, []string{"c", "a", "b"}) {
		t.Fatalf("queue = %v", got)
	}
	if len(s.Logs) != 0 {
		t.Fatal("choosing must not log anything")
	}
	// choosing as many times as wanted
	s.Choose(2)
	s.Choose(1)
	if len(s.Queue) != 3 {
		t.Fatalf("queue length changed: %d", len(s.Queue))
	}
	if _, ok := s.Choose(9); ok {
		t.Fatal("out of range choose must fail")
	}
}

func TestUndoLastRestoresTop(t *testing.T) {
	s := smallState()
	s.Complete(SessionFull, at(19, 10, 0))
	act, _, ok := s.UndoLast()
	if !ok || act.ID != "a" {
		t.Fatalf("undo = %v %v", act, ok)
	}
	if got := queueIDs(s); !equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("queue = %v", got)
	}
	if len(s.Logs) != 0 {
		t.Fatal("log should be removed")
	}
	if _, _, ok := s.UndoLast(); ok {
		t.Fatal("nothing left to undo")
	}
}

func TestWeeklyStatsStartMonday(t *testing.T) {
	s := smallState()
	s.Complete(SessionFull, at(18, 23, 0)) // Sunday: previous week
	s.Complete(SessionMicro, at(19, 8, 0))
	now := at(21, 12, 0)
	st := s.WeeklyStats(now)
	if st["b"].Total != 1 || st["b"].Micro != 1 {
		t.Fatalf("b = %+v", st["b"])
	}
	if st["a"].Total != 0 {
		t.Fatalf("a = %+v (previous week must not count)", st["a"])
	}
}

// withTracks gives smallState two tracks: "job" (8h on every day) and "read"
// (3h per week).
func withTracks() State {
	s := smallState()
	s.Tracks = []Track{
		{ID: "job", Name: "Job", Period: PeriodDay, TargetMinutes: 480},
		{ID: "read", Name: "Read", Period: PeriodWeek, TargetMinutes: 180},
	}
	return s
}

func TestTimerSlicesAccumulate(t *testing.T) {
	s := withTracks()
	if _, _, err := s.StartTimer("job", at(19, 9, 0)); err != nil {
		t.Fatal(err)
	}
	if got := s.WorkedOn("job", at(19, 11, 0), at(19, 11, 0)); got != 2*time.Hour {
		t.Fatalf("running = %v", got)
	}
	s.StopTimer(at(19, 11, 0))
	s.StartTimer("job", at(19, 14, 0))
	s.StopTimer(at(19, 17, 0))
	now := at(19, 18, 0)
	if got := s.WorkedOn("job", now, now); got != 5*time.Hour {
		t.Fatalf("worked = %v, want 5h", got)
	}
	if got := s.RemainingToday("job", now); got != 3*time.Hour {
		t.Fatalf("remaining = %v, want 3h", got)
	}
	if len(s.ItemsOn("job", now, now)) != 2 {
		t.Fatal("expected two slices")
	}
	if s.WorkedOn("read", now, now) != 0 {
		t.Fatal("time must not leak between tracks")
	}
}

func TestOnlyOneTimerRuns(t *testing.T) {
	s := withTracks()
	s.StartTimer("job", at(19, 9, 0))
	prev, slice, err := s.StartTimer("read", at(19, 10, 0))
	if err != nil || prev.TrackID != "job" || slice != time.Hour {
		t.Fatalf("prev=%+v slice=%v err=%v", prev, slice, err)
	}
	if s.Timer.TrackID != "read" || len(s.TimeSessions) != 1 {
		t.Fatalf("timer=%+v sessions=%d", s.Timer, len(s.TimeSessions))
	}
	if _, _, err := s.StartTimer("nope", at(19, 11, 0)); err == nil {
		t.Fatal("unknown track must fail")
	}
	if s.Timer.TrackID != "read" {
		t.Fatal("a failed start must not stop the running timer")
	}
}

func TestSliceCrossingMidnightIsSplit(t *testing.T) {
	s := withTracks()
	s.StartTimer("job", at(19, 22, 0))
	now := at(20, 2, 0)
	if got := s.WorkedOn("job", at(19, 12, 0), now); got != 2*time.Hour {
		t.Fatalf("monday = %v", got)
	}
	if got := s.WorkedOn("job", at(20, 12, 0), now); got != 2*time.Hour {
		t.Fatalf("tuesday = %v", got)
	}
}

func TestLogTimeAndDelete(t *testing.T) {
	s := withTracks()
	now := at(21, 12, 0)
	yesterday := at(20, 0, 0)

	e, err := s.LogTime("job", yesterday, 90, "  sprint planning ", now)
	if err != nil {
		t.Fatal(err)
	}
	if e.Note != "sprint planning" || e.Date != "2026-10-20" {
		t.Fatalf("entry = %+v", e)
	}
	if got := s.WorkedOn("job", yesterday, now); got != 90*time.Minute {
		t.Fatalf("worked = %v", got)
	}
	s.LogTime("job", yesterday, 30, "", now)
	if got := s.WorkedOn("job", yesterday, now); got != 2*time.Hour {
		t.Fatalf("entries must add up, got %v", got)
	}

	for name, fn := range map[string]func() error{
		"zero":      func() error { _, err := s.LogTime("job", yesterday, 0, "", now); return err },
		"too long":  func() error { _, err := s.LogTime("job", yesterday, 25*60, "", now); return err },
		"future":    func() error { _, err := s.LogTime("job", at(22, 0, 0), 10, "", now); return err },
		"bad track": func() error { _, err := s.LogTime("zzz", yesterday, 10, "", now); return err },
	} {
		if fn() == nil {
			t.Errorf("%s: expected an error", name)
		}
	}

	if !s.DeleteTimeItem(e.ID) || s.WorkedOn("job", yesterday, now) != 30*time.Minute {
		t.Fatal("delete should remove just that entry")
	}
	s.StartTimer("job", at(21, 9, 0))
	s.StopTimer(at(21, 10, 0))
	if id := s.TimeSessions[0].ID; !s.DeleteTimeItem(id) || len(s.TimeSessions) != 0 {
		t.Fatal("timer slices can be deleted too")
	}
	if s.DeleteTimeItem("nope") {
		t.Fatal("unknown id")
	}
}

func TestDailyWeekBalance(t *testing.T) {
	s := withTracks()
	// Mon 8h, Tue 6h, today (Wed) 3h so far
	now := at(21, 15, 0)
	s.LogTime("job", at(19, 0, 0), 8*60, "", now)
	s.LogTime("job", at(20, 0, 0), 6*60, "", now)
	s.LogTime("job", at(21, 0, 0), 3*60, "", now)

	w := s.Week("job", monday, now)
	if len(w.Days) != 7 || !w.Days[2].IsToday || !w.Days[3].IsFuture {
		t.Fatalf("week layout wrong: %+v", w.Days)
	}
	if w.Balance != -2*time.Hour {
		t.Fatalf("balance = %v, want -2h (today's pending goal is not debt)", w.Balance)
	}
	if w.Worked != 17*time.Hour || w.Target != 56*time.Hour {
		t.Fatalf("week = %v / %v", w.Worked, w.Target)
	}
}

func TestWeeklyGoal(t *testing.T) {
	s := withTracks()
	now := at(22, 12, 0)
	s.LogTime("read", at(19, 0, 0), 60, "", now)
	s.LogTime("read", at(21, 0, 0), 45, "", now)

	w := s.Week("read", monday, now)
	if w.Target != 3*time.Hour || w.Worked != 105*time.Minute || w.Remaining() != 75*time.Minute {
		t.Fatalf("week = %+v remaining %v", w, w.Remaining())
	}
	if w.Balance != 0 {
		t.Fatal("weekly goals have no daily balance")
	}
	if s.TargetOn("read", now) != 0 || s.RemainingToday("read", now) != 0 {
		t.Fatal("a weekly goal has no per-day target")
	}
}

func TestDaysOff(t *testing.T) {
	s := withTracks()
	s.Tracks[0].Days = []string{"mon"}
	if s.TargetOn("job", at(20, 12, 0)) != 0 || s.RemainingToday("job", at(20, 12, 0)) != 0 {
		t.Fatal("tuesday should be a day off")
	}
	if s.TargetOn("job", at(19, 12, 0)) != 8*time.Hour {
		t.Fatal("monday should count")
	}
}

func TestParseMinutes(t *testing.T) {
	good := map[string]int{"90": 90, "90m": 90, "1h": 60, "1h30": 90, "1h30m": 90, " 2H ": 120, "1:30": 90, "0:45": 45, "1.5h": 90, "0": 0}
	for in, want := range good {
		if got, err := ParseMinutes(in); err != nil || got != want {
			t.Errorf("ParseMinutes(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, in := range []string{"", "abc", "-5", "1:75", "h", "1h-3", "1.2.3h"} {
		if _, err := ParseMinutes(in); err == nil {
			t.Errorf("ParseMinutes(%q) should fail", in)
		}
	}
}

func TestMigrateLegacyWork(t *testing.T) {
	end := at(19, 12, 0)
	s := State{
		Version:                2,
		LegacyWork:             &WorkConfig{DailyTargetMinutes: 360, Days: []time.Weekday{time.Monday, time.Tuesday}},
		LegacyWorkSessions:     []WorkSession{{ID: "w-1", Start: at(19, 10, 0), End: end}},
		LegacyWorkAdjustments:  []WorkAdjustment{{Date: "2026-10-19", Minutes: 30}},
		LegacyWorkRunningSince: &end,
	}
	s.Normalize()

	tr, ok := s.FindTrack("work")
	if !ok || tr.TargetMinutes != 360 || tr.Period != PeriodDay || len(tr.Days) != 2 || tr.Days[0] != "mon" {
		t.Fatalf("track = %+v ok=%v", tr, ok)
	}
	now := at(19, 13, 0)
	if got := s.WorkedOn("work", now, now); got != 2*time.Hour+30*time.Minute+time.Hour {
		t.Fatalf("worked = %v (2h session + 30m adjustment + 1h running)", got)
	}
	if s.Timer == nil || s.Timer.TrackID != "work" {
		t.Fatal("running timer lost")
	}
	if s.Version != CurrentVersion || s.LegacyWork != nil || s.LegacyWorkSessions != nil {
		t.Fatal("legacy fields must be cleared")
	}

	var fresh State
	fresh.Version = 2 // a v2 file that never used work
	fresh.Normalize()
	if len(fresh.Tracks) != 0 {
		t.Fatal("no work data, no track")
	}
}
