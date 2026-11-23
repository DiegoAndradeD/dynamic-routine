package routine

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Plan is the part of the state the user edits (in the app, or as TOML with
// `routine edit`): which activities and time tracks exist, the rules, and a few
// settings. It is a view over State: history (logs, time entries) is never part
// of it.
type Plan struct {
	MicroMinutes int               `toml:"micro_minutes"`
	Categories   map[string]string `toml:"categories"`
	Activities   []Activity        `toml:"activity"`
	Tracks       []Track           `toml:"track"`
	Rules        []Rule            `toml:"rule"`
}

var weekdayKeys = [...]string{"sun", "mon", "tue", "wed", "thu", "fri", "sat"}

// DefaultCategories are the colours used when a plan does not define its own.
func DefaultCategories() map[string]string {
	return map[string]string{
		"body":    "#10B981",
		"mind":    "#06B6D4",
		"focus":   "#F59E0B",
		"leisure": "#A855F7",
	}
}

// DefaultPlan is the example plan a new install starts from. It is only a
// starting point to show what the app does: everything in it can be changed.
func DefaultPlan() Plan {
	return Plan{
		MicroMinutes: DefaultMicroMinutes,
		Categories:   DefaultCategories(),
		Activities: []Activity{
			{ID: "walk", Name: "Go for a walk", Category: "body", TargetPerWeek: 3, RecommendedMinutes: 30, Weight: 5},
			{ID: "stretch", Name: "Stretching", Category: "body", TargetPerWeek: 4, RecommendedMinutes: 15, Weight: 4},
			{ID: "language", Name: "Practice a language", Category: "focus", TargetPerWeek: 3, RecommendedMinutes: 30, Weight: 3},
			{ID: "journal", Name: "Write in a journal", Category: "mind", TargetPerWeek: 4, RecommendedMinutes: 10, Weight: 3},
			{ID: "draw", Name: "Draw or paint", Category: "leisure", TargetPerWeek: 2, RecommendedMinutes: 45, Weight: 2},
			{ID: "cook", Name: "Cook something new", Category: "leisure", TargetPerWeek: 1, RecommendedMinutes: 60, Weight: 1},
		},
		Tracks: []Track{
			{ID: "focus", Name: "Focus time", Category: "focus", Period: PeriodDay, TargetMinutes: 120, Days: []string{"mon", "tue", "wed", "thu", "fri"}},
			{ID: "outdoors", Name: "Time outdoors", Category: "body", Period: PeriodWeek, TargetMinutes: 240},
		},
	}
}

// ExampleRules are shown, greyed out, where there are no rules yet, to give an
// idea of what to write. They are never stored: a new plan starts with no rules.
func ExampleRules() []Rule {
	return []Rule{
		{Trigger: "The day started late", Instruction: "Do the minimum: one short activity from the top of the queue. Skip the rest."},
		{Trigger: "Low on energy", Instruction: "Pick the lightest activity in the queue. Showing up counts."},
		{Trigger: "A week went off the rails", Instruction: "Do not double up to make up for it. Continue from the top of the queue."},
	}
}

// NewState builds a fresh state from a plan, seeding the queue from the
// activities' weekly targets.
func NewState(p Plan) (State, error) {
	s := State{Version: CurrentVersion}
	s.Normalize()
	if err := s.ApplyPlan(p); err != nil {
		return State{}, err
	}
	return s, nil
}

// Plan returns the editable view of the state.
func (s *State) Plan() Plan {
	cats := make(map[string]string, len(s.Categories))
	for k, v := range s.Categories {
		cats[k] = v
	}
	tracks := make([]Track, len(s.Tracks))
	for i, t := range s.Tracks {
		t.Days = append([]string(nil), t.Days...)
		tracks[i] = t
	}
	return Plan{
		MicroMinutes: s.MicroMinutes,
		Categories:   cats,
		Activities:   s.SortedActivities(),
		Tracks:       tracks,
		Rules:        append([]Rule(nil), s.Rules...),
	}
}

// ApplyPlan validates p and makes it the user-editable part of the state. The
// history is untouched and the queue is reconciled with the plan: removed
// activities leave it, new ones are spread into it, and a changed weekly
// target adds or drops instances. Nothing changes if the plan is invalid.
func (s *State) ApplyPlan(p Plan) error {
	if err := p.normalize(); err != nil {
		return err
	}
	next := make(map[string]Activity, len(p.Activities))
	for _, a := range p.Activities {
		next[a.ID] = a
	}

	queue := make([]QueueItem, 0, len(s.Queue))
	for _, it := range s.Queue {
		if _, ok := next[it.ActivityID]; ok {
			queue = append(queue, it)
		}
	}

	// Walk activities in a stable order so the seeded queue is deterministic.
	ordered := append([]Activity(nil), p.Activities...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Weight > ordered[j].Weight })
	now := time.Now()
	for _, a := range ordered {
		want := a.TargetPerWeek
		if old, ok := s.Activities[a.ID]; ok {
			want = a.TargetPerWeek - old.TargetPerWeek
		}
		switch {
		case want > 0:
			queue = spread(queue, a.ID, want, now)
		case want < 0:
			queue = dropInstances(queue, a.ID, -want)
		}
	}

	s.MicroMinutes = p.MicroMinutes
	s.Categories = p.Categories
	s.Tracks = p.Tracks
	if s.Timer != nil {
		if _, ok := s.FindTrack(s.Timer.TrackID); !ok {
			s.Timer = nil // its track was deleted
		}
	}
	s.Activities = next
	s.Rules = p.Rules
	s.Queue = queue
	return nil
}

// normalize fills defaults and checks the plan. It reports the first problem
// in a form meant to be read by the person editing the file.
func (p *Plan) normalize() error {
	if p.MicroMinutes <= 0 {
		p.MicroMinutes = DefaultMicroMinutes
	}
	if p.Categories == nil {
		p.Categories = DefaultCategories()
	}
	for id, color := range p.Categories {
		if !validHexColor(color) {
			return fmt.Errorf("categories.%s: %q is not a #RRGGBB colour", id, color)
		}
	}
	seen := map[string]bool{}
	for i := range p.Activities {
		a := &p.Activities[i]
		a.Name = strings.TrimSpace(a.Name)
		if a.Name == "" {
			return fmt.Errorf("activity #%d needs a name", i+1)
		}
		if a.ID = strings.TrimSpace(a.ID); a.ID == "" {
			a.ID = slug(a.Name)
		}
		if seen[a.ID] {
			return fmt.Errorf("activity id %q is used twice", a.ID)
		}
		seen[a.ID] = true
		if a.TargetPerWeek < 1 {
			return fmt.Errorf("activity %q: times per week (target_per_week) must be at least 1", a.Name)
		}
		if a.RecommendedMinutes < 1 {
			return fmt.Errorf("activity %q: minutes (recommended_minutes) must be at least 1", a.Name)
		}
		if a.Weight == 0 {
			a.Weight = a.TargetPerWeek
		}
		if a.Category = strings.TrimSpace(a.Category); a.Category == "" {
			a.Category = "general"
		}
	}

	trackIDs := map[string]bool{}
	for i := range p.Tracks {
		t := &p.Tracks[i]
		t.Name = strings.TrimSpace(t.Name)
		if t.Name == "" {
			return fmt.Errorf("time track #%d needs a name", i+1)
		}
		if t.ID = strings.TrimSpace(t.ID); t.ID == "" {
			t.ID = slug(t.Name)
		}
		if trackIDs[t.ID] {
			return fmt.Errorf("time track id %q is used twice", t.ID)
		}
		trackIDs[t.ID] = true
		switch t.Period = strings.ToLower(strings.TrimSpace(t.Period)); t.Period {
		case "":
			t.Period = PeriodDay
		case PeriodDay, PeriodWeek:
		default:
			return fmt.Errorf("time track %q: period must be \"day\" or \"week\", not %q", t.Name, t.Period)
		}
		if t.TargetMinutes < 0 {
			return fmt.Errorf("time track %q: the goal cannot be negative", t.Name)
		}
		days, err := NormalizeDays(t.Days)
		if err != nil {
			return fmt.Errorf("time track %q: %w", t.Name, err)
		}
		t.Days = days
		if t.Category = strings.TrimSpace(t.Category); t.Category == "" {
			t.Category = "general"
		}
	}

	ruleIDs := map[string]bool{}
	for i := range p.Rules {
		r := &p.Rules[i]
		if strings.TrimSpace(r.Trigger) == "" || strings.TrimSpace(r.Instruction) == "" {
			return fmt.Errorf("rule #%d needs both a trigger and an instruction", i+1)
		}
		if r.ID == "" {
			r.ID = fmt.Sprintf("r-%d", i+1)
		}
		if ruleIDs[r.ID] {
			return fmt.Errorf("rule id %q is used twice", r.ID)
		}
		ruleIDs[r.ID] = true
	}
	return nil
}

// NormalizeDays checks day names ("mon", "Tuesday"...) and returns them as
// sorted three-letter keys without duplicates.
func NormalizeDays(names []string) ([]string, error) {
	seen := map[int]bool{}
	for _, n := range names {
		key := strings.ToLower(strings.TrimSpace(n))
		if len(key) > 3 {
			key = key[:3]
		}
		found := false
		for i, k := range weekdayKeys {
			if k == key {
				seen[i] = true
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%q is not a day (use mon, tue, wed, thu, fri, sat, sun)", n)
		}
	}
	out := []string{}
	for i := 0; i < 7; i++ { // Monday first, so lists read naturally
		if d := (i + 1) % 7; seen[d] {
			out = append(out, weekdayKeys[d])
		}
	}
	return out, nil
}

func validHexColor(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	for _, c := range s[1:] {
		if !strings.ContainsRune("0123456789abcdefABCDEF", c) {
			return false
		}
	}
	return true
}

// slug turns a name into an id: "Read a book" -> "read_a_book".
func slug(name string) string {
	var b strings.Builder
	underscore := false
	for _, r := range strings.ToLower(name) {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			underscore = false
		} else if !underscore && b.Len() > 0 {
			b.WriteByte('_')
			underscore = true
		}
	}
	if id := strings.TrimRight(b.String(), "_"); id != "" {
		return id
	}
	return "activity"
}

func newID(prefix string, now time.Time) string {
	return fmt.Sprintf("%s-%d", prefix, now.UnixMilli())
}
