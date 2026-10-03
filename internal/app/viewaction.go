package app

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jaisuriya-11/tsuzuri/internal/preview"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Interactive views: clicks in the preview edit the note's Markdown source.

// ---------------------------------------------------------------------------
// Generic prompt and menu dialogs

type promptDialog struct {
	title    string
	hint     string
	input    textinput.Model
	onSubmit func(m *Model, value string) tea.Cmd
}

type menuDialog struct {
	title    string
	items    []string
	sel      int
	onChoose func(m *Model, i int) tea.Cmd
}

func (m *Model) prompt(title, value, hint string, onSubmit func(*Model, string) tea.Cmd) tea.Cmd {
	in := textinput.New()
	in.Prompt = ""
	in.CharLimit = 300
	in.SetValue(value)
	in.CursorEnd()
	d := &promptDialog{title: title, hint: hint, input: in, onSubmit: onSubmit}
	m.promptBox = d
	return d.input.Focus()
}

func (m *Model) menu(title string, items []string, onChoose func(*Model, int) tea.Cmd) {
	m.menuBox = &menuDialog{title: title, items: items, onChoose: onChoose}
}

func (d *promptDialog) update(m *Model, msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc", "ctrl+c":
			m.promptBox = nil
			return nil
		case "enter":
			m.promptBox = nil
			return d.onSubmit(m, strings.TrimSpace(d.input.Value()))
		}
	}
	if mm, ok := msg.(tea.MouseMsg); ok {
		if mm.Action == tea.MouseActionPress && mm.Button == tea.MouseButtonLeft {
			x, y, w, h := d.geom(m)
			if mm.X < x || mm.X >= x+w || mm.Y < y || mm.Y >= y+h {
				m.promptBox = nil
			}
		}
		return nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return cmd
}

func (d *promptDialog) geom(m *Model) (x, y, w, h int) {
	w = min(60, m.width-4)
	h = 6
	if d.hint != "" {
		h++
	}
	x, y = ui.Center(m.width, m.height, w, h)
	return
}

func (d *promptDialog) view(m *Model) (string, int, int) {
	th := m.theme
	x, y, w, _ := d.geom(m)
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	field := lipgloss.NewStyle().Background(th.Bg2)
	d.input.TextStyle = field.Foreground(th.Fg)
	d.input.Cursor.Style = lipgloss.NewStyle().Foreground(th.Blue)
	d.input.Cursor.TextStyle = field.Foreground(th.Fg)
	d.input.Width = inner - 6
	rows := []string{
		bg.Foreground(th.Blue).Render(" \U000f03eb ") + bg.Foreground(th.Fg).Bold(true).Render(ui.Truncate(d.title, inner-4)),
		"",
		bg.Render(" ") + ui.FitLine(field.Foreground(th.Blue).Render(" ▏")+d.input.View(), inner-2, field) + bg.Render(" "),
	}
	if d.hint != "" {
		rows = append(rows, bg.Foreground(th.GreyFg2).Render(" "+ui.Truncate(d.hint, inner-2)))
	}
	rows = append(rows, bg.Foreground(th.GreyFg).Render(" Enter save · Esc cancel"))
	return panel(th, rows, inner), x, y
}

func (d *menuDialog) geom(m *Model) (x, y, w, h int) {
	w = 34
	for _, it := range d.items {
		w = max(w, lipgloss.Width(it)+8)
	}
	w = min(w, m.width-4)
	h = len(d.items) + 4
	x, y = ui.Center(m.width, m.height, w, h)
	return
}

func (d *menuDialog) update(m *Model, msg tea.Msg) tea.Cmd {
	choose := func(i int) tea.Cmd {
		m.menuBox = nil
		return d.onChoose(m, i)
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q", "ctrl+c":
			m.menuBox = nil
		case "up", "k", "shift+tab":
			d.sel = (d.sel - 1 + len(d.items)) % len(d.items)
		case "down", "j", "tab":
			d.sel = (d.sel + 1) % len(d.items)
		case "enter", " ":
			return choose(d.sel)
		default:
			if n, err := strconv.Atoi(msg.String()); err == nil && n >= 1 && n <= len(d.items) {
				return choose(n - 1)
			}
		}
	case tea.MouseMsg:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		x, y, w, h := d.geom(m)
		if msg.X < x || msg.X >= x+w || msg.Y < y || msg.Y >= y+h {
			m.menuBox = nil
			return nil
		}
		if i := msg.Y - y - 3; msg.Button == tea.MouseButtonLeft && i >= 0 && i < len(d.items) {
			return choose(i)
		}
	}
	return nil
}

func (d *menuDialog) view(m *Model) (string, int, int) {
	th := m.theme
	x, y, w, _ := d.geom(m)
	inner := w - 2
	bg := lipgloss.NewStyle().Background(th.DarkerBg)
	rows := []string{bg.Foreground(th.Fg).Bold(true).Render(" " + ui.Truncate(d.title, inner-2)), ""}
	for i, it := range d.items {
		st := bg.Foreground(th.Fg)
		marker := "  "
		if i == d.sel {
			st = lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Blue).Bold(true)
			marker = "▌ "
		}
		rows = append(rows, ui.FitLine(st.Render(marker+fmt.Sprintf("%d  ", i+1)+it), inner, st))
	}
	rows = append(rows, bg.Foreground(th.GreyFg).Render(" ↑↓ Enter · Esc"))
	return panel(th, rows, inner), x, y
}

// ---------------------------------------------------------------------------
// Editing the note's lines

func (m *Model) docLines() []string { return strings.Split(m.content.Value(), "\n") }

func (m *Model) setDocLines(lines []string) {
	m.content.ReplaceText(strings.Join(lines, "\n"))
	m.preview.SetContent(m.content.Value())
	m.refreshModified()
}

func insertAt(lines []string, at int, add ...string) []string {
	at = max(0, min(at, len(lines)))
	out := append([]string{}, lines[:at]...)
	out = append(out, add...)
	return append(out, lines[at:]...)
}

func removeAt(lines []string, at, n int) []string {
	if at < 0 || at >= len(lines) {
		return lines
	}
	end := min(at+n, len(lines))
	return append(append([]string{}, lines[:at]...), lines[end:]...)
}

// ---------------------------------------------------------------------------
// Dispatch

func (m *Model) handleViewHit(h preview.Hit) tea.Cmd {
	if m.activeBuffer() == nil {
		return nil
	}
	switch {
	case strings.HasPrefix(h.Kind, "board:"):
		return m.boardHit(h)
	case strings.HasPrefix(h.Kind, "cal:"):
		return m.calendarHit(h)
	case strings.HasPrefix(h.Kind, "tl:"):
		return m.timelineHit(h)
	case strings.HasPrefix(h.Kind, "chart:"):
		return m.chartHit(h)
	case strings.HasPrefix(h.Kind, "form:"):
		return m.formHit(h)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Board

type boardCol struct {
	name     string
	line     int // heading line, -1 if implicit
	endLine  int // where new cards go (before the next heading / fence)
	cardLine []int
}

func boardColumns(lines []string, h preview.Hit) []boardCol {
	var cols []boardCol
	for i := h.Block + 1; i < h.End && i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		switch {
		case strings.HasPrefix(t, "#"):
			cols = append(cols, boardCol{name: strings.TrimSpace(strings.TrimLeft(t, "#")), line: i})
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* "):
			if len(cols) == 0 {
				cols = append(cols, boardCol{name: "Cards", line: -1})
			}
			cols[len(cols)-1].cardLine = append(cols[len(cols)-1].cardLine, i)
		}
	}
	for c := range cols {
		end := h.End
		if c+1 < len(cols) {
			end = cols[c+1].line
		}
		// New cards go after the last non-blank line of the column.
		for end-1 > cols[c].line && end-1 > h.Block && strings.TrimSpace(lines[end-1]) == "" {
			end--
		}
		cols[c].endLine = end
	}
	return cols
}

// cardExtent is the card line plus its description lines.
func cardExtent(lines []string, at, end int) int {
	n := 1
	for i := at + 1; i < end && i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || t == "" {
			break
		}
		n++
	}
	return n
}

func (m *Model) boardHit(h preview.Hit) tea.Cmd {
	lines := m.docLines()
	cols := boardColumns(lines, h)
	switch h.Kind {
	case "board:add":
		return m.prompt("New card in "+h.Arg, "", "Lines you add under the card in the editor become its description", func(m *Model, v string) tea.Cmd {
			if v == "" {
				return nil
			}
			lines := m.docLines()
			cols := boardColumns(lines, h)
			at := h.End
			if h.Index < len(cols) {
				at = cols[h.Index].endLine
			}
			m.setDocLines(insertAt(lines, at, "- "+v))
			return nil
		})

	case "board:card":
		items := []string{"Rename"}
		var targets []int
		for i, c := range cols {
			if i != h.Index {
				items = append(items, "Move to → "+c.name)
				targets = append(targets, i)
			}
		}
		items = append(items, "Delete")
		m.menu(h.Arg, items, func(m *Model, choice int) tea.Cmd {
			lines := m.docLines()
			n := cardExtent(lines, h.Line, h.End)
			switch {
			case choice == 0:
				return m.prompt("Rename card", h.Arg, "", func(m *Model, v string) tea.Cmd {
					if v == "" {
						return nil
					}
					lines := m.docLines()
					indent := lines[h.Line][:len(lines[h.Line])-len(strings.TrimLeft(lines[h.Line], " "))]
					m.setDocLines(append(append(lines[:h.Line:h.Line], indent+"- "+v), lines[h.Line+1:]...))
					return nil
				})
			case choice == len(items)-1:
				m.setDocLines(removeAt(lines, h.Line, n))
				m.setStatus("Card deleted")
			default:
				card := append([]string{}, lines[h.Line:h.Line+n]...)
				lines = removeAt(lines, h.Line, n)
				hh := h
				hh.End -= n
				cols := boardColumns(lines, hh)
				to := targets[choice-1]
				if to < len(cols) {
					m.setDocLines(insertAt(lines, cols[to].endLine, card...))
					m.setStatus("Moved to " + cols[to].name)
				}
			}
			return nil
		})

	case "board:col":
		if h.Line < 0 {
			return nil
		}
		m.menu(h.Arg, []string{"Add card", "Rename column", "Delete column and its cards"}, func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				hh := h
				hh.Kind = "board:add"
				return m.boardHit(hh)
			case 1:
				return m.prompt("Rename column", h.Arg, "", func(m *Model, v string) tea.Cmd {
					if v != "" {
						lines := m.docLines()
						hashes := strings.TrimSpace(lines[h.Line])
						hashes = hashes[:len(hashes)-len(strings.TrimLeft(hashes, "#"))]
						lines[h.Line] = hashes + " " + v
						m.setDocLines(lines)
					}
					return nil
				})
			default:
				lines := m.docLines()
				cols := boardColumns(lines, h)
				if h.Index < len(cols) {
					end := h.End
					if h.Index+1 < len(cols) {
						end = cols[h.Index+1].line
					}
					m.setDocLines(removeAt(lines, h.Line, end-h.Line))
				}
			}
			return nil
		})
	}
	return nil
}

// ---------------------------------------------------------------------------
// Calendar

func (m *Model) calendarHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case "cal:day":
		return m.prompt("New event on "+h.Arg, "", "", func(m *Model, v string) tea.Cmd {
			if v != "" {
				m.setDocLines(insertAt(m.docLines(), h.End, h.Arg+": "+v))
			}
			return nil
		})
	case "cal:event":
		date, _, _ := strings.Cut(strings.TrimSpace(m.docLines()[h.Line]), ":")
		m.menu(h.Arg, []string{"Rename", "Move to another date", "Delete"}, func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				return m.prompt("Rename event", h.Arg, "", func(m *Model, v string) tea.Cmd {
					if v != "" {
						lines := m.docLines()
						lines[h.Line] = strings.TrimSpace(date) + ": " + v
						m.setDocLines(lines)
					}
					return nil
				})
			case 1:
				from, _ := time.Parse("2006-01-02", strings.TrimSpace(date))
				m.pickDate("Move \""+h.Arg+"\" to", from, time.Time{}, func(m *Model, d time.Time) tea.Cmd {
					lines := m.docLines()
					lines[h.Line] = d.Format("2006-01-02") + ": " + h.Arg
					m.setDocLines(lines)
					return nil
				})
			default:
				m.setDocLines(removeAt(m.docLines(), h.Line, 1))
			}
			return nil
		})
	}
	return nil
}

// ---------------------------------------------------------------------------
// Timeline

func parseSpan(line string) (name string, start, end time.Time, ok bool) {
	n, span, found := strings.Cut(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- ")), ":")
	if !found {
		return "", start, end, false
	}
	a, b, _ := strings.Cut(span, "->")
	var err error
	if start, err = time.Parse("2006-01-02", strings.TrimSpace(a)); err != nil {
		return "", start, end, false
	}
	if end, err = time.Parse("2006-01-02", strings.TrimSpace(b)); err != nil || end.Before(start) {
		end = start
	}
	return strings.TrimSpace(n), start, end, true
}

func spanLine(name string, start, end time.Time) string {
	return name + ": " + start.Format("2006-01-02") + " -> " + end.Format("2006-01-02")
}

func (m *Model) timelineHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case "tl:add":
		return m.prompt("New timeline item", "", "Next you'll pick the start and end dates", func(m *Model, v string) tea.Cmd {
			if v == "" {
				return nil
			}
			m.pickDates(v, time.Time{}, time.Time{}, func(m *Model, start, end time.Time) {
				m.setDocLines(insertAt(m.docLines(), h.End, spanLine(v, start, end)))
			})
			return nil
		})
	case "tl:item":
		items := []string{"Move 1 day later", "Move 1 day earlier", "Move 1 week later", "Move 1 week earlier",
			"Extend by 1 day", "Shorten by 1 day", "Set dates…", "Rename", "Delete"}
		m.menu(h.Arg, items, func(m *Model, choice int) tea.Cmd {
			lines := m.docLines()
			name, start, end, ok := parseSpan(lines[h.Line])
			if !ok {
				return nil
			}
			shift := func(a, b int) {
				start, end = start.AddDate(0, 0, a), end.AddDate(0, 0, b)
				if end.Before(start) {
					end = start
				}
				lines[h.Line] = spanLine(name, start, end)
				m.setDocLines(lines)
			}
			switch choice {
			case 0:
				shift(1, 1)
			case 1:
				shift(-1, -1)
			case 2:
				shift(7, 7)
			case 3:
				shift(-7, -7)
			case 4:
				shift(0, 1)
			case 5:
				shift(0, -1)
			case 6:
				m.pickDates(name, start, end, func(m *Model, s, e time.Time) {
					lines := m.docLines()
					lines[h.Line] = spanLine(name, s, e)
					m.setDocLines(lines)
				})
			case 7:
				return m.prompt("Rename", name, "", func(m *Model, v string) tea.Cmd {
					if v != "" {
						lines := m.docLines()
						lines[h.Line] = spanLine(v, start, end)
						m.setDocLines(lines)
					}
					return nil
				})
			case 8:
				m.setDocLines(removeAt(lines, h.Line, 1))
			}
			return nil
		})
	}
	return nil
}

// pickDates asks for a start date, then an end date on or after it.
func (m *Model) pickDates(name string, start, end time.Time, done func(*Model, time.Time, time.Time)) {
	m.pickDate("Start date · "+name, start, time.Time{}, func(m *Model, s time.Time) tea.Cmd {
		initial := end
		if initial.IsZero() || initial.Before(s) {
			initial = s
		}
		m.pickDate("End date · "+name, initial, s, func(m *Model, e time.Time) tea.Cmd {
			done(m, s, e)
			return nil
		})
		return nil
	})
}

// ---------------------------------------------------------------------------
// Charts

func (m *Model) chartHit(h preview.Hit) tea.Cmd {
	switch h.Kind {
	case "chart:add":
		return m.prompt("Add a value", "", "Label: number, e.g. May: 18", func(m *Model, v string) tea.Cmd {
			if label, num, ok := strings.Cut(v, ":"); ok {
				if _, err := strconv.ParseFloat(strings.TrimSpace(num), 64); err == nil {
					m.setDocLines(insertAt(m.docLines(), h.End, strings.TrimSpace(label)+": "+strings.TrimSpace(num)))
					return nil
				}
			}
			m.setError("Use Label: number")
			return nil
		})
	case "chart:value":
		_, current, _ := strings.Cut(m.docLines()[h.Line], ":")
		m.menu(h.Arg+" ="+current, []string{"Edit value", "Rename label", "Delete"}, func(m *Model, choice int) tea.Cmd {
			switch choice {
			case 0:
				return m.prompt("Value for "+h.Arg, strings.TrimSpace(current), "", func(m *Model, v string) tea.Cmd {
					if _, err := strconv.ParseFloat(v, 64); err != nil {
						m.setError("Not a number: " + v)
						return nil
					}
					lines := m.docLines()
					lines[h.Line] = h.Arg + ": " + v
					m.setDocLines(lines)
					return nil
				})
			case 1:
				return m.prompt("Rename "+h.Arg, h.Arg, "", func(m *Model, v string) tea.Cmd {
					if v != "" {
						lines := m.docLines()
						lines[h.Line] = v + ":" + current
						m.setDocLines(lines)
					}
					return nil
				})
			default:
				m.setDocLines(removeAt(m.docLines(), h.Line, 1))
			}
			return nil
		})
	}
	return nil
}

// ---------------------------------------------------------------------------
// Forms

// questionTypes are the choices offered when adding a form question:
// label shown, type tag written in the Markdown, whether it has options.
var questionTypes = []struct {
	label, tag string
	options    bool
}{
	{"Short answer", "", false},
	{"Long answer", "long", false},
	{"Single choice  ○", "choice", true},
	{"Multiple choice  ☐", "multi", true},
	{"Rating  ☆☆☆☆☆", "rating", false},
	{"Date", "date", false},
	{"Email", "email", false},
}

// chooseQuestionType asks for a question type (and options when needed) and
// hands the finished "? …" line to done.
func (m *Model) chooseQuestionType(text, options string, done func(*Model, string)) tea.Cmd {
	labels := make([]string, len(questionTypes))
	for i, t := range questionTypes {
		labels[i] = t.label
	}
	m.menu("Type of \""+text+"\"", labels, func(m *Model, i int) tea.Cmd {
		t := questionTypes[i]
		line := "? " + text
		if t.tag != "" {
			line += " (" + t.tag + ")"
		}
		if !t.options {
			done(m, line)
			return nil
		}
		if options == "" {
			options = "Option 1 | Option 2 | Option 3"
		}
		return m.prompt("Options for \""+text+"\"", options, "Separate options with | or commas", func(m *Model, v string) tea.Cmd {
			var opts []string
			for _, o := range strings.FieldsFunc(v, func(r rune) bool { return r == '|' || r == ',' }) {
				if o = strings.TrimSpace(o); o != "" {
					opts = append(opts, o)
				}
			}
			if len(opts) == 0 {
				opts = []string{"Option 1"}
			}
			done(m, line+": "+strings.Join(opts, " | "))
			return nil
		})
	})
	return nil
}

// answerLine returns the "= answer" line belonging to question line q, or -1.
func answerLine(lines []string, q int) int {
	if q+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[q+1]), "=") {
		return q + 1
	}
	return -1
}

func (m *Model) setAnswer(q int, value string) {
	lines := m.docLines()
	at := answerLine(lines, q)
	switch {
	case value == "" && at >= 0:
		lines = removeAt(lines, at, 1)
	case value == "":
	case at >= 0:
		lines[at] = "  = " + value
	default:
		lines = insertAt(lines, q+1, "  = "+value)
	}
	m.setDocLines(lines)
}

func (m *Model) currentAnswer(q int) string {
	lines := m.docLines()
	if at := answerLine(lines, q); at >= 0 {
		return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[at]), "="))
	}
	return ""
}

func questionOptions(line string) []string {
	_, opts, _ := strings.Cut(line, ":")
	var out []string
	for _, o := range strings.Split(opts, "|") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func (m *Model) formHit(h preview.Hit) tea.Cmd {
	lines := m.docLines()
	switch h.Kind {
	case "form:title", "form:desc":
		key, label := "title", "Form title"
		if h.Kind == "form:desc" {
			key, label = "description", "Description"
		}
		return m.prompt(label, h.Arg, "", func(m *Model, v string) tea.Cmd {
			lines := m.docLines()
			if h.Line >= 0 {
				lines[h.Line] = key + ": " + v
			} else {
				lines = insertAt(lines, h.Block+1, key+": "+v)
			}
			m.setDocLines(lines)
			return nil
		})

	case "form:text":
		if strings.Contains(strings.ToLower(lines[h.Line]), "(date)") {
			current, _ := time.Parse("2006-01-02", m.currentAnswer(h.Line))
			m.pickDate(h.Arg, current, time.Time{}, func(m *Model, d time.Time) tea.Cmd {
				m.setAnswer(h.Line, d.Format("2006-01-02"))
				return nil
			})
			return nil
		}
		return m.prompt(h.Arg, m.currentAnswer(h.Line), "Your answer (saved in the note as \"= answer\")", func(m *Model, v string) tea.Cmd {
			m.setAnswer(h.Line, v)
			return nil
		})

	case "form:option":
		opts := questionOptions(lines[h.Line])
		if h.Index < len(opts) {
			v := opts[h.Index]
			if m.currentAnswer(h.Line) == v {
				v = ""
			}
			m.setAnswer(h.Line, v)
		}

	case "form:multi":
		opts := questionOptions(lines[h.Line])
		if h.Index < len(opts) {
			picked := map[string]bool{}
			for _, a := range strings.Split(m.currentAnswer(h.Line), ",") {
				if a = strings.TrimSpace(a); a != "" {
					picked[a] = true
				}
			}
			picked[opts[h.Index]] = !picked[opts[h.Index]]
			var keep []string
			for _, o := range opts {
				if picked[o] {
					keep = append(keep, o)
				}
			}
			m.setAnswer(h.Line, strings.Join(keep, ", "))
		}

	case "form:rating":
		v := strconv.Itoa(h.Index)
		if m.currentAnswer(h.Line) == v {
			v = ""
		}
		m.setAnswer(h.Line, v)

	case "form:addopt":
		return m.prompt("New option for "+h.Arg, "", "", func(m *Model, v string) tea.Cmd {
			if v == "" {
				return nil
			}
			lines := m.docLines()
			if strings.Contains(lines[h.Line], ":") {
				lines[h.Line] = strings.TrimRight(lines[h.Line], " ") + " | " + v
			} else {
				lines[h.Line] += ": " + v
			}
			m.setDocLines(lines)
			return nil
		})

	case "form:question":
		required := strings.HasPrefix(strings.TrimSpace(lines[h.Line]), "?*")
		toggle := "Make required"
		if required {
			toggle = "Make optional"
		}
		const (
			rename = iota
			changeType
			toggleRequired
			clearAnswer
			deleteQuestion
		)
		m.menu(h.Arg, []string{"Rename", "Change type", toggle, "Clear answer", "Delete question"}, func(m *Model, choice int) tea.Cmd {
			lines := m.docLines()
			switch choice {
			case rename:
				return m.prompt("Rename question", h.Arg, "", func(m *Model, v string) tea.Cmd {
					if v == "" {
						return nil
					}
					lines := m.docLines()
					lines[h.Line] = strings.Replace(lines[h.Line], h.Arg, v, 1)
					m.setDocLines(lines)
					return nil
				})
			case changeType:
				prefix := "? "
				if required {
					prefix = "?* "
				}
				opts := strings.Join(questionOptions(lines[h.Line]), " | ")
				return m.chooseQuestionType(h.Arg, opts, func(m *Model, line string) {
					lines := m.docLines()
					lines[h.Line] = prefix + strings.TrimPrefix(line, "? ")
					if at := answerLine(lines, h.Line); at >= 0 {
						lines = removeAt(lines, at, 1) // the old answer may not fit the new type
					}
					m.setDocLines(lines)
				})
			case toggleRequired:
				t := strings.TrimSpace(lines[h.Line])
				if required {
					lines[h.Line] = "?" + strings.TrimPrefix(t, "?*")
				} else {
					lines[h.Line] = "?*" + strings.TrimPrefix(t, "?")
				}
				m.setDocLines(lines)
			case clearAnswer:
				m.setAnswer(h.Line, "")
			case deleteQuestion:
				n := 1
				if answerLine(lines, h.Line) >= 0 {
					n = 2
				}
				m.setDocLines(removeAt(lines, h.Line, n))
			}
			return nil
		})

	case "form:addq":
		return m.prompt("New question", "", "Next you'll pick its type (text, choice, rating…)", func(m *Model, v string) tea.Cmd {
			if v == "" {
				return nil
			}
			return m.chooseQuestionType(v, "", func(m *Model, line string) {
				m.setDocLines(insertAt(m.docLines(), h.End, line))
			})
		})
	}
	return nil
}
