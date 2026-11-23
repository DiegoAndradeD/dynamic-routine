// Package routine holds the domain model and rules of the dynamic routine:
// the priority queue of activities, weekly stats and the daily work quota.
//
// It knows nothing about files, encryption or the terminal.
package routine

import "time"

const (
	// CurrentVersion is the schema version written to the state file.
	CurrentVersion = 4

	// Session kinds stored in ActivityLog.Type.
	SessionFull  = "full"
	SessionMicro = "micro"

	// DefaultMicroMinutes is the length of a micro block unless the plan says otherwise.
	DefaultMicroMinutes = 20
)

// Activity is something you want to do regularly. Its TargetPerWeek is also how
// many times it sits on the queue conveyor.
type Activity struct {
	ID                 string `json:"id" toml:"id"`
	Name               string `json:"name" toml:"name"`
	Category           string `json:"category" toml:"category"`
	TargetPerWeek      int    `json:"targetPerWeek" toml:"target_per_week"`
	RecommendedMinutes int    `json:"recommendedMinutes" toml:"recommended_minutes"`
	Weight             int    `json:"weight" toml:"weight"` // display order, higher first
}

type QueueItem struct {
	InstanceID string `json:"instanceId"`
	ActivityID string `json:"activityId"`
}

type ActivityLog struct {
	ID          string    `json:"id"`
	ActivityID  string    `json:"activityId"`
	CompletedAt time.Time `json:"completedAt"`
	Type        string    `json:"type"` // SessionFull or SessionMicro
}

// Rule is a contingency: "when this happens, do that".
type Rule struct {
	ID          string `json:"id" toml:"id"`
	Category    string `json:"category,omitempty" toml:"category"`
	Trigger     string `json:"trigger" toml:"trigger"`
	Instruction string `json:"instruction" toml:"instruction"`
}

// Track periods.
const (
	PeriodDay  = "day"  // a goal for each of the track's days
	PeriodWeek = "week" // one goal for the whole week
)

// Track is a time-based task: something you spend time on rather than tick off
// (work, studying, being outdoors...). Time is added with a timer or logged by
// hand, and measured against an optional goal.
type Track struct {
	ID            string   `json:"id" toml:"id"`
	Name          string   `json:"name" toml:"name"`
	Category      string   `json:"category" toml:"category"`
	Period        string   `json:"period" toml:"period"`                // PeriodDay or PeriodWeek
	TargetMinutes int      `json:"targetMinutes" toml:"target_minutes"` // per day or per week; 0 = no goal
	Days          []string `json:"days" toml:"days"`                    // "mon".."sun" a daily goal applies to; empty = every day
}

// TimeSession is one closed slice recorded by the timer.
type TimeSession struct {
	ID      string    `json:"id"`
	TrackID string    `json:"trackId"`
	Start   time.Time `json:"start"`
	End     time.Time `json:"end"`
}

// TimeEntry is time logged by hand for a calendar day.
type TimeEntry struct {
	ID       string    `json:"id"`
	TrackID  string    `json:"trackId"`
	Date     string    `json:"date"` // YYYY-MM-DD
	Minutes  int       `json:"minutes"`
	Note     string    `json:"note,omitempty"`
	LoggedAt time.Time `json:"loggedAt"`
}

// RunningTimer is the timer that is counting right now. Only one runs at a time.
type RunningTimer struct {
	TrackID string    `json:"trackId"`
	Since   time.Time `json:"since"`
}

// Legacy shapes (versions before 4 had one hard-wired "work" quota). They are
// only read, to migrate old files into a Track.
type WorkConfig struct {
	DailyTargetMinutes int            `json:"dailyTargetMinutes"`
	Days               []time.Weekday `json:"days"`
}

type WorkSession struct {
	ID    string    `json:"id"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type WorkAdjustment struct {
	Date    string `json:"date"`
	Minutes int    `json:"minutes"`
}

// State is everything that is persisted. The user-editable part of it is
// exposed as a Plan (see plan.go); the rest is history.
type State struct {
	Version      int                 `json:"version"`
	MicroMinutes int                 `json:"microMinutes"`
	Categories   map[string]string   `json:"categories"` // category id -> hex colour
	Queue        []QueueItem         `json:"queue"`
	Activities   map[string]Activity `json:"activities"`
	Logs         []ActivityLog       `json:"logs"` // newest first
	Rules        []Rule              `json:"rules"`

	Tracks       []Track       `json:"tracks"`
	TimeSessions []TimeSession `json:"timeSessions"`
	TimeEntries  []TimeEntry   `json:"timeEntries"`
	Timer        *RunningTimer `json:"timer,omitempty"`

	LegacyWork             *WorkConfig      `json:"work,omitempty"`
	LegacyWorkSessions     []WorkSession    `json:"workSessions,omitempty"`
	LegacyWorkAdjustments  []WorkAdjustment `json:"workAdjustments,omitempty"`
	LegacyWorkRunningSince *time.Time       `json:"workRunningSince,omitempty"`
}

type WeeklyStat struct {
	Full  int
	Micro int
	Total int
}

// Normalize fills in fields missing from older or hand-edited state files and
// migrates them to CurrentVersion.
func (s *State) Normalize() {
	if s.Activities == nil {
		s.Activities = map[string]Activity{}
	}
	if s.Queue == nil {
		s.Queue = []QueueItem{}
	}
	if s.Logs == nil {
		s.Logs = []ActivityLog{}
	}
	if s.Rules == nil {
		s.Rules = []Rule{}
	}
	if s.Tracks == nil {
		s.Tracks = []Track{}
	}
	if s.TimeSessions == nil {
		s.TimeSessions = []TimeSession{}
	}
	if s.TimeEntries == nil {
		s.TimeEntries = []TimeEntry{}
	}
	if s.Categories == nil {
		s.Categories = DefaultCategories()
	}
	if s.MicroMinutes <= 0 {
		s.MicroMinutes = DefaultMicroMinutes
	}
	if s.Version < 4 {
		s.migrateWork()
	}
	s.Version = CurrentVersion
}

// migrateWork turns the old built-in work quota into an ordinary Track called
// "Work", keeping its sessions, adjustments and running timer.
func (s *State) migrateWork() {
	had := s.LegacyWork != nil || len(s.LegacyWorkSessions) > 0 ||
		len(s.LegacyWorkAdjustments) > 0 || s.LegacyWorkRunningSince != nil
	if had {
		t := Track{ID: "work", Name: "Work", Category: "work", Period: PeriodDay, TargetMinutes: 8 * 60}
		if w := s.LegacyWork; w != nil {
			t.TargetMinutes = w.DailyTargetMinutes
			t.Days = []string{}
			for _, d := range w.Days {
				t.Days = append(t.Days, weekdayKeys[d])
			}
			if len(w.Days) == 0 {
				t.TargetMinutes = 0 // a quota on no days is no quota
			}
		}
		s.Tracks = append(s.Tracks, t)

		for _, ws := range s.LegacyWorkSessions {
			s.TimeSessions = append(s.TimeSessions, TimeSession{ID: ws.ID, TrackID: t.ID, Start: ws.Start, End: ws.End})
		}
		for _, a := range s.LegacyWorkAdjustments {
			s.TimeEntries = append(s.TimeEntries, TimeEntry{
				ID: "m-" + a.Date, TrackID: t.ID, Date: a.Date, Minutes: a.Minutes, Note: "adjustment",
			})
		}
		if s.LegacyWorkRunningSince != nil {
			s.Timer = &RunningTimer{TrackID: t.ID, Since: *s.LegacyWorkRunningSince}
		}
	}
	s.LegacyWork, s.LegacyWorkSessions, s.LegacyWorkAdjustments, s.LegacyWorkRunningSince = nil, nil, nil, nil
}
