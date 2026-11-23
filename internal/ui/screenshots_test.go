package ui

// Regenerates the screenshots in docs/screenshots from the real views:
//
//	SCREENSHOTS_DIR=$PWD/docs/screenshots go test ./internal/ui -run TestScreenshots
//
// Without SCREENSHOTS_DIR the test is skipped. The data is a fixed demo (the
// example plan plus a made-up week), so the output is reproducible.

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"

	"github.com/DiegoAndradeD/dynamic-routine/internal/routine"
	"github.com/DiegoAndradeD/dynamic-routine/internal/store"
	"github.com/DiegoAndradeD/dynamic-routine/internal/vault"
)

// demoNow is a Wednesday afternoon.
var demoNow = time.Date(2026, 11, 18, 15, 42, 0, 0, time.UTC)

func demoModel(t *testing.T) model {
	t.Helper()
	st, err := routine.NewState(routine.DefaultPlan())
	if err != nil {
		t.Fatal(err)
	}
	p := st.Plan()
	p.Rules = routine.ExampleRules()
	if err := st.ApplyPlan(p); err != nil {
		t.Fatal(err)
	}

	at := func(day, h, m int) time.Time { return time.Date(2026, 11, day, h, m, 0, 0, time.UTC) }
	// Activities done earlier this week.
	for _, c := range []struct {
		when time.Time
		kind string
	}{
		{at(16, 7, 40), routine.SessionFull}, {at(16, 12, 55), routine.SessionMicro}, {at(16, 21, 10), routine.SessionFull},
		{at(17, 8, 5), routine.SessionFull}, {at(17, 20, 30), routine.SessionFull},
		{at(18, 7, 50), routine.SessionFull}, {at(18, 13, 20), routine.SessionMicro},
	} {
		st.Complete(c.kind, c.when)
	}
	// Time: timer slices and hand-logged entries.
	slice := func(track string, from, to time.Time) {
		st.StartTimer(track, from)
		st.StopTimer(to)
	}
	slice("focus", at(16, 9, 0), at(16, 11, 30))
	st.LogTime("focus", at(16, 0, 0), 30, "Planning the week", at(16, 17, 0))
	slice("focus", at(17, 10, 0), at(17, 11, 45))
	slice("focus", at(18, 9, 30), at(18, 10, 20))
	st.LogTime("outdoors", at(16, 0, 0), 75, "Run by the river", at(16, 18, 30))
	st.LogTime("outdoors", at(17, 0, 0), 40, "", at(17, 19, 0))
	st.StartTimer("focus", at(18, 15, 0)) // still running

	s := store.New("~/Sync/routine/state.rtn", vault.New([]byte("demo")))
	m := newModel(s, st, false)
	m.now = demoNow
	m.dayCursor = todayIndex(demoNow)
	m.flashUntil = demoNow.Add(time.Hour)
	return m
}

func TestScreenshots(t *testing.T) {
	dir := os.Getenv("SCREENSHOTS_DIR")
	if dir == "" {
		t.Skip("set SCREENSHOTS_DIR to regenerate the screenshots")
	}
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(prev)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	base := demoModel(t)
	base.flashMsg = "Done: Stretching (full)"

	type shot struct {
		name       string
		cols, rows int
		build      func(model) model
		title      string
	}
	shots := []shot{
		{name: "dashboard", cols: 132, rows: 32, title: "routine", build: func(m model) model { return m }},
		{name: "dashboard-narrow", cols: 80, rows: 30, title: "routine", build: func(m model) model { return m }},
		{name: "time", cols: 132, rows: 30, title: "routine · Time", build: func(m model) model {
			m.flashMsg = "Timer started: Focus time • today 50m/2h00"
			return press(m, "2")
		}},
		{name: "history", cols: 132, rows: 30, title: "routine · History", build: func(m model) model { return press(m, "3") }},
		{name: "plan", cols: 132, rows: 24, title: "routine · Plan", build: func(m model) model { return press(m, "5", "down", "down") }},
		{name: "rules", cols: 132, rows: 18, title: "routine · Rules", build: func(m model) model { return press(m, "4") }},
		{name: "form-track", cols: 110, rows: 16, title: "routine · New time track", build: func(m model) model {
			m = press(m, "5", "t", "Studying", "enter", "focus", "enter", "right", "enter", "ctrl+u", "6h")
			m.flashMsg = ""
			return m
		}},
		{name: "form-log", cols: 110, rows: 14, title: "routine · Log time", build: func(m model) model {
			m = press(m, "2", "a", "ctrl+u", "1h30", "enter", "Reviewing the design")
			m.flashMsg = ""
			return m
		}},
		{name: "help", cols: 110, rows: 40, title: "routine · Help", build: func(m model) model { return press(m, "?") }},
		{name: "welcome", cols: 110, rows: 24, title: "routine", build: func(m model) model {
			m.welcome = true
			m.flashMsg = ""
			return m
		}},
	}

	for _, sh := range shots {
		m := sh.build(base)
		m.width, m.height = sh.cols, sh.rows
		svg := ansiToSVG(m.View(), sh.cols, sh.rows, sh.title)
		path := filepath.Join(dir, sh.name+".svg")
		if err := os.WriteFile(path, []byte(svg), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s (%d bytes)", path, len(svg))
	}
}

// --- ANSI -> SVG ---

type cell struct {
	r      rune
	fg, bg string // "" = terminal default
	bold   bool
	skip   bool // second half of a wide character
}

const (
	defFg   = "#E5E5E5"
	defBg   = "#0E1116"
	cellW   = 8.4
	cellH   = 19.0
	padX    = 18.0
	padTop  = 48.0
	padBot  = 16.0
	fontPx  = 14
	fontFam = "'JetBrains Mono','Fira Code','DejaVu Sans Mono',Menlo,Consolas,monospace"
)

func parseANSI(s string, cols, rows int) [][]cell {
	grid := make([][]cell, rows)
	for i := range grid {
		grid[i] = make([]cell, cols)
		for j := range grid[i] {
			grid[i][j] = cell{r: ' '}
		}
	}
	var fg, bg string
	bold := false
	row, col := 0, 0

	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == 0x1b && i+1 < len(rs) && rs[i+1] == '[':
			j := i + 2
			for j < len(rs) && (rs[j] < '@' || rs[j] > '~') {
				j++
			}
			if j < len(rs) && rs[j] == 'm' {
				fg, bg, bold = applySGR(string(rs[i+2:j]), fg, bg, bold)
			}
			i = j
		case r == '\n':
			row, col = row+1, 0
		case r == '\r':
			col = 0
		default:
			w := runewidth.RuneWidth(r)
			if w == 0 || row >= rows || col+w > cols {
				continue
			}
			grid[row][col] = cell{r: r, fg: fg, bg: bg, bold: bold}
			if w == 2 {
				grid[row][col+1] = cell{skip: true, bg: bg}
			}
			col += w
		}
	}
	return grid
}

func applySGR(params, fg, bg string, bold bool) (string, string, bool) {
	if params == "" {
		return "", "", false
	}
	p := strings.Split(params, ";")
	num := func(k int) int { n, _ := strconv.Atoi(p[k]); return n }
	for k := 0; k < len(p); k++ {
		switch n := num(k); {
		case n == 0:
			fg, bg, bold = "", "", false
		case n == 1:
			bold = true
		case n == 22:
			bold = false
		case n == 39:
			fg = ""
		case n == 49:
			bg = ""
		case (n == 38 || n == 48) && k+4 < len(p) && p[k+1] == "2":
			c := fmt.Sprintf("#%02x%02x%02x", num(k+2), num(k+3), num(k+4))
			if n == 38 {
				fg = c
			} else {
				bg = c
			}
			k += 4
		}
	}
	return fg, bg, bold
}

func ansiToSVG(ansi string, cols, rows int, title string) string {
	grid := parseANSI(ansi, cols, rows)
	width := padX*2 + float64(cols)*cellW
	height := padTop + float64(rows)*cellH + padBot

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" role="img" aria-label="%s">`+"\n",
		width, height, width, height, esc(title))
	fmt.Fprintf(&b, `<rect width="%.0f" height="%.0f" rx="10" fill="%s"/>`+"\n", width, height, defBg)
	fmt.Fprintf(&b, `<path d="M0 10a10 10 0 0 1 10-10h%.0fa10 10 0 0 1 10 10v26H0z" fill="#171B22"/>`+"\n", width-20)
	for i, c := range []string{"#FF5F56", "#FFBD2E", "#27C93F"} {
		fmt.Fprintf(&b, `<circle cx="%d" cy="18" r="6" fill="%s"/>`+"\n", 20+i*20, c)
	}
	fmt.Fprintf(&b, `<text x="%.0f" y="23" fill="#8B949E" font-family="%s" font-size="13" text-anchor="middle">%s</text>`+"\n",
		width/2, fontFam, esc(title))
	fmt.Fprintf(&b, `<g font-family="%s" font-size="%d">`+"\n", fontFam, fontPx)

	for y, line := range grid {
		top := padTop + float64(y)*cellH

		// backgrounds: runs of the same colour
		for x := 0; x < cols; {
			if line[x].bg == "" {
				x++
				continue
			}
			end := x
			for end < cols && line[end].bg == line[x].bg {
				end++
			}
			fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s"/>`+"\n",
				padX+float64(x)*cellW, top, float64(end-x)*cellW, cellH, line[x].bg)
			x = end
		}

		// text: one element per word, so spacing never depends on the font
		for x := 0; x < cols; {
			c := line[x]
			if c.skip || c.r == ' ' {
				x++
				continue
			}
			var word strings.Builder
			end, span := x, 0
			for end < cols && !line[end].skip && line[end].r != ' ' && line[end].fg == c.fg && line[end].bold == c.bold {
				word.WriteRune(line[end].r)
				w := runewidth.RuneWidth(line[end].r)
				end += w
				span += w
			}
			if span == 0 {
				x++
				continue
			}
			fill := c.fg
			if fill == "" {
				fill = defFg
			}
			weight := ""
			if c.bold {
				weight = ` font-weight="bold"`
			}
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" fill="%s"%s textLength="%.1f" lengthAdjust="spacing" xml:space="preserve">%s</text>`+"\n",
				padX+float64(x)*cellW, top+14, fill, weight, float64(span)*cellW, esc(word.String()))
			x = end
		}
	}
	b.WriteString("</g>\n</svg>\n")
	return b.String()
}

func esc(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(s)
}
