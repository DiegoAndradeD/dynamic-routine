package routine

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

// DayKey is the YYYY-MM-DD key used for per-day records.
func DayKey(t time.Time) string { return t.Format(dateLayout) }

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func overlap(aStart, aEnd, bStart, bEnd time.Time) time.Duration {
	start, end := aStart, aEnd
	if bStart.After(start) {
		start = bStart
	}
	if bEnd.Before(end) {
		end = bEnd
	}
	if !end.After(start) {
		return 0
	}
	return end.Sub(start)
}

// FindTrack looks a track up by id.
func (s *State) FindTrack(id string) (Track, bool) {
	for _, t := range s.Tracks {
		if t.ID == id {
			return t, true
		}
	}
	return Track{}, false
}

// AppliesOn reports whether a daily goal counts on that weekday. A track with
// no days listed counts every day.
func (t Track) AppliesOn(d time.Weekday) bool {
	if len(t.Days) == 0 {
		return true
	}
	for _, k := range t.Days {
		if k == weekdayKeys[d] {
			return true
		}
	}
	return false
}

// TargetOn is the goal for one day: zero on days off and for weekly goals.
func (s *State) TargetOn(trackID string, day time.Time) time.Duration {
	t, ok := s.FindTrack(trackID)
	if !ok || t.Period != PeriodDay || !t.AppliesOn(day.Weekday()) {
		return 0
	}
	return time.Duration(t.TargetMinutes) * time.Minute
}

// TimerRunning reports whether any timer is counting.
func (s *State) TimerRunning() bool { return s.Timer != nil }

// StartTimer starts counting time for a track. A timer already running (for
// this or another track) is stopped first, as you can only do one thing at a
// time; the closed slice is returned so the caller can tell the user.
func (s *State) StartTimer(trackID string, now time.Time) (prev RunningTimer, prevSlice time.Duration, err error) {
	if _, ok := s.FindTrack(trackID); !ok {
		return prev, 0, errors.New("that time track no longer exists")
	}
	if s.Timer != nil {
		prev = *s.Timer
		prevSlice, _ = s.StopTimer(now)
	}
	s.Timer = &RunningTimer{TrackID: trackID, Since: now}
	return prev, prevSlice, nil
}

// StopTimer closes the running slice and returns its length.
func (s *State) StopTimer(now time.Time) (time.Duration, bool) {
	if s.Timer == nil {
		return 0, false
	}
	t := *s.Timer
	s.Timer = nil
	if !now.After(t.Since) {
		return 0, true
	}
	s.TimeSessions = append(s.TimeSessions, TimeSession{ID: newID("w", now), TrackID: t.TrackID, Start: t.Since, End: now})
	return now.Sub(t.Since), true
}

// MaxLoggedMinutes bounds one manual entry: a day has 24 hours.
const MaxLoggedMinutes = 24 * 60

// LogTime records time spent by hand on a calendar day.
func (s *State) LogTime(trackID string, day time.Time, minutes int, note string, now time.Time) (TimeEntry, error) {
	if _, ok := s.FindTrack(trackID); !ok {
		return TimeEntry{}, errors.New("that time track no longer exists")
	}
	if minutes <= 0 {
		return TimeEntry{}, errors.New("time must be more than zero")
	}
	if minutes > MaxLoggedMinutes {
		return TimeEntry{}, errors.New("a single entry cannot be longer than 24h")
	}
	if dayStart(day).After(dayStart(now)) {
		return TimeEntry{}, errors.New("that day has not happened yet")
	}
	e := TimeEntry{
		ID: fmt.Sprintf("m-%d-%d", now.UnixMilli(), len(s.TimeEntries)), TrackID: trackID,
		Date: DayKey(day), Minutes: minutes, Note: strings.TrimSpace(note), LoggedAt: now,
	}
	s.TimeEntries = append(s.TimeEntries, e)
	return e, nil
}

// DeleteTimeItem removes a timer slice or a manual entry by id.
func (s *State) DeleteTimeItem(id string) bool {
	for i, e := range s.TimeEntries {
		if e.ID == id {
			s.TimeEntries = append(s.TimeEntries[:i], s.TimeEntries[i+1:]...)
			return true
		}
	}
	for i, w := range s.TimeSessions {
		if w.ID == id {
			s.TimeSessions = append(s.TimeSessions[:i], s.TimeSessions[i+1:]...)
			return true
		}
	}
	return false
}

// WorkedOn totals everything spent on a track on a calendar day: closed slices
// and the running one, clipped to the day (a slice crossing midnight is split),
// plus the manual entries.
func (s *State) WorkedOn(trackID string, day, now time.Time) time.Duration {
	var total time.Duration
	for _, it := range s.ItemsOn(trackID, day, now) {
		total += it.Duration()
	}
	if total < 0 {
		return 0
	}
	return total
}

// DayItem is one thing that adds time to a day: a timer slice (clipped to the
// day) or a manual entry.
type DayItem struct {
	ID      string
	Manual  bool
	Start   time.Time // timer slices only
	End     time.Time
	Minutes int // manual entries only
	Note    string
	Running bool
	At      time.Time // for ordering
}

func (d DayItem) Duration() time.Duration {
	if d.Manual {
		return time.Duration(d.Minutes) * time.Minute
	}
	return d.End.Sub(d.Start)
}

// ItemsOn lists what makes up a day on a track, oldest first.
func (s *State) ItemsOn(trackID string, day, now time.Time) []DayItem {
	start := dayStart(day)
	end := start.AddDate(0, 0, 1)

	var out []DayItem
	clip := func(id string, a, b time.Time, running bool) {
		if overlap(a, b, start, end) == 0 {
			return
		}
		if a.Before(start) {
			a = start
		}
		if b.After(end) {
			b = end
		}
		out = append(out, DayItem{ID: id, Start: a, End: b, Running: running, At: a})
	}
	for _, ws := range s.TimeSessions {
		if ws.TrackID == trackID {
			clip(ws.ID, ws.Start, ws.End, false)
		}
	}
	if s.Timer != nil && s.Timer.TrackID == trackID {
		clip("running", s.Timer.Since, now, true)
	}
	key := DayKey(day)
	for _, e := range s.TimeEntries {
		if e.TrackID == trackID && e.Date == key {
			out = append(out, DayItem{ID: e.ID, Manual: true, Minutes: e.Minutes, Note: e.Note, At: e.LoggedAt})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At.Before(out[j].At) })
	return out
}

// RemainingToday is how much of today's goal is still missing (zero for weekly
// goals and days off).
func (s *State) RemainingToday(trackID string, now time.Time) time.Duration {
	left := s.TargetOn(trackID, now) - s.WorkedOn(trackID, now, now)
	if left < 0 {
		return 0
	}
	return left
}

type TrackDay struct {
	Date     time.Time
	Worked   time.Duration
	Target   time.Duration
	IsToday  bool
	IsFuture bool
}

func (d TrackDay) Done() bool { return d.Target > 0 && d.Worked >= d.Target }

type TrackWeek struct {
	Track   Track
	Days    []TrackDay // Monday to Sunday
	Worked  time.Duration
	Target  time.Duration
	Balance time.Duration // daily goals only: worked minus target over the days before today
}

// Remaining is what is left of the week's goal.
func (w TrackWeek) Remaining() time.Duration {
	if left := w.Target - w.Worked; left > 0 {
		return left
	}
	return 0
}

// Week summarises Monday..Sunday for a track, for the week starting at monday.
// The balance only looks at finished days, so today's pending goal is not a
// debt yet; surpluses offset shortfalls.
func (s *State) Week(trackID string, monday, now time.Time) TrackWeek {
	t, _ := s.FindTrack(trackID)
	today := dayStart(now)

	w := TrackWeek{Track: t}
	for i := 0; i < 7; i++ {
		date := monday.AddDate(0, 0, i)
		d := TrackDay{
			Date:     date,
			Worked:   s.WorkedOn(trackID, date, now),
			Target:   s.TargetOn(trackID, date),
			IsToday:  date.Equal(today),
			IsFuture: date.After(today),
		}
		w.Days = append(w.Days, d)
		w.Worked += d.Worked
		w.Target += d.Target
		if t.Period == PeriodDay && date.Before(today) {
			w.Balance += d.Worked - d.Target
		}
	}
	if t.Period == PeriodWeek {
		w.Target = time.Duration(t.TargetMinutes) * time.Minute
	}
	return w
}

// ParseMinutes reads a duration the way people write it: "90", "90m", "1h",
// "1h30", "1h30m", "1:30" or "1.5h". A bare number is minutes.
func ParseMinutes(s string) (int, error) {
	in := strings.ToLower(strings.TrimSpace(s))
	if in == "" {
		return 0, errors.New("enter a time, like 45, 1h30 or 1:30")
	}
	bad := fmt.Errorf("cannot read %q as a time (try 45, 1h30 or 1:30)", s)

	if h, m, ok := strings.Cut(in, ":"); ok {
		hh, err1 := strconv.Atoi(h)
		mm, err2 := strconv.Atoi(m)
		if err1 != nil || err2 != nil || hh < 0 || mm < 0 || mm > 59 {
			return 0, bad
		}
		return hh*60 + mm, nil
	}
	if h, rest, ok := strings.Cut(in, "h"); ok {
		hours, err := strconv.ParseFloat(strings.TrimSpace(h), 64)
		if err != nil || hours < 0 {
			return 0, bad
		}
		rest = strings.TrimSuffix(strings.TrimSpace(rest), "m")
		mins := 0
		if rest != "" {
			if mins, err = strconv.Atoi(strings.TrimSpace(rest)); err != nil || mins < 0 {
				return 0, bad
			}
		}
		return int(hours*60+0.5) + mins, nil
	}
	n, err := strconv.Atoi(strings.TrimSuffix(in, "m"))
	if err != nil || n < 0 {
		return 0, bad
	}
	return n, nil
}
