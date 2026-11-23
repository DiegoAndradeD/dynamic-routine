package routine

import "sort"

// Merge folds another copy of the state into s. It is meant for the conflict
// copies a sync tool leaves behind when two machines were edited at once.
// History is unioned by id; s keeps its own queue, plan and running timer, and
// takes from other only history it does not have (a timer running in the other
// copy may be stale, so it is ignored).
func (s *State) Merge(other State) (logs, timeItems int) {
	haveLog := map[string]bool{}
	for _, l := range s.Logs {
		haveLog[l.ID] = true
	}
	for _, l := range other.Logs {
		if !haveLog[l.ID] {
			s.Logs = append(s.Logs, l)
			logs++
		}
	}
	sort.SliceStable(s.Logs, func(i, j int) bool { return s.Logs[i].CompletedAt.After(s.Logs[j].CompletedAt) })

	haveSession := map[string]bool{}
	for _, w := range s.TimeSessions {
		haveSession[w.ID] = true
	}
	for _, w := range other.TimeSessions {
		if !haveSession[w.ID] {
			s.TimeSessions = append(s.TimeSessions, w)
			timeItems++
		}
	}
	sort.SliceStable(s.TimeSessions, func(i, j int) bool { return s.TimeSessions[i].Start.Before(s.TimeSessions[j].Start) })

	haveEntry := map[string]bool{}
	for _, e := range s.TimeEntries {
		haveEntry[e.ID] = true
	}
	for _, e := range other.TimeEntries {
		if !haveEntry[e.ID] {
			s.TimeEntries = append(s.TimeEntries, e)
			timeItems++
		}
	}

	return logs, timeItems
}
