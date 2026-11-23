package routine

import (
	"testing"
)

func countOf(s State, id string) int {
	n := 0
	for _, q := range s.Queue {
		if q.ActivityID == id {
			n++
		}
	}
	return n
}

func TestDefaultPlanSeedsQueueFromTargets(t *testing.T) {
	st, err := NewState(DefaultPlan())
	if err != nil {
		t.Fatal(err)
	}
	total := 0
	for _, a := range st.Activities {
		if got := countOf(st, a.ID); got != a.TargetPerWeek {
			t.Fatalf("%s has %d instances, want %d", a.ID, got, a.TargetPerWeek)
		}
		total += a.TargetPerWeek
	}
	if len(st.Queue) != total {
		t.Fatalf("queue length %d, want %d", len(st.Queue), total)
	}
	// spread: the most frequent activity must not come up back to back
	for i := 1; i < len(st.Queue); i++ {
		if st.Queue[i].ActivityID == st.Queue[i-1].ActivityID {
			t.Fatalf("clumped at %d: %v", i, queueIDs(st))
		}
	}
}

func TestPlanRoundTrip(t *testing.T) {
	st, _ := NewState(DefaultPlan())
	st.Complete(SessionFull, at(19, 10, 0))
	queue := queueIDs(st)

	if err := st.ApplyPlan(st.Plan()); err != nil {
		t.Fatal(err)
	}
	if !equal(queue, queueIDs(st)) || len(st.Logs) != 1 {
		t.Fatal("re-applying the same plan must change nothing")
	}
}

func TestApplyPlanReconcilesQueue(t *testing.T) {
	st, _ := NewState(DefaultPlan())
	p := st.Plan()

	for i := range p.Activities {
		switch p.Activities[i].ID {
		case "walk":
			p.Activities[i].TargetPerWeek = 2 // 3 -> 2
		case "cook":
			p.Activities[i].TargetPerWeek = 5 // 1 -> 5
		}
	}
	p.Activities = append(p.Activities, Activity{Name: "Learn Rust!", TargetPerWeek: 3, RecommendedMinutes: 30})
	kept := p.Activities[:0:0]
	for _, a := range p.Activities {
		if a.ID != "draw" { // removed
			kept = append(kept, a)
		}
	}
	p.Activities = kept

	if err := st.ApplyPlan(p); err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"walk": 2, "cook": 5, "learn_rust": 3, "draw": 0, "stretch": 4}
	for id, n := range want {
		if got := countOf(st, id); got != n {
			t.Errorf("%s: %d instances, want %d", id, got, n)
		}
	}
	if _, ok := st.Activities["learn_rust"]; !ok {
		t.Fatal("id should be derived from the name")
	}
}

func TestApplyPlanRejectsBadInputAndChangesNothing(t *testing.T) {
	cases := map[string]func(*Plan){
		"empty name":     func(p *Plan) { p.Activities[0].Name = " " },
		"duplicate id":   func(p *Plan) { p.Activities[1].ID = p.Activities[0].ID },
		"zero target":    func(p *Plan) { p.Activities[0].TargetPerWeek = 0 },
		"zero minutes":   func(p *Plan) { p.Activities[0].RecommendedMinutes = 0 },
		"bad day":        func(p *Plan) { p.Tracks[0].Days = []string{"funday"} },
		"bad period":     func(p *Plan) { p.Tracks[0].Period = "fortnight" },
		"nameless track": func(p *Plan) { p.Tracks[0].Name = "" },
		"dup track id":   func(p *Plan) { p.Tracks[1].ID = p.Tracks[0].ID },
		"bad colour":     func(p *Plan) { p.Categories["body"] = "green" },
		"half a rule":    func(p *Plan) { p.Rules = []Rule{{Trigger: "only a trigger"}} },
		"negative goal":  func(p *Plan) { p.Tracks[0].TargetMinutes = -1 },
	}
	for name, mutate := range cases {
		st, _ := NewState(DefaultPlan())
		before := queueIDs(st)
		p := st.Plan()
		mutate(&p)
		if err := st.ApplyPlan(p); err == nil {
			t.Errorf("%s: expected an error", name)
		}
		if !equal(before, queueIDs(st)) {
			t.Errorf("%s: invalid plan modified the state", name)
		}
	}
}

func TestTrackDaysParsing(t *testing.T) {
	st, _ := NewState(DefaultPlan())
	p := st.Plan()
	p.Tracks[0].Days = []string{"Saturday", "mon", "MON"}
	if err := st.ApplyPlan(p); err != nil {
		t.Fatal(err)
	}
	if got := st.Tracks[0].Days; len(got) != 2 || got[0] != "mon" || got[1] != "sat" {
		t.Fatalf("days = %v", got)
	}
}

func TestDeletingATrackKeepsHistoryAndStopsItsTimer(t *testing.T) {
	st, _ := NewState(DefaultPlan())
	now := at(19, 12, 0)
	st.LogTime("focus", now, 30, "", now)
	st.StartTimer("focus", now)

	p := st.Plan()
	p.Tracks = p.Tracks[1:]
	if err := st.ApplyPlan(p); err != nil {
		t.Fatal(err)
	}
	if st.Timer != nil {
		t.Fatal("the timer of a deleted track must stop")
	}
	if len(st.TimeEntries) != 1 {
		t.Fatal("history stays")
	}
}

func TestMergeUnionsHistory(t *testing.T) {
	a, _ := NewState(DefaultPlan())
	a.Complete(SessionFull, at(19, 9, 0))
	b := a
	b.Logs = append([]ActivityLog(nil), a.Logs...)

	a.Complete(SessionFull, at(19, 10, 0))
	b.Logs = append([]ActivityLog{{ID: "log-other", ActivityID: "walk", CompletedAt: at(19, 11, 0), Type: SessionFull}}, b.Logs...)
	b.TimeSessions = []TimeSession{{ID: "w-other", TrackID: "focus", Start: at(19, 9, 0), End: at(19, 10, 0)}}

	logs, sessions := a.Merge(b)
	if logs != 1 || sessions != 1 || len(a.Logs) != 3 {
		t.Fatalf("merged %d logs %d sessions, total logs %d", logs, sessions, len(a.Logs))
	}
	if a.Logs[0].ID != "log-other" {
		t.Fatal("logs must stay newest first")
	}
	if l, _ := a.Merge(b); l != 0 {
		t.Fatal("merging twice must not duplicate")
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"Read a book": "read_a_book", "  C++ / Rust!! ": "c_rust", "!!!": "activity"} {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewPlanHasNoRulesOnlyExamples(t *testing.T) {
	st, _ := NewState(DefaultPlan())
	if len(st.Rules) != 0 {
		t.Fatalf("a new state must not ship rules, got %d", len(st.Rules))
	}
	if len(ExampleRules()) == 0 {
		t.Fatal("examples are what the empty Rules tab shows")
	}
}

func TestEmptyPlanIsAllowed(t *testing.T) {
	st, _ := NewState(DefaultPlan())
	p := st.Plan()
	p.Activities, p.Rules = nil, nil
	if err := st.ApplyPlan(p); err != nil {
		t.Fatal(err)
	}
	if len(st.Queue) != 0 || len(st.Activities) != 0 {
		t.Fatalf("queue=%d activities=%d", len(st.Queue), len(st.Activities))
	}
	if _, ok := st.Complete(SessionFull, at(19, 9, 0)); ok {
		t.Fatal("nothing to complete on an empty queue")
	}
}
