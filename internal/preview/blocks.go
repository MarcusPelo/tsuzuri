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

// renderBlock draws the special fenced blocks (board, calendar, timeline,
// chart, form). ok is false for ordinary code.
func renderBlock(lang string, body []string, th theme.Theme, width int) ([]string, bool) {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "board", "kanban":
		return renderBoard(body, th, width), true
	case "calendar":
		return renderCalendar(body, th, width), true
	case "timeline", "gantt":
		return renderTimeline(body, th, width), true
	case "chart":
		return renderChart(body, th, width), true
	case "form":
		return renderForm(body, th, width), true
	}
	return nil, false
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

type boardColumn struct {
	name  string
	cards []string
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

func renderBoard(body []string, th theme.Theme, width int) []string {
	var cols []boardColumn
	for _, l := range body {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "#"):
			cols = append(cols, boardColumn{name: strings.TrimSpace(strings.TrimLeft(t, "#"))})
		case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* "):
			if len(cols) == 0 {
				cols = append(cols, boardColumn{name: "Cards"})
			}
			card := strings.TrimSpace(t[2:])
			card = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(card, "[ ] "), "[x] "), "[X] ")
			cols[len(cols)-1].cards = append(cols[len(cols)-1].cards, card)
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
	for i, c := range cols {
		color := statusColor(c.name, i, th)
		pill := lipgloss.NewStyle().Background(color).Foreground(th.Bg).Bold(true).Render(" ● " + ui.Truncate(c.name, colW-8) + " ")
		count := lipgloss.NewStyle().Foreground(color).Render(fmt.Sprintf(" %d", len(c.cards)))
		lines := []string{pill + count}
		card := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(th.Line).
			Foreground(th.Fg).
			Width(colW-2).
			Padding(0, 1)
		for _, txt := range c.cards {
			lines = append(lines, strings.Split(card.Render(ui.Truncate(txt, (colW-4)*2)), "\n")...)
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(color).Render(" + New page"))
		for j := range lines {
			lines[j] = pad(lines[j], colW)
		}
		blocks = append(blocks, strings.Join(lines, "\n"))
	}
	if stacked {
		return strings.Split(strings.Join(blocks, "\n\n"), "\n")
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
}

// parseEvents reads "2026-10-03: title" lines.
func parseEvents(lines []string) []event {
	var out []event
	for _, l := range lines {
		d, title, ok := strings.Cut(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- ")), ":")
		if !ok {
			continue
		}
		t, err := time.Parse(dateLayout, strings.TrimSpace(d))
		if err != nil {
			continue
		}
		out = append(out, event{date: t, title: strings.TrimSpace(title)})
	}
	return out
}

func renderCalendar(body []string, th theme.Theme, width int) []string {
	meta, rest := keyValues(body, "month")
	today := now()
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	if m, err := time.Parse("2006-01", meta["month"]); err == nil {
		month = m
	}
	events := parseEvents(rest)
	byDay := map[string][]string{}
	for _, e := range events {
		k := e.date.Format(dateLayout)
		byDay[k] = append(byDay[k], e.title)
	}

	cw := max(min((width-8)/7, 16), 3)
	cellH := 3
	if cw < 6 {
		cellH = 1
	}
	line := lipgloss.NewStyle().Foreground(th.Line)
	muted := lipgloss.NewStyle().Foreground(th.GreyFg)
	text := lipgloss.NewStyle().Foreground(th.Fg)

	title := text.Bold(true).Render(month.Format("January 2006"))
	out := []string{title + "  " + muted.Render(fmt.Sprintf("%d events", len(events))), ""}

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
			for r := 1; r < cellH; r++ {
				cell := ""
				if i := r - 1; i < len(evs) {
					label := evs[i]
					if r == cellH-1 && len(evs) > cellH-1 {
						label = fmt.Sprintf("+%d more", len(evs)-(cellH-2))
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

func renderTimeline(body []string, th theme.Theme, width int) []string {
	var items []event
	for _, l := range body {
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
		items = append(items, event{date: start, end: end, title: strings.TrimSpace(name)})
	}
	muted := lipgloss.NewStyle().Foreground(th.GreyFg)
	if len(items) == 0 {
		return []string{muted.Render("(empty timeline: add \"Task: 2026-10-01 -> 2026-10-05\" lines)")}
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
	first = first.AddDate(0, 0, -2)
	cell := 4
	days := max(int(last.Sub(first).Hours()/24)+4, 7)
	for cell > 2 && days*cell > width {
		cell--
	}
	days = min(days, width/cell)

	today := now()
	t0 := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	dayIdx := func(t time.Time) int { return int(t.Sub(first).Hours() / 24) }
	todayIdx := dayIdx(t0)

	var months, nums strings.Builder
	lastMonth := time.Month(0)
	for d := 0; d < days; d++ {
		day := first.AddDate(0, 0, d)
		if day.Month() != lastMonth {
			label := day.Format("Jan 2006")
			if lastMonth != 0 {
				label = day.Format("January")
			}
			lastMonth = day.Month()
			if ansi.StringWidth(months.String()) <= d*cell {
				months.WriteString(strings.Repeat(" ", d*cell-ansi.StringWidth(ansi.Strip(months.String()))))
				months.WriteString(lipgloss.NewStyle().Foreground(th.Fg).Bold(true).Render(label))
			}
		}
		n := center(strconv.Itoa(day.Day()), cell)
		if d == todayIdx {
			nums.WriteString(lipgloss.NewStyle().Background(th.Red).Foreground(th.Bg).Bold(true).Render(n))
		} else {
			nums.WriteString(muted.Render(n))
		}
	}
	out := []string{ui.Truncate(months.String(), width), nums.String(), lipgloss.NewStyle().Foreground(th.Line).Render(strings.Repeat("─", days*cell))}

	for i, it := range items {
		a, b := dayIdx(it.date), dayIdx(it.end)
		color := palette(th)[i%len(palette(th))]
		var row strings.Builder
		for d := 0; d < days; d++ {
			inBar := d >= a && d <= b
			weekend := first.AddDate(0, 0, d).Weekday()%6 == 0
			bg := lipgloss.NewStyle()
			if weekend {
				bg = bg.Background(th.Bg2)
			}
			switch {
			case inBar:
				label := []rune(" " + it.title + strings.Repeat(" ", (b-a+1)*cell))
				off := (d - a) * cell
				seg := string(label[min(off, len(label)):min(off+cell, len(label))])
				st := lipgloss.NewStyle().Background(th.OneBg2).Foreground(th.Fg).Bold(true)
				if d == a {
					seg = string([]rune(seg)[1:])
					row.WriteString(lipgloss.NewStyle().Background(th.OneBg2).Foreground(color).Render("▌") + st.Render(seg))
				} else {
					row.WriteString(st.Render(seg))
				}
			case d == todayIdx:
				row.WriteString(bg.Render(strings.Repeat(" ", cell/2)) + bg.Foreground(th.Red).Render("│") + bg.Render(strings.Repeat(" ", cell-cell/2-1)))
			default:
				row.WriteString(bg.Render(strings.Repeat(" ", cell)))
			}
		}
		out = append(out, row.String())
	}
	out = append(out, muted.Render("+ New"))
	return out
}

// ---------------------------------------------------------------------------
// Charts (pixel art)

type datum struct {
	label string
	value float64
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

func renderChart(body []string, th theme.Theme, width int) []string {
	meta, rest := keyValues(body, "type", "title", "height")
	var data []datum
	for _, l := range rest {
		k, v, ok := strings.Cut(strings.TrimPrefix(strings.TrimSpace(l), "- "), ":")
		if !ok {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(v), "%")), 64)
		if err != nil {
			continue
		}
		data = append(data, datum{strings.TrimSpace(k), f})
	}
	muted := lipgloss.NewStyle().Foreground(th.GreyFg)
	var out []string
	if t := meta["title"]; t != "" {
		out = append(out, lipgloss.NewStyle().Foreground(th.Fg).Bold(true).Render(t), "")
	}
	if len(data) == 0 {
		return append(out, muted.Render("(empty chart: add \"Label: 12\" lines)"))
	}
	height := 10
	if h, err := strconv.Atoi(meta["height"]); err == nil && h >= 3 && h <= 30 {
		height = h
	}
	switch strings.ToLower(meta["type"]) {
	case "hbar", "barh", "horizontal":
		return append(out, hbarChart(data, th, width)...)
	case "line":
		return append(out, lineChart(data, th, width, height)...)
	case "pie", "donut":
		return append(out, pieChart(data, th, width, strings.ToLower(meta["type"]) == "donut")...)
	}
	return append(out, barChart(data, th, width, height)...)
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

func barChart(data []datum, th theme.Theme, width, height int) []string {
	maxV := maxValue(data)
	axisW := len(fmtNum(maxV)) + 1
	slot := min(max((width-axisW-1)/len(data), 2), 10)
	bw := max(slot-1, 1)
	plotW := slot * len(data)
	px := canvas(plotW, height*2)
	cols := palette(th)
	for i, d := range data {
		h := int(math.Round(d.value / maxV * float64(height*2)))
		for y := height*2 - h; y < height*2; y++ {
			for x := i * slot; x < i*slot+bw; x++ {
				px[y][x] = cols[i%len(cols)]
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
	return out
}

func hbarChart(data []datum, th theme.Theme, width int) []string {
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
		n := d.value / maxV * float64(barW)
		full := int(n)
		bar := strings.Repeat("█", full)
		if frac := n - float64(full); frac > 0.5 {
			bar += "▌"
		}
		out = append(out, pad(ui.Truncate(d.label, labelW), labelW)+" "+
			lipgloss.NewStyle().Foreground(cols[i%len(cols)]).Render(bar)+" "+
			lipgloss.NewStyle().Foreground(th.GreyFg2).Render(fmtNum(d.value)))
	}
	return out
}

func lineChart(data []datum, th theme.Theme, width, height int) []string {
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
	return out
}

func pieChart(data []datum, th theme.Theme, width int, donut bool) []string {
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

func renderForm(body []string, th theme.Theme, width int) []string {
	meta, rest := keyValues(body, "title", "description")
	w := min(width, 64)
	text := lipgloss.NewStyle().Foreground(th.Fg)
	muted := lipgloss.NewStyle().Foreground(th.GreyFg2)
	faint := lipgloss.NewStyle().Foreground(th.GreyFg)

	title := meta["title"]
	if title == "" {
		title = "Form title"
	}
	out := []string{text.Bold(true).Render(strings.ToUpper(title[:1]) + title[1:])}
	if d := meta["description"]; d != "" {
		out = append(out, muted.Render(d))
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
		Foreground(th.GreyFg).
		Width(w - 6)

	n := 0
	for _, l := range rest {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "?") {
			continue
		}
		n++
		required := strings.HasPrefix(t, "?*")
		t = strings.TrimSpace(strings.TrimLeft(t, "?*"))
		q, opts, _ := strings.Cut(t, ":")
		kind := "text"
		if i := strings.LastIndex(q, "("); i > 0 && strings.HasSuffix(strings.TrimSpace(q), ")") {
			kind = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(q[i+1:]), ")"))
			q = strings.TrimSpace(q[:i])
		}
		head := text.Bold(true).Render(q)
		if required {
			head += lipgloss.NewStyle().Foreground(th.Red).Render(" *")
		}
		lines := []string{head}
		var options []string
		for _, o := range strings.Split(opts, "|") {
			if o = strings.TrimSpace(o); o != "" {
				options = append(options, o)
			}
		}
		switch kind {
		case "choice", "select", "radio":
			lines = append(lines, faint.Render("(Respondents can select up to 1)"))
			for _, o := range options {
				lines = append(lines, muted.Render("○ ")+text.Render(o))
			}
			lines = append(lines, faint.Render("+ Add option"))
		case "multi", "checkbox", "checkboxes":
			lines = append(lines, faint.Render(fmt.Sprintf("(Respondents can select up to %d)", len(options))))
			for _, o := range options {
				lines = append(lines, muted.Render("☐ ")+text.Render(o))
			}
			lines = append(lines, faint.Render("+ Add option"))
		case "rating":
			lines = append(lines, lipgloss.NewStyle().Foreground(th.Yellow).Render("☆ ☆ ☆ ☆ ☆"))
		case "date":
			lines = append(lines, input.Render(" 󰃭 Pick a date"))
		case "long", "paragraph":
			lines = append(lines, input.Height(3).Render(" Long answer"))
		case "email":
			lines = append(lines, input.Render(" name@example.com"))
		default:
			lines = append(lines, input.Render(" Respondent's answer"))
		}
		out = append(out, strings.Split(card.Render(strings.Join(lines, "\n")), "\n")...)
		out = append(out, "")
	}
	if n == 0 {
		out = append(out, faint.Render("(add questions: \"? Question\", \"? Pick one (choice): A | B\")"))
	}
	out = append(out, center(lipgloss.NewStyle().Foreground(th.Blue).Render("⊕ add question"), w))
	return out
}
