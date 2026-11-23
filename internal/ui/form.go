package ui

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
)

type formKind int

const (
	formActivity formKind = iota
	formTrack
	formSettings
	formRule
	formLog
)

type formField struct {
	label   string
	hint    string
	value   string
	numeric bool
	choices []string // if set, the field is a choice: Left/Right cycle through them
	choice  int
	back    int // text fields: how many characters the cursor is from the end
}

func (f formField) text() string {
	if f.choices != nil {
		return f.choices[f.choice]
	}
	return f.value
}

// form is a small modal editor: a few labelled fields, Tab to move, Enter on the
// last field (or Ctrl+S) to save. It edits the plan or logs time, and shows
// validation errors in place instead of closing.
type form struct {
	kind   formKind
	title  string
	editID string // id of the thing being edited; empty when adding
	fields []formField
	focus  int
	err    string
}

func (f *form) values() []string {
	out := make([]string, len(f.fields))
	for i, fl := range f.fields {
		out[i] = strings.TrimSpace(fl.text())
	}
	return out
}

func (m *model) handleFormKey(msg tea.KeyMsg) {
	f := m.form
	last := len(f.fields) - 1
	field := &f.fields[f.focus]

	switch msg.Type {
	case tea.KeyEsc:
		m.form = nil
	case tea.KeyCtrlS:
		m.submitForm()
	case tea.KeyEnter:
		if f.focus < last {
			f.focus++
		} else {
			m.submitForm()
		}
	case tea.KeyTab, tea.KeyDown:
		f.focus = (f.focus + 1) % len(f.fields)
	case tea.KeyShiftTab, tea.KeyUp:
		f.focus = (f.focus + len(f.fields) - 1) % len(f.fields)
	case tea.KeyLeft, tea.KeyRight:
		if n := len(field.choices); n > 0 {
			step := 1
			if msg.Type == tea.KeyLeft {
				step = n - 1
			}
			field.choice = (field.choice + step) % n
		} else if msg.Type == tea.KeyLeft {
			field.back = clampInt(field.back+1, 0, len([]rune(field.value)))
		} else {
			field.back = clampInt(field.back-1, 0, len([]rune(field.value)))
		}
	case tea.KeyHome, tea.KeyCtrlA:
		field.back = len([]rune(field.value))
	case tea.KeyEnd, tea.KeyCtrlE:
		field.back = 0
	case tea.KeyBackspace:
		if r := []rune(field.value); field.choices == nil && len(r) > field.back {
			at := len(r) - field.back
			field.value = string(r[:at-1]) + string(r[at:])
		}
	case tea.KeyDelete:
		if r := []rune(field.value); field.choices == nil && field.back > 0 {
			at := len(r) - field.back
			field.value = string(r[:at]) + string(r[at+1:])
			field.back--
		}
	case tea.KeyCtrlU:
		if field.choices == nil {
			field.value, field.back = "", 0
		}
	case tea.KeySpace, tea.KeyRunes:
		if field.choices != nil {
			// space or any key also cycles a choice
			field.choice = (field.choice + 1) % len(field.choices)
			return
		}
		text := string(msg.Runes)
		if msg.Type == tea.KeySpace {
			text = " "
		}
		for _, r := range text {
			if field.numeric && !unicode.IsDigit(r) {
				continue
			}
			if unicode.IsPrint(r) {
				rs := []rune(field.value)
				at := len(rs) - field.back
				field.value = string(rs[:at]) + string(r) + string(rs[at:])
			}
		}
		f.err = ""
	}
}

// editPlan applies fn to the plan inside the freshest copy of the state and
// saves it. A plan the user got wrong comes back as an error to show in the
// form, with nothing written.
func (m *model) editPlan(fn func(*routine.Plan) error) error {
	_, err := m.apply(func(s *routine.State) (string, error) {
		p := s.Plan()
		if err := fn(&p); err != nil {
			return "", refusal(err.Error())
		}
		if err := s.ApplyPlan(p); err != nil {
			return "", refusal(err.Error())
		}
		return "", nil
	})
	return err
}

func (m *model) submitForm() {
	f := m.form
	var (
		msg string
		err error
	)
	switch f.kind {
	case formActivity:
		msg, err = m.saveActivity(f.editID, f.values())
	case formTrack:
		msg, err = m.saveTrack(f.editID, f.values())
	case formSettings:
		msg, err = m.saveSettings(f.values())
	case formRule:
		msg, err = m.saveRule(f.editID, f.values())
	case formLog:
		msg, err = m.saveLog()
	}
	if err != nil {
		f.err = err.Error()
		return
	}
	m.form = nil
	m.flash(msg)
}

// --- activities ---

func (m *model) openActivityForm(id string) {
	a := routine.Activity{TargetPerWeek: 3, RecommendedMinutes: 45}
	title := "NEW ACTIVITY"
	if cur, ok := m.state.Activities[id]; ok {
		a, title = cur, "EDIT ACTIVITY"
	}
	m.form = &form{
		kind: formActivity, title: title, editID: id,
		fields: []formField{
			{label: "Name", value: a.Name, hint: "Something you do in a sitting and tick off, e.g. Guitar practice"},
			{label: "Category", value: a.Category, hint: m.categoryHint()},
			{label: "Times per week", value: strconv.Itoa(a.TargetPerWeek), numeric: true,
				hint: "Also how many times it sits on the queue"},
			{label: "Minutes per session", value: strconv.Itoa(a.RecommendedMinutes), numeric: true,
				hint: "Recommended length of a full session"},
		},
	}
}

func (m model) categoryHint() string {
	names := make([]string, 0, len(m.state.Categories))
	for k := range m.state.Categories {
		names = append(names, k)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "Any name groups things by colour"
	}
	return "Existing: " + strings.Join(names, ", ") + " (or type a new one)"
}

func (m *model) saveActivity(id string, v []string) (string, error) {
	target, _ := strconv.Atoi(v[2])
	minutes, _ := strconv.Atoi(v[3])
	err := m.editPlan(func(p *routine.Plan) error {
		a := routine.Activity{ID: id, Name: v[0], Category: v[1], TargetPerWeek: target, RecommendedMinutes: minutes}
		for i := range p.Activities {
			if id != "" && p.Activities[i].ID == id {
				a.Weight = p.Activities[i].Weight
				p.Activities[i] = a
				return nil
			}
		}
		p.Activities = append(p.Activities, a)
		return nil
	})
	if err != nil {
		return "", err
	}
	if id == "" {
		return "Added: " + v[0], nil
	}
	return "Saved: " + v[0], nil
}

// --- time tracks ---

var periodChoices = []string{"per day", "per week"}

func (m *model) openTrackForm(id string) {
	t := routine.Track{Period: routine.PeriodDay, TargetMinutes: 120, Days: []string{"mon", "tue", "wed", "thu", "fri"}}
	title := "NEW TIME TRACK"
	if cur, ok := m.state.FindTrack(id); ok {
		t, title = cur, "EDIT TIME TRACK"
	}
	choice := 0
	if t.Period == routine.PeriodWeek {
		choice = 1
	}
	m.form = &form{
		kind: formTrack, title: title, editID: id,
		fields: []formField{
			{label: "Name", value: t.Name, hint: "Something you spend time on, e.g. Work, Studying, Practice"},
			{label: "Category", value: t.Category, hint: m.categoryHint()},
			{label: "Goal is", choices: periodChoices, choice: choice, hint: "←/→ to change: a goal for each day, or one for the whole week"},
			{label: "Goal", value: fmtInput(t.TargetMinutes), hint: "e.g. 8h, 1h30, 45. Empty = just track time, no goal"},
			{label: "Days", value: strings.Join(t.Days, ", "), hint: "For a goal per day: mon, tue... Empty = every day"},
		},
	}
}

func (m *model) saveTrack(id string, v []string) (string, error) {
	goal := 0
	if v[3] != "" {
		var err error
		if goal, err = routine.ParseMinutes(v[3]); err != nil {
			return "", err
		}
	}
	period := routine.PeriodDay
	if m.form.fields[2].choice == 1 {
		period = routine.PeriodWeek
	}
	days := strings.FieldsFunc(v[4], func(r rune) bool { return r == ',' || r == ' ' })

	err := m.editPlan(func(p *routine.Plan) error {
		t := routine.Track{ID: id, Name: v[0], Category: v[1], Period: period, TargetMinutes: goal, Days: days}
		for i := range p.Tracks {
			if id != "" && p.Tracks[i].ID == id {
				p.Tracks[i] = t
				return nil
			}
		}
		p.Tracks = append(p.Tracks, t)
		return nil
	})
	if err != nil {
		return "", err
	}
	if id == "" {
		return "Added time track: " + v[0], nil
	}
	return "Saved: " + v[0], nil
}

// --- settings ---

func (m *model) openSettingsForm() {
	m.form = &form{
		kind: formSettings, title: "SETTINGS",
		fields: []formField{
			{label: "Micro block (minutes)", value: strconv.Itoa(m.state.MicroMinutes), numeric: true,
				hint: "Length of a short session, the M key"},
		},
	}
}

func (m *model) saveSettings(v []string) (string, error) {
	micro, _ := strconv.Atoi(v[0])
	err := m.editPlan(func(p *routine.Plan) error {
		p.MicroMinutes = micro
		return nil
	})
	if err != nil {
		return "", err
	}
	return "Settings saved", nil
}

// --- rules ---

func (m *model) openRuleForm(id string) {
	r := routine.Rule{}
	title := "NEW RULE"
	for _, cur := range m.state.Rules {
		if cur.ID == id {
			r, title = cur, "EDIT RULE"
		}
	}
	m.form = &form{
		kind: formRule, title: title, editID: id,
		fields: []formField{
			{label: "When...", value: r.Trigger, hint: "The situation, e.g. I slept badly"},
			{label: "Then...", value: r.Instruction, hint: "What to do, e.g. Only do the lightest activity and rest"},
		},
	}
}

func (m *model) saveRule(id string, v []string) (string, error) {
	err := m.editPlan(func(p *routine.Plan) error {
		r := routine.Rule{ID: id, Trigger: v[0], Instruction: v[1]}
		for i := range p.Rules {
			if id != "" && p.Rules[i].ID == id {
				r.Category = p.Rules[i].Category
				p.Rules[i] = r
				return nil
			}
		}
		r.ID = fmt.Sprintf("r-%d", time.Now().UnixMilli())
		p.Rules = append(p.Rules, r)
		return nil
	})
	if err != nil {
		return "", err
	}
	return "Rule saved", nil
}

// --- logging time by hand ---

// openLogForm opens "I spent time on this" for a track and a day.
func (m *model) openLogForm(trackIdx int, day time.Time) {
	if len(m.state.Tracks) == 0 {
		m.flashError("No time tracks yet. Press 5, then T to add one.")
		return
	}
	names := make([]string, len(m.state.Tracks))
	for i, t := range m.state.Tracks {
		names[i] = t.Name
	}
	m.form = &form{
		kind: formLog, title: "LOG TIME",
		fields: []formField{
			{label: "Track", choices: names, choice: clamp(trackIdx, len(names)-1), hint: "←/→ to change"},
			{label: "Day", value: dayWord(day, m.now), hint: "today, yesterday or YYYY-MM-DD"},
			{label: "Time spent", hint: "e.g. 45, 1h30 or 1:30"},
			{label: "Note (optional)", hint: "What you did"},
		},
		focus: 2,
	}
}

func (m *model) saveLog() (string, error) {
	f := m.form
	v := f.values()
	track := m.state.Tracks[f.fields[0].choice]
	day, err := parseDay(v[1], m.now)
	if err != nil {
		return "", err
	}
	minutes, err := routine.ParseMinutes(v[2])
	if err != nil {
		return "", err
	}
	now := time.Now()
	return m.apply(func(s *routine.State) (string, error) {
		if _, err := s.LogTime(track.ID, day, minutes, v[3], now); err != nil {
			return "", refusal(err.Error())
		}
		return fmt.Sprintf("Logged %s on %s (%s total that day)", fmtDur(time.Duration(minutes)*time.Minute),
			track.Name, fmtDur(s.WorkedOn(track.ID, day, now))), nil
	})
}

// askDelete asks for confirmation before removing something.
func (m *model) askDelete(what string, remove func(*model) error) {
	m.confirmText = fmt.Sprintf("Delete %s? ", what)
	m.confirmYes = func(m *model) {
		if err := remove(m); err != nil {
			m.flashError(err.Error())
			return
		}
		m.flash("Deleted " + what)
	}
}

func (m *model) askDeleteFromPlan(what string, remove func(*routine.Plan)) {
	m.askDelete(what, func(m *model) error {
		return m.editPlan(func(p *routine.Plan) error { remove(p); return nil })
	})
}

func (m model) viewForm(w int) string {
	f := m.form
	labelW := 0
	for _, fl := range f.fields {
		if n := len([]rune(fl.label)); n > labelW {
			labelW = n
		}
	}
	valueW := w - labelW - 8
	if valueW > 70 {
		valueW = 70
	}
	if valueW < 10 {
		valueW = 10
	}

	var b strings.Builder
	b.WriteString(bold(colorAmber).Render(f.title) + "\n\n")
	for i, fl := range f.fields {
		label, value := muted(pad(fl.label, labelW)), fl.text()
		input := lipgloss.NewStyle().Background(colorSelBg).Foreground(colorText).Width(valueW)
		cursor := "  "
		if fl.choices != nil {
			value = "◂ " + value + " ▸"
		}
		if i == f.focus {
			cursor = bold(colorAmber).Render("▸ ")
			label = bold(colorAmber).Render(pad(fl.label, labelW))
			if fl.choices == nil {
				r := []rune(value)
				at := len(r) - fl.back
				value = string(r[:at]) + "▏" + string(r[at:])
			}
			input = input.Background(lipgloss.Color("#3F3F46"))
		}
		b.WriteString(cursor + label + "  " + input.Render(value) + "\n")
		if i == f.focus && fl.hint != "" {
			b.WriteString("  " + strings.Repeat(" ", labelW) + "  " + muted(fl.hint) + "\n")
		}
	}
	if f.err != "" {
		b.WriteString("\n  " + bold(colorRose).Width(w-4).Render("✖ "+f.err) + "\n")
	}
	return b.String()
}
