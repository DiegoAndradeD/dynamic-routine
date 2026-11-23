package routine

import (
	"sort"
	"time"
)

// WeekStart returns Monday 00:00 of the week containing t.
func WeekStart(t time.Time) time.Time {
	daysSinceMonday := (int(t.Weekday()) + 6) % 7
	return time.Date(t.Year(), t.Month(), t.Day()-daysSinceMonday, 0, 0, 0, 0, t.Location())
}

// WeeklyStats counts completions since Monday 00:00 for every activity.
func (s *State) WeeklyStats(now time.Time) map[string]WeeklyStat {
	monday := WeekStart(now)

	stats := make(map[string]WeeklyStat, len(s.Activities))
	for id := range s.Activities {
		stats[id] = WeeklyStat{}
	}
	for _, l := range s.Logs {
		if l.CompletedAt.Before(monday) {
			continue
		}
		st := stats[l.ActivityID]
		if l.Type == SessionMicro {
			st.Micro++
		} else {
			st.Full++
		}
		st.Total++
		stats[l.ActivityID] = st
	}
	return stats
}

// SortedActivities lists activities by weight (desc) then name, so views have a
// stable order (map iteration is random).
func (s *State) SortedActivities() []Activity {
	acts := make([]Activity, 0, len(s.Activities))
	for _, a := range s.Activities {
		acts = append(acts, a)
	}
	sort.Slice(acts, func(i, j int) bool {
		if acts[i].Weight != acts[j].Weight {
			return acts[i].Weight > acts[j].Weight
		}
		return acts[i].Name < acts[j].Name
	})
	return acts
}
