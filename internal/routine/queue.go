package routine

import (
	"fmt"
	"time"
)

// Current returns the activity at the top of the queue.
func (s *State) Current() (Activity, bool) {
	if len(s.Queue) == 0 {
		return Activity{}, false
	}
	return s.Activities[s.Queue[0].ActivityID], true
}

// Complete logs the current activity and sends it to the back of the queue.
func (s *State) Complete(kind string, now time.Time) (Activity, bool) {
	if len(s.Queue) == 0 {
		return Activity{}, false
	}
	current := s.Queue[0]

	s.Logs = append([]ActivityLog{{
		ID:          newID("log", now),
		ActivityID:  current.ActivityID,
		CompletedAt: now,
		Type:        kind,
	}}, s.Logs...)

	rest := append([]QueueItem(nil), s.Queue[1:]...)
	s.Queue = append(rest, QueueItem{InstanceID: newID("q", now), ActivityID: current.ActivityID})
	return s.Activities[current.ActivityID], true
}

// Choose pulls the item at index to the top of the queue, keeping the order of
// everything else. Nothing is logged: the queue still reflects priority, the
// choice only decides what happens now.
func (s *State) Choose(index int) (Activity, bool) {
	if index < 0 || index >= len(s.Queue) {
		return Activity{}, false
	}
	item := s.Queue[index]
	s.Queue = moveToFront(s.Queue, index)
	return s.Activities[item.ActivityID], true
}

// UndoLast reverts the most recent completion: the log entry is dropped and the
// activity returns from the back of the queue to the top.
func (s *State) UndoLast() (Activity, ActivityLog, bool) {
	if len(s.Logs) == 0 {
		return Activity{}, ActivityLog{}, false
	}
	last := s.Logs[0]
	s.Logs = s.Logs[1:]

	for i := len(s.Queue) - 1; i >= 0; i-- {
		if s.Queue[i].ActivityID == last.ActivityID {
			s.Queue = moveToFront(s.Queue, i)
			break
		}
	}
	return s.Activities[last.ActivityID], last, true
}

func moveToFront(q []QueueItem, index int) []QueueItem {
	out := make([]QueueItem, 0, len(q))
	out = append(out, q[index])
	out = append(out, q[:index]...)
	return append(out, q[index+1:]...)
}

// spread inserts n instances of an activity into q, evenly spaced so that the
// activity does not come up in clumps.
func spread(q []QueueItem, activityID string, n int, now time.Time) []QueueItem {
	total := len(q) + n
	isNew := make(map[int]bool, n)
	for k := 0; k < n; k++ {
		isNew[(2*k+1)*total/(2*n)] = true
	}

	out := make([]QueueItem, 0, total)
	next, added := 0, 0
	for i := 0; i < total; i++ {
		if isNew[i] {
			out = append(out, QueueItem{InstanceID: fmt.Sprintf("q-%d-%s-%d", now.UnixMilli(), activityID, added), ActivityID: activityID})
			added++
			continue
		}
		out = append(out, q[next])
		next++
	}
	return out
}

// dropInstances removes n instances of an activity, starting from the back so
// that the order of what is coming up soon is kept.
func dropInstances(q []QueueItem, activityID string, n int) []QueueItem {
	out := append([]QueueItem(nil), q...)
	for i := len(out) - 1; i >= 0 && n > 0; i-- {
		if out[i].ActivityID == activityID {
			out = append(out[:i], out[i+1:]...)
			n--
		}
	}
	return out
}
