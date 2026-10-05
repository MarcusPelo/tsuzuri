package preview

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jaisuriya-11/tsuzuri/internal/theme"
	"github.com/jaisuriya-11/tsuzuri/internal/ui"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// now is replaceable in tests.
var now = time.Now

const dateLayout = "2006-01-02"

// Hit is a clickable region of a rendered view block. Inside a renderer Row
// is relative to the block's first output line and Line to the block body;
// the compiler turns both into absolute positions (preview row, document
// line).
type Hit struct {
	Row, H int // first row and height (rows)
	X0, X1 int // columns [X0, X1)
	Kind   string
	Line   int // source line of the element (-1 if none)
	Index  int
	Arg    string
	Block  int // document line of the opening fence
	End    int // document line of the closing fence
}

func addHit(hs *[]Hit, h Hit) {
	if h.H == 0 {
		h.H = 1
	}
	*hs = append(*hs, h)
}

// CalendarView shifts every calendar block while browsing the preview.
type CalendarView struct {
	Shift int  // months forward (negative = back)
	Today bool // start from the current month instead of the block's
	// Folded holds the keys of collapsed headings / code blocks.
	Folded map[string]bool
	// FoldAll collapses every heading and code block (zM).
	FoldAll bool
}

// calendarNav is the clickable header on every calendar.
const calendarNav = "‹  Today  ›"

// renderBlock draws the special fenced blocks (board, calendar, timeline,
// chart, form). ok is false for ordinary code.
func renderBlock(lang string, body []string, th theme.Theme, width int, cal CalendarView) ([]string, []Hit, bool) {
	var hs []Hit
	var out []string
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "board", "kanban":
		out = renderBoard(body, th, width, &hs)
	case "calendar":
		out = renderCalendar(body, th, width, cal, &hs)
	case "timeline", "gantt":
		out = renderTimeline(body, th, width, &hs)
	case "chart":
		out = renderChart(body, th, width, &hs)
	case "form":
		out = renderForm(body, th, width, &hs)
	default:
		return nil, nil, false
	}
	return out, hs, true
}

// keyValues parses "key: value" lines (keys lowercased) and returns the
// remaining lines too.
func keyValues(body []string, keys ...string) (map[string]string, []string) {
	want := map[string]bool{}
	for _, k := range keys {
		want[k] = true
	}
	meta := map[string]string{}
	var rest []string
	for _, l := range body {
		if k, v, ok := strings.Cut(l, ":"); ok && want[strings.ToLower(strings.TrimSpace(k))] {
			meta[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
			continue
		}
		rest = append(rest, l)
	}
	return meta, rest
}

func palette(th theme.Theme) []lipgloss.Color {
	return []lipgloss.Color{th.Blue, th.Green, th.Purple, th.Yellow, th.Orange, th.Cyan, th.Red, th.Pink, th.Teal}
}

func pad(s string, w int) string {
	if d := w - ansi.StringWidth(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

func center(s string, w int) string {
	d := w - ansi.StringWidth(s)
	if d <= 0 {
		return s
	}
	return strings.Repeat(" ", d/2) + s + strings.Repeat(" ", d-d/2)
}

// ---------------------------------------------------------------------------
// Board

type boardCard struct {
	title string
	desc  []string
	line  int
}

type boardColumn struct {
	name  string
	line  int
	cards []boardCard
}

func statusColor(name string, i int, th theme.Theme) lipgloss.Color {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "done") || strings.Contains(n, "complete") || strings.Contains(n, "shipped"):
		return th.Green
	case strings.Contains(n, "progress") || strings.Contains(n, "doing") || strings.Contains(n, "review"):
		return th.Blue
	case strings.Contains(n, "not") || strings.Contains(n, "todo") || strings.Contains(n, "backlog"):
		return th.GreyFg2
	case strings.Contains(n, "block") || strings.Contains(n, "stuck"):
		return th.Red
	}
	return palette(th)[(i+2)%len(palette(th))]
}

func renderBoard(body []string, th theme.Theme, width int, hs *[]Hit) []string {
	var cols []boardColumn
	for i, l := range body {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "#"):
			cols = append(cols, boardColumn{name: strings.TrimSpace(strings.TrimLeft(t, "#")), line: i})
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* "):
			if len(cols) == 0 {
				cols = append(cols, boardColumn{name: "Cards", line: -1})
			}
			card := strings.TrimSpace(t[2:])
			card = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(card, "[ ] "), "[x] "), "[X] ")
			cols[len(cols)-1].cards = append(cols[len(cols)-1].cards, boardCard{title: card, line: i})
		case t != "" && t != "-" && len(cols) > 0 && len(cols[len(cols)-1].cards) > 0:
			// Any other text under a card is that card's description.
			c := &cols[len(cols)-1]
			last := &c.cards[len(c.cards)-1]
			last.desc = append(last.desc, t)
		}
	}
	if len(cols) == 0 {
		return []string{lipgloss.NewStyle().Foreground(th.GreyFg).Render("(empty board: add \"## Column\" headings and \"- card\" lines)")}
	}

	gap := 2
	colW := min(30, (width-gap*(len(cols)-1))/len(cols))
	stacked := colW < 16
	if stacked {
		colW = min(width, 40)
	}

	var blocks []string
	type colHits struct{ hits []Hit }
	perCol := make([]colHits, len(cols))
	for i, c := range cols {
		color := statusColor(c.name, i, th)
		pill := lipgloss.NewStyle().Background(color).Foreground(th.Bg).Bold(true).Render(" ● " + ui.Truncate(c.name, colW-8) + " ")
		count := lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf(" %d", len(c.cards)))
		lines := []string{pill + count}
		perCol[i].hits = append(perCol[i].hits, Hit{Row: 0, H: 1, X1: colW, Kind: "board:col", Line: c.line, Index: i, Arg: c.name})
		card := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(th.Line).
			Foreground(th.Fg).
			Width(colW-2).
			Padding(0, 1)
		inner := colW - 4
		for _, cd := range c.cards {
			body := lipgloss.NewStyle().Bold(true).Render(ui.Truncate(cd.title, inner*2))
			if len(cd.desc) > 0 {
				desc := strings.Split(ansi.Wrap(strings.Join(cd.desc, " "), inner, ""), "\n")
				if len(desc) > 4 {
					desc = append(desc[:3], ui.Truncate(desc[3], inner-1)+"…")
				}
				body += "\n" + lipgloss.NewStyle().Foreground(th.GreyFg2).Render(strings.Join(desc, "\n"))
			}
			rendered := strings.Split(card.Render(body), "\n")
			perCol[i].hits = append(perCol[i].hits, Hit{Row: len(lines), H: len(rendered), X1: colW, Kind: "board:card", Line: cd.line, Index: i, Arg: cd.title})
			lines = append(lines, rendered...)
		}
		perCol[i].hits = append(perCol[i].hits, Hit{Row: len(lines), H: 1, X1: colW, Kind: "board:add", Line: c.line, Index: i, Arg: c.name})
		lines = append(lines, lipgloss.NewStyle().Foreground(color).Render(" + New page"))
		for j := range lines {
			lines[j] = pad(lines[j], colW)
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	if stacked {
		row := 0
		for i, b := range blocks {
			for _, h := range perCol[i].hits {
				h.Row += row
				addHit(hs, h)
			}
			row += strings.Count(b, "\n") + 2
		}
		return strings.Split(strings.Join(blocks, "\n\n"), "\n")
	}
	for i := range blocks {
		for _, h := range perCol[i].hits {
			h.X0 += i * (colW + gap)
			h.X1 += i * (colW + gap)
			addHit(hs, h)
		}
	}
	spacer := strings.Repeat(" ", gap)
	parts := []string{}
	for i, b := range blocks {
		if i > 0 {
			parts = append(parts, spacer)
		}
		parts = append(parts, b)
	}
	return strings.Split(lipgloss.JoinHorizontal(lipgloss.Top, parts...), "\n")
}

// ---------------------------------------------------------------------------
// Calendar

type event struct {
	date  time.Time
	end   time.Time
	title string
	line  int
}

// parseEvents reads "2026-10-03: title" lines.
func parseEvents(lines []string) []event {
	var out []event
	for i, l := range lines {
		d, title, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- ")), ":")
		if !ok {
			continue
		}
		t, err := time.Parse(dateLayout, strings.TrimSpace(d))
		if err != nil {
			continue
		}
		out = append(out, event{date: t, title: strings.TrimSpace(title), line: i})
	}
	return out
}

func renderCalendar(body []string, th theme.Theme, width int, cal CalendarView, hs *[]Hit) []string {
	meta, _ := keyValues(body, "month")
	today := now()
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	if m, err := time.Parse("2006-01", meta["month"]); err == nil && !cal.Today {
		month = m
	}
	month = month.AddDate(0, cal.Shift, 0)
	events := parseEvents(body)
	byDay := map[string][]event{}
	for _, e := range events {
		k := e.date.Format(dateLayout)
		byDay[k] = append(byDay[k], e)
	}

	cw := max(min((width-8)/7, 16), 3)
	cellH := 3
	if cw < 6 {
		cellH = 1
	}
	line := lipgloss.NewStyle().Foreground(th.Line)
	muted := lipgloss.NewStyle().Foreground(th.GreyFg)
	text := lipgloss.NewStyle().Foreground(th.Fg)

	title := text.Bold(true).Render(month.Format("January 2006")) + "  " + muted.Render(fmt.Sprintf("%d events", len(events)))
	nav := lipgloss.NewStyle().Foreground(th.GreyFg2).Render(calendarNav)
	gridW := 7*cw + 8
	gap := max(gridW-ansi.StringWidth(title)-ansi.StringWidth(nav), 2)
	out := []string{title + strings.Repeat(" ", gap) + nav, ""}
	navX := ansi.StringWidth(title) + gap
	addHit(hs, Hit{Row: 0, X0: navX, X1: navX + 2, Kind: "cal:prev", Line: -1})
	addHit(hs, Hit{Row: 0, X0: navX + 3, X1: navX + 8, Kind: "cal:today", Line: -1})
	addHit(hs, Hit{Row: 0, X0: navX + 9, X1: navX + 12, Kind: "cal:next", Line: -1})

	var head strings.Builder
	head.WriteString(" ")
	for _, d := range []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"} {
		head.WriteString(muted.Render(center(d[:min(len(d), cw)], cw)) + " ")
	}
	out = append(out, head.String())

	border := func(l, m, r string) string {
		segs := make([]string, 7)
		for i := range segs {
			segs[i] = strings.Repeat("─", cw)
		}
		return line.Render(l + strings.Join(segs, m) + r)
	}
	out = append(out, border("┌", "┬", "┐"))

	start := month.AddDate(0, 0, -int(month.Weekday()))
	todayKey := today.Format(dateLayout)
	for week := 0; week < 6; week++ {
		weekStart := start.AddDate(0, 0, week*7)
		if week > 0 && weekStart.Month() != month.Month() {
			break
		}
		rows := make([]strings.Builder, cellH)
		for r := range rows {
			rows[r].WriteString(line.Render("│"))
		}
		for d := 0; d < 7; d++ {
			day := weekStart.AddDate(0, 0, d)
			key := day.Format(dateLayout)
			shade := lipgloss.NewStyle()
			if d == 0 || d == 6 {
				shade = shade.Background(th.Bg2)
			}
			num := strconv.Itoa(day.Day())
			if day.Day() == 1 && cw >= 6 {
				num = day.Format("Jan 2")
			}
			numStyle := shade.Foreground(th.Fg)
			if day.Month() != month.Month() {
				numStyle = shade.Foreground(th.GreyFg)
			}
			numStr := numStyle.Render(num + " ")
			if key == todayKey {
				numStr = lipgloss.NewStyle().Background(th.Red).Foreground(th.Bg).Bold(true).Render(" "+num+" ") + shade.Render(" ")
			}
			rows[0].WriteString(shade.Render(strings.Repeat(" ", max(cw-ansi.StringWidth(numStr), 0))) + numStr)
			evs := byDay[key]
			x0 := 1 + d*(cw+1)
			rowBase := len(out)
			if week > 0 {
				rowBase++ // the separator line is appended first
			}
			addHit(hs, Hit{Row: rowBase, H: cellH, X0: x0, X1: x0 + cw, Kind: "cal:day", Line: -1, Arg: key})
			for r := 1; r < cellH; r++ {
				cell := ""
				if i := r - 1; i < len(evs) {
					label := evs[i].title
					if r == cellH-1 && len(evs) > cellH-1 {
						label = fmt.Sprintf("+%d more", len(evs)-(cellH-2))
					} else {
						addHit(hs, Hit{Row: rowBase + r, X0: x0, X1: x0 + cw, Kind: "cal:event", Line: evs[i].line, Arg: evs[i].title})
					}
					cell = lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Fg).Render(ui.Truncate(" "+label, cw-1))
				}
				rows[r].WriteString(cell + shade.Render(strings.Repeat(" ", max(cw-ansi.StringWidth(cell), 0))))
			}
			for r := range rows {
				rows[r].WriteString(line.Render("│"))
			}
		}
		if week > 0 {
			out = append(out, border("├", "┼", "┤"))
		}
		for r := range rows {
			out = append(out, rows[r].String())
		}
	}
	out = append(out, border("└", "┴", "┘"))
	return out
}

// ---------------------------------------------------------------------------
// Timeline

func renderTimeline(body []string, th theme.Theme, width int, hs *[]Hit) []string {
	var items []event
	for i, l := range body {
		name, span, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- ")), ":")
		if !ok {
			continue
		}
		a, b, _ := strings.Cut(span, "->")
		start, err1 := time.Parse(dateLayout, strings.TrimSpace(a))
		end, err2 := time.Parse(dateLayout, strings.TrimSpace(b))
		if err1 != nil {
			continue
		}
		if err2 != nil || end.Before(start) {
			end = start
		}
		items = append(items, event{date: start, end: end, title: strings.TrimSpace(name), line: i})
	}
	muted := lipgloss.NewStyle().Foreground(th.GreyFg)
	if len(items) == 0 {
		addHit(hs, Hit{Row: 1, X1: 5, Kind: "tl:add", Line: -1})
		return []string{muted.Render("(empty timeline: add \"Task: 2026-10-01 -> 2026-10-05\" lines)"), muted.Render("+ New")}
	}

	first, last := items[0].date, items[0].end
	for _, it := range items {
		if it.date.Before(first) {
			first = it.date
		}
		if it.end.After(last) {
			last = it.end
		}
	}
	today := now()
	t0 := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	// Pad the range a little and include today when it's close by.
	first = first.AddDate(0, 0, -1)
	last = last.AddDate(0, 0, 2)
	span := int(last.Sub(first).Hours()/24) + 1
	if t0.After(first.AddDate(0, 0, -span/4)) && t0.Before(last.AddDate(0, 0, span/4)) {
		if t0.Before(first) {
			first = t0.AddDate(0, 0, -1)
		}
		if t0.After(last) {
			last = t0.AddDate(0, 0, 1)
		}
	}
	days := int(last.Sub(first).Hours()/24) + 1
	if days < 7 {
		last = first.AddDate(0, 0, 6)
		days = 7
	}

	// Every day maps onto the full width, so long ranges always fit.
	col := func(t time.Time) int {
		d := t.Sub(first).Hours() / 24
		return min(int(d*float64(width)/float64(days)), width-1)
	}
	perDay := float64(width) / float64(days)

	// Header: month names on one row, finer ticks (days, weeks or months) on
	// the next; labels that would collide are skipped.
	monthRow := []rune(strings.Repeat(" ", width))
	tickRow := []rune(strings.Repeat(" ", width))
	put := func(row []rune, at int, label string, gap int) bool {
		r := []rune(label)
		if at+len(r) > len(row) {
			at = len(row) - len(r) // slide labels at the right edge inwards
		}
		if at < 0 {
			return false
		}
		for i := max(at-gap, 0); i < min(at+len(r)+gap, len(row)); i++ {
			if row[i] != ' ' {
				return false
			}
		}
		copy(row[at:], r)
		return true
	}
	yearShown := false
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		// Label each month where it starts (or the first, if it starts
		// early enough in its month to be worth naming).
		if d.Day() == 1 || (d.Equal(first) && d.Day() <= 20) {
			label := d.Format("Jan")
			if !yearShown || d.Month() == time.January {
				label = d.Format("Jan 2006")
			}
			if put(monthRow, col(d), label, 1) {
				yearShown = true
			}
		}
	}
	todayCol := -1
	todayLabel := []rune(strconv.Itoa(t0.Day()))
	if !t0.Before(first) && !t0.After(last) {
		todayCol = min(col(t0), width-len(todayLabel))
		copy(tickRow[todayCol:], todayLabel) // reserve it so ticks avoid it
	}
	for d := first; !d.After(last); d = d.AddDate(0, 0, 1) {
		var label string
		switch {
		case perDay >= 3:
			label = strconv.Itoa(d.Day())
		case perDay*7 >= 4 && d.Weekday() == time.Monday:
			label = strconv.Itoa(d.Day())
		default:
			continue
		}
		put(tickRow, col(d), label, 1)
	}

	var tickLine strings.Builder
	if todayCol >= 0 {
		end := todayCol + len(todayLabel)
		tickLine.WriteString(muted.Render(string(tickRow[:todayCol])))
		tickLine.WriteString(lipgloss.NewStyle().Background(th.Red).Foreground(th.Bg).Bold(true).Render(string(todayLabel)))
		tickLine.WriteString(muted.Render(string(tickRow[end:])))
	} else {
		tickLine.WriteString(muted.Render(string(tickRow)))
	}
	out := []string{
		lipgloss.NewStyle().Foreground(th.Fg).Bold(true).Render(string(monthRow)),
		tickLine.String(),
		lipgloss.NewStyle().Foreground(th.Line).Render(strings.Repeat("─", width)),
	}

	for i, it := range items {
		a := col(it.date)
		b := max(col(it.end.AddDate(0, 0, 1))-1, a)
		color := palette(th)[i%len(palette(th))]
		label := []rune(" " + it.title)
		if dur := int(it.end.Sub(it.date).Hours()/24) + 1; dur > 1 {
			label = append(label, []rune(fmt.Sprintf(" · %dd", dur))...)
		}
		// Names that don't fit inside the bar go after it, or before it when
		// the bar sits against the right edge.
		inside := b - a
		outside := []rune{}
		outStart := -1
		if inside < len(label) {
			outside = []rune(strings.TrimSpace(string(label)))
			if b+2+len(outside) <= width {
				outStart = b + 2
			} else {
				outStart = max(a-1-len(outside), 0)
			}
		}
		var row strings.Builder
		for x := 0; x < width; x++ {
			if outStart >= 0 && x >= outStart && x < outStart+len(outside) && (x < a || x > b) {
				row.WriteString(lipgloss.NewStyle().Foreground(th.GreyFg2).Render(string(outside[x-outStart])))
				continue
			}
			switch {
			case x == a:
				row.WriteString(lipgloss.NewStyle().Background(th.OneBg2).Foreground(color).Render("▌"))
			case x > a && x <= b:
				ch := " "
				if k := x - a - 1; inside >= len(label) && k < len(label) {
					ch = string(label[k])
				}
				row.WriteString(lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Fg).Bold(true).Render(ch))
			case x == todayCol:
				row.WriteString(lipgloss.NewStyle().Foreground(th.Red).Render("│"))
			default:
				row.WriteString(" ")
			}
		}
		addHit(hs, Hit{Row: len(out), X1: width, Kind: "tl:item", Line: it.line, Arg: it.title})
		out = append(out, row.String())
	}
	addHit(hs, Hit{Row: len(out), X1: 5, Kind: "tl:add", Line: -1})
	out = append(out, muted.Render("+ New"))
	return out
}

// ---------------------------------------------------------------------------
// Charts (pixel art)

type datum struct {
	label string
	value float64
	line  int
}

// pixelGrid paints a w×h pixel canvas with half blocks (two pixels per row).
func pixelGrid(px [][]lipgloss.Color) []string {
	h := len(px)
	if h == 0 {
		return nil
	}
	w := len(px[0])
	var out []string
	for y := 0; y < h; y += 2 {
		var b strings.Builder
		for x := 0; x < w; x++ {
			top := px[y][x]
			var bot lipgloss.Color
			if y+1 < h {
				bot = px[y+1][x]
			}
			switch {
			case top == "" && bot == "":
				b.WriteString(" ")
			case top == bot:
				b.WriteString(lipgloss.NewStyle().Foreground(top).Render("█"))
			case top == "":
				b.WriteString(lipgloss.NewStyle().Foreground(bot).Render("▄"))
			case bot == "":
				b.WriteString(lipgloss.NewStyle().Foreground(top).Render("▀"))
			default:
				b.WriteString(lipgloss.NewStyle().Foreground(top).Background(bot).Render("▀"))
			}
		}
		out = append(out, b.String())
	}
	return out
}

func canvas(w, h int) [][]lipgloss.Color {
	px := make([][]lipgloss.Color, h)
	for i := range px {
		px[i] = make([]lipgloss.Color, w)
	}
	return px
}

func fmtNum(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e12 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', 1, 64)
}

func renderChart(body []string, th theme.Theme, width int, hs *[]Hit) []string {
	meta, _ := keyValues(body, "type", "title", "height")
	var data []datum
	for i, l := range body {
		k, v, ok := strings.Cut(strings.TrimPrefix(strings.TrimSpace(l), "- "), ":")
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "type", "title", "height":
			continue
		}
		if !ok {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "%")), 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			continue
		}
		data = append(data, datum{strings.TrimSpace(k), f, i})
	}
	muted := lipgloss.NewStyle().Foreground(th.GreyFg)
	var out []string
	if t := meta["title"]; t != "" {
		out = append(out, lipgloss.NewStyle().Foreground(th.Fg).Bold(true).Render(t), "")
	}
	if len(data) == 0 {
		addHit(hs, Hit{Row: len(out) + 1, X1: 13, Kind: "chart:add", Line: -1})
		return append(out, muted.Render("(empty chart: add \"Label: 12\" lines)"), muted.Render("+ Add value"))
	}
	height := 10
	if h, err := strconv.Atoi(meta["height"]); err == nil && h >= 3 && h <= 30 {
		height = h
	}
	var chart []string
	var local []Hit
	switch strings.ToLower(meta["type"]) {
	case "hbar", "barh", "horizontal":
		chart = hbarChart(data, th, width, &local)
	case "line":
		chart = lineChart(data, th, width, height, &local)
	case "pie", "donut":
		chart = pieChart(data, th, width, strings.ToLower(meta["type"]) == "donut", &local)
	default:
		chart = barChart(data, th, width, height, &local)
	}
	for _, h := range local {
		h.Row += len(out)
		addHit(hs, h)
	}
	out = append(out, chart...)
	addHit(hs, Hit{Row: len(out), X1: 13, Kind: "chart:add", Line: -1})
	return append(out, muted.Render("+ Add value"))
}

func maxValue(data []datum) float64 {
	m := 0.0
	for _, d := range data {
		m = math.Max(m, d.value)
	}
	if m == 0 {
		m = 1
	}
	return m
}

func barChart(data []datum, th theme.Theme, width, height int, hs *[]Hit) []string {
	maxV := maxValue(data)
	// Wide enough for every axis label (max, half and 0), e.g. "7.5".
	axisW := max(len(fmtNum(maxV)), len(fmtNum(maxV/2))) + 1
	slot := min(max((width-axisW-1)/len(data), 2), 10)
	bw := max(slot-1, 1)
	plotW := slot * len(data)
	px := canvas(plotW, height*2)
	cols := palette(th)
	for i, d := range data {
		h := int(math.Round(math.Max(d.value, 0) / maxV * float64(height*2)))
		h = max(0, min(height*2, h))
		for y := height*2 - h; y < height*2; y++ {
			for x := i * slot; x < i*slot+bw; x++ {
				if y >= 0 && y < len(px) && x >= 0 && x < len(px[y]) {
					px[y][x] = cols[i%len(cols)]
				}
			}
		}
	}
	muted := lipgloss.NewStyle().Foreground(th.GreyFg)
	axis := lipgloss.NewStyle().Foreground(th.Line)
	rows := pixelGrid(px)
	var out []string
	for i, r := range rows {
		label := ""
		switch i {
		case 0:
			label = fmtNum(maxV)
		case len(rows) - 1:
			label = "0"
		case len(rows) / 2:
			label = fmtNum(maxV / 2)
		}
		out = append(out, muted.Render(fmt.Sprintf("%*s", axisW-1, label))+axis.Render("┤")+r)
	}
	out = append(out, strings.Repeat(" ", axisW-1)+axis.Render("└"+strings.Repeat("─", plotW)))
	var labels, values strings.Builder
	for i, d := range data {
		labels.WriteString(pad(ui.Truncate(d.label, bw), slot))
		values.WriteString(lipgloss.NewStyle().Foreground(cols[i%len(cols)]).Render(pad(ui.Truncate(fmtNum(d.value), bw), slot)))
	}
	out = append(out, strings.Repeat(" ", axisW)+muted.Render(labels.String()))
	out = append(out, strings.Repeat(" ", axisW)+values.String())
	for i, d := range data {
		addHit(hs, Hit{Row: 0, H: len(out), X0: axisW + i*slot, X1: axisW + i*slot + bw, Kind: "chart:value", Line: d.line, Arg: d.label})
	}
	return out
}

func hbarChart(data []datum, th theme.Theme, width int, hs *[]Hit) []string {
	maxV := maxValue(data)
	labelW := 0
	for _, d := range data {
		labelW = max(labelW, ansi.StringWidth(d.label))
	}
	labelW = min(labelW, width/3)
	barW := max(width-labelW-len(fmtNum(maxV))-3, 4)
	cols := palette(th)
	var out []string
	for i, d := range data {
		n := math.Max(d.value, 0) / maxV * float64(barW)
		full := max(0, min(int(n), barW))
		bar := strings.Repeat("█", full)
		if frac := n - float64(full); frac > 0.5 && full < barW {
			bar += "▌"
		}
		addHit(hs, Hit{Row: len(out), X1: width, Kind: "chart:value", Line: d.line, Arg: d.label})
		out = append(out, pad(ui.Truncate(d.label, labelW), labelW)+" "+
			lipgloss.NewStyle().Foreground(cols[i%len(cols)]).Render(bar)+" "+
			lipgloss.NewStyle().Foreground(th.GreyFg2).Render(fmtNum(d.value)))
	}
	return out
}

func lineChart(data []datum, th theme.Theme, width, height int, hs *[]Hit) []string {
	minV, maxV := data[0].value, data[0].value
	for _, d := range data {
		minV, maxV = math.Min(minV, d.value), math.Max(maxV, d.value)
	}
	if maxV == minV {
		maxV = minV + 1
	}
	axisW := max(len(fmtNum(maxV)), len(fmtNum(minV))) + 1
	plotW := max(width-axisW-1, len(data))
	ph := height * 2
	px := canvas(plotW, ph)
	pt := func(i int) (int, int) {
		x := 0
		if len(data) > 1 {
			x = i * (plotW - 1) / (len(data) - 1)
		}
		y := ph - 1 - int(math.Round((data[i].value-minV)/(maxV-minV)*float64(ph-1)))
		x = max(0, min(plotW-1, x))
		y = max(0, min(ph-1, y))
		return x, y
	}
	for i := 0; i < len(data); i++ {
		x0, y0 := pt(i)
		px[y0][x0] = th.Blue
		if i+1 < len(data) {
			x1, y1 := pt(i + 1)
			// Bresenham line between neighbouring points.
			dx, dy := abs(x1-x0), -abs(y1-y0)
			sx, sy := sign(x1-x0), sign(y1-y0)
			e := dx + dy
			for x, y := x0, y0; ; {
				px[y][x] = th.Blue
				if x == x1 && y == y1 {
					break
				}
				if e2 := 2 * e; e2 >= dy {
					e += dy
					x += sx
				} else {
					e += dx
					y += sy
				}
			}
		}
		px[y0][x0] = th.Yellow
	}
	muted := lipgloss.NewStyle().Foreground(th.GreyFg)
	axis := lipgloss.NewStyle().Foreground(th.Line)
	rows := pixelGrid(px)
	var out []string
	for i, r := range rows {
		label := ""
		if i == 0 {
			label = fmtNum(maxV)
		} else if i == len(rows)-1 {
			label = fmtNum(minV)
		}
		out = append(out, muted.Render(fmt.Sprintf("%*s", axisW-1, label))+axis.Render("┤")+r)
	}
	out = append(out, strings.Repeat(" ", axisW-1)+axis.Render("└"+strings.Repeat("─", plotW)))
	labels := []rune(strings.Repeat(" ", plotW))
	for i, d := range data {
		x, _ := pt(i)
		x = max(min(x, plotW-len([]rune(d.label))), 0)
		for j, r := range []rune(d.label) {
			if x+j < len(labels) {
				labels[x+j] = r
			}
		}
	}
	out = append(out, strings.Repeat(" ", axisW)+muted.Render(string(labels)))
	// Each point owns the columns closest to it.
	for i, d := range data {
		x, _ := pt(i)
		left, right := 0, plotW
		if i > 0 {
			px, _ := pt(i - 1)
			left = (px + x + 1) / 2
		}
		if i+1 < len(data) {
			nx, _ := pt(i + 1)
			right = (x + nx + 1) / 2
		}
		addHit(hs, Hit{Row: 0, H: len(out), X0: axisW + left, X1: axisW + right, Kind: "chart:value", Line: d.line, Arg: d.label})
	}
	return out
}

func pieChart(data []datum, th theme.Theme, width int, donut bool, hs *[]Hit) []string {
	total := 0.0
	for _, d := range data {
		total += math.Max(d.value, 0)
	}
	if total == 0 {
		total = 1
	}
	r := 8
	size := r * 2
	px := canvas(size, size)
	cols := palette(th)
	// Cumulative angles, starting at 12 o'clock and going clockwise.
	bounds := make([]float64, len(data))
	acc := 0.0
	for i, d := range data {
		acc += math.Max(d.value, 0) / total
		bounds[i] = acc
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			fx, fy := float64(x)-float64(r)+0.5, float64(y)-float64(r)+0.5
			dist := math.Hypot(fx, fy)
			if dist > float64(r) || (donut && dist < float64(r)*0.5) {
				continue
			}
			ang := math.Atan2(fx, -fy) / (2 * math.Pi)
			if ang < 0 {
				ang++
			}
			idx := sort.SearchFloat64s(bounds, ang)
			px[y][x] = cols[min(idx, len(data)-1)%len(cols)]
		}
	}
	pie := pixelGrid(px)
	var legend []string
	for i, d := range data {
		pct := math.Max(d.value, 0) / total * 100
		legend = append(legend, lipgloss.NewStyle().Foreground(cols[i%len(cols)]).Render("■ ")+
			lipgloss.NewStyle().Foreground(th.Fg).Render(ui.Truncate(d.label, max(width-size-16, 6)))+
			lipgloss.NewStyle().Foreground(th.GreyFg2).Render(fmt.Sprintf("  %s (%.0f%%)", fmtNum(d.value), pct)))
	}
	for i, d := range data {
		addHit(hs, Hit{Row: i, X0: size + 3, X1: width, Kind: "chart:value", Line: d.line, Arg: d.label})
	}
	out := make([]string, max(len(pie), len(legend)))
	for i := range out {
		left := strings.Repeat(" ", size)
		if i < len(pie) {
			left = pie[i]
		}
		right := ""
		if i < len(legend) {
			right = legend[i]
		}
		out[i] = left + "   " + right
	}
	return out
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func sign(x int) int {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return 0
}

// ---------------------------------------------------------------------------
// Form

// formQuestion is one "? ..." line plus its "= answer" line, if any.
type formQuestion struct {
	text, kind string
	required   bool
	options    []string
	line       int
	answer     string
}

func parseForm(body []string) []formQuestion {
	var qs []formQuestion
	for i, l := range body {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "?"):
			q := formQuestion{line: i, kind: "text", required: strings.HasPrefix(t, "?*")}
			t = strings.TrimSpace(strings.TrimLeft(t, "?*"))
			text, opts, _ := strings.Cut(t, ":")
			if j := strings.LastIndex(text, "("); j > 0 && strings.HasSuffix(strings.TrimSpace(text), ")") {
				q.kind = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(text[j+1:]), ")"))
				text = strings.TrimSpace(text[:j])
			}
			q.text = strings.TrimSpace(text)
			for _, o := range strings.Split(opts, "|") {
				if o = strings.TrimSpace(o); o != "" {
					q.options = append(q.options, o)
				}
			}
			qs = append(qs, q)
		case strings.HasPrefix(t, "=") && len(qs) > 0:
			qs[len(qs)-1].answer = strings.TrimSpace(t[1:])
		}
	}
	return qs
}

func renderForm(body []string, th theme.Theme, width int, hs *[]Hit) []string {
	meta, _ := keyValues(body, "title", "description")
	w := min(width, 64)
	text := lipgloss.NewStyle().Foreground(th.Fg)
	muted := lipgloss.NewStyle().Foreground(th.GreyFg2)
	faint := lipgloss.NewStyle().Foreground(th.GreyFg)
	accent := lipgloss.NewStyle().Foreground(th.Blue)

	lineOf := func(key string) int {
		for i, l := range body {
			if k, _, ok := strings.Cut(l, ":"); ok && strings.EqualFold(strings.TrimSpace(k), key) {
				return i
			}
		}
		return -1
	}
	title := meta["title"]
	if title == "" {
		title = "Form title"
	}
	addHit(hs, Hit{Row: 0, X1: w, Kind: "form:title", Line: lineOf("title"), Arg: meta["title"]})
	out := []string{text.Bold(true).Render(strings.ToUpper(title[:1]) + title[1:])}
	desc := meta["description"]
	addHit(hs, Hit{Row: 1, X1: w, Kind: "form:desc", Line: lineOf("description"), Arg: desc})
	if desc == "" {
		out = append(out, faint.Render("Description (optional)"))
	} else {
		out = append(out, muted.Render(desc))
	}
	out = append(out, "")

	card := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(th.Line).
		Width(w-2).
		Padding(0, 1)
	input := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(th.OneBg3).
		Width(w - 6)

	qs := parseForm(body)
	for _, q := range qs {
		var lines []string
		var local []Hit
		mark := func(kind string, h, idx int) {
			local = append(local, Hit{Row: len(lines), H: h, X1: w, Kind: kind, Line: q.line, Index: idx, Arg: q.text})
		}
		head := text.Bold(true).Render(q.text)
		if q.required {
			head += lipgloss.NewStyle().Foreground(th.Red).Render(" *")
		}
		mark("form:question", 1, 0)
		lines = append(lines, head)

		answerBox := func(placeholder string, height int) {
			body := faint.Render(" " + placeholder)
			if q.answer != "" {
				body = text.Render(" " + q.answer)
			}
			st := input
			if height > 1 {
				st = st.Height(height)
			}
			box := strings.Split(st.Render(body), "\n")
			mark("form:text", len(box), 0)
			lines = append(lines, box...)
		}
		chosen := map[string]bool{}
		for _, a := range strings.Split(q.answer, ",") {
			chosen[strings.TrimSpace(a)] = true
		}
		switch q.kind {
		case "choice", "select", "radio":
			lines = append(lines, faint.Render("(Respondents can select up to 1)"))
			for i, o := range q.options {
				mark("form:option", 1, i)
				if chosen[o] {
					lines = append(lines, accent.Render("● ")+text.Bold(true).Render(o))
				} else {
					lines = append(lines, muted.Render("○ ")+text.Render(o))
				}
			}
			mark("form:addopt", 1, 0)
			lines = append(lines, faint.Render("+ Add option"))
		case "multi", "checkbox", "checkboxes":
			lines = append(lines, faint.Render(fmt.Sprintf("(Respondents can select up to %d)", len(q.options))))
			for i, o := range q.options {
				mark("form:multi", 1, i)
				if chosen[o] {
					lines = append(lines, accent.Render("☑ ")+text.Bold(true).Render(o))
				} else {
					lines = append(lines, muted.Render("☐ ")+text.Render(o))
				}
			}
			mark("form:addopt", 1, 0)
			lines = append(lines, faint.Render("+ Add option"))
		case "rating":
			n, _ := strconv.Atoi(q.answer)
			var stars []string
			for i := 1; i <= 5; i++ {
				if i <= n {
					stars = append(stars, "★")
				} else {
					stars = append(stars, "☆")
				}
				local = append(local, Hit{Row: len(lines), H: 1, X0: 2 + (i-1)*2, X1: 4 + (i-1)*2, Kind: "form:rating", Line: q.line, Index: i, Arg: q.text})
			}
			lines = append(lines, lipgloss.NewStyle().Foreground(th.Yellow).Render(strings.Join(stars, " ")))
		case "date":
			answerBox("󰃭 Pick a date (YYYY-MM-DD)", 1)
		case "long", "paragraph":
			answerBox("Long answer", 3)
		case "email":
			answerBox("name@example.com", 1)
		default:
			answerBox("Respondent's answer", 1)
		}
		cardStart := len(out)
		for _, h := range local {
			h.Row += cardStart + 1 // top border
			addHit(hs, h)
		}
		out = append(out, strings.Split(card.Render(strings.Join(lines, "\n")), "\n")...)
		out = append(out, "")
	}
	if len(qs) == 0 {
		out = append(out, faint.Render("(add questions: \"? Question\", \"? Pick one (choice): A | B\")"))
	}
	addHit(hs, Hit{Row: len(out), X1: w, Kind: "form:addq", Line: -1})
	out = append(out, center(accent.Render("⊕ add question"), w))
	return out
}
