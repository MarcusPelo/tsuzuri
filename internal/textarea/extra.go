package textarea

// tsuzuri: helpers that are not part of the upstream bubbles textarea API.

// YOffset returns the first visible visual (soft-wrapped) line.
func (m Model) YOffset() int {
	return m.viewport.YOffset
}

// EnsureVisible scrolls so the cursor is on screen. Unlike the upstream
// repositionView it does not clamp against the content rendered by the last
// View call, so it also works right after SetValue on a long document.
func (m *Model) EnsureVisible() {
	off := m.viewport.YOffset
	h := max(m.viewport.Height, 1)
	row := m.cursorLineNumber()
	if row < off {
		off = row
	} else if row > off+h-1 {
		off = row - h + 1
	}
	m.viewport.YOffset = max(off, 0)
}

// visualLineCount returns the number of soft-wrapped lines in the buffer.
func (m Model) visualLineCount() int {
	n := 0
	for _, l := range m.value {
		n += len(m.memoizedWrap(l, m.width))
	}
	return n
}

// CursorPosition returns the 1-based logical line and column of the cursor.
func (m Model) CursorPosition() (line, col int) {
	return m.row + 1, m.col + 1
}

// moveToVisual places the cursor on visual line v, at display column col
// within that wrapped segment (clamped to the segment's length).
func (m *Model) moveToVisual(v, col int) {
	if v < 0 {
		v = 0
	}
	acc := 0
	for r, l := range m.value {
		segs := m.memoizedWrap(l, m.width)
		if v < acc+len(segs) {
			start := 0
			for i := 0; i < v-acc; i++ {
				start += len(segs[i])
			}
			seg := segs[v-acc]
			segLen := len(seg)
			// A wrapped segment (not the last) ends where the next begins;
			// keep the cursor on this segment rather than jumping down.
			if v-acc < len(segs)-1 && segLen > 0 {
				segLen--
			}
			m.row = r
			m.col = start + clamp(col, 0, segLen)
			return
		}
		acc += len(segs)
	}
	// Past the end: last character of the buffer.
	m.row = len(m.value) - 1
	m.col = len(m.value[m.row])
}

// ScrollBy scrolls the view by n visual lines (negative scrolls up) without
// requiring focus, dragging the cursor along only when it would leave the
// screen — the same feel as scrolling a Vim window with the mouse.
func (m *Model) ScrollBy(n int) {
	total := m.visualLineCount()
	maxOff := max(0, total-1)
	off := clamp(m.viewport.YOffset+n, 0, maxOff)
	m.viewport.YOffset = off

	top := off
	bottom := off + m.viewport.Height - 1
	cur := m.cursorLineNumber()
	col := m.LineInfo().ColumnOffset
	switch {
	case cur < top:
		m.moveToVisual(top, col)
	case cur > bottom:
		m.moveToVisual(bottom, col)
	}
}

// ClickAt moves the cursor to the cell clicked at (x, y), relative to the
// textarea's top-left corner, accounting for the prompt and line-number gutter.
func (m *Model) ClickAt(x, y int) {
	gutter := m.promptWidth
	if m.ShowLineNumbers {
		gutter += 6
	}
	m.moveToVisual(m.viewport.YOffset+y, x-gutter)
	m.EnsureVisible()
}

// MoveCursorBy moves the cursor by n logical lines and keeps it in view.
func (m *Model) MoveCursorBy(n int) {
	for ; n > 0; n-- {
		m.CursorDown()
	}
	for ; n < 0; n++ {
		m.CursorUp()
	}
	m.EnsureVisible()
}

// GotoLine moves the cursor to the start of 1-based line n.
func (m *Model) GotoLine(n int) {
	m.row = clamp(n-1, 0, len(m.value)-1)
	m.SetCursor(0)
	m.EnsureVisible()
}

// CharLeft / CharRight move the cursor one character within the line.
func (m *Model) CharLeft() {
	if m.col > 0 {
		m.SetCursor(m.col - 1)
	}
}

// CharRight moves the cursor one character right, stopping at line end.
func (m *Model) CharRight() {
	if m.col < len(m.value[m.row]) {
		m.SetCursor(m.col + 1)
	}
}

// WordForward moves the cursor to the start of the next word.
func (m *Model) WordForward() {
	m.wordRight()
	m.EnsureVisible()
}

// WordBackward moves the cursor to the start of the previous word.
func (m *Model) WordBackward() {
	m.wordLeft()
	m.EnsureVisible()
}

// GotoTop moves the cursor to the first line.
func (m *Model) GotoTop() {
	m.moveToBegin()
	m.EnsureVisible()
}

// GotoBottom moves the cursor to the start of the last line.
func (m *Model) GotoBottom() {
	m.row = len(m.value) - 1
	m.SetCursor(0)
	m.EnsureVisible()
}

// DeleteCharForward deletes the character under the cursor (Vim's x).
func (m *Model) DeleteCharForward() {
	line := m.value[m.row]
	if m.col < len(line) {
		m.value[m.row] = append(line[:m.col:m.col], line[m.col+1:]...)
		if m.col > 0 && m.col >= len(m.value[m.row]) {
			m.col = len(m.value[m.row]) - 1
		}
	}
}

// DeleteLine removes the cursor's line (Vim's dd).
func (m *Model) DeleteLine() {
	if len(m.value) == 1 {
		m.value[0] = m.value[0][:0]
		m.col = 0
		return
	}
	m.value = append(m.value[:m.row:m.row], m.value[m.row+1:]...)
	if m.row >= len(m.value) {
		m.row = len(m.value) - 1
	}
	m.SetCursor(0)
	m.EnsureVisible()
}

// OpenLineBelow inserts an empty line below the cursor and moves onto it (Vim's o).
func (m *Model) OpenLineBelow() {
	m.CursorEnd()
	m.splitLine(m.row, m.col)
	m.EnsureVisible()
}

// OpenLineAbove inserts an empty line above the cursor and moves onto it (Vim's O).
func (m *Model) OpenLineAbove() {
	m.CursorStart()
	m.splitLine(m.row, 0)
	m.row--
	m.col = 0
	m.EnsureVisible()
}

// CursorScreen returns the cursor cell relative to the textarea's top-left
// corner (including the line-number gutter), or ok=false when off screen.
func (m Model) CursorScreen() (x, y int, ok bool) {
	gutter := m.promptWidth
	if m.ShowLineNumbers {
		gutter += 6
	}
	y = m.cursorLineNumber() - m.viewport.YOffset
	if y < 0 || y >= m.viewport.Height {
		return 0, 0, false
	}
	return gutter + m.LineInfo().CharOffset, y, true
}

// RowCol returns the 0-based logical row and rune column of the cursor.
func (m Model) RowCol() (int, int) { return m.row, m.col }

// SetRowCol moves the cursor to a 0-based logical row and rune column.
func (m *Model) SetRowCol(row, col int) {
	m.row = clamp(row, 0, len(m.value)-1)
	m.col = clamp(col, 0, len(m.value[m.row]))
	m.lastCharOffset = 0
	m.EnsureVisible()
}

// LineBeforeCursor returns the text of the cursor's line up to the cursor.
func (m Model) LineBeforeCursor() string {
	return string(m.value[m.row][:clamp(m.col, 0, len(m.value[m.row]))])
}

// DeleteBefore removes n runes before the cursor on the current line.
func (m *Model) DeleteBefore(n int) {
	line := m.value[m.row]
	n = clamp(n, 0, m.col)
	m.value[m.row] = append(line[:m.col-n:m.col-n], line[m.col:]...)
	m.col -= n
}

// IndentLine prepends n spaces to the cursor's line, keeping the cursor on
// the same character.
func (m *Model) IndentLine(n int) {
	pad := make([]rune, n)
	for i := range pad {
		pad[i] = ' '
	}
	m.value[m.row] = append(pad, m.value[m.row]...)
	m.col += n
}

// OutdentLine removes up to n leading spaces from the cursor's line.
func (m *Model) OutdentLine(n int) {
	line := m.value[m.row]
	k := 0
	for k < n && k < len(line) && line[k] == ' ' {
		k++
	}
	m.value[m.row] = append([]rune(nil), line[k:]...)
	m.col = max(m.col-k, 0)
}

// CurrentLine returns the cursor's whole line.
func (m Model) CurrentLine() string { return string(m.value[m.row]) }
