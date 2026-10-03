package app

const (
	// SidebarWidth is the explorer width when open (nvim-tree's default).
	SidebarWidth = 30
	// MinTermWidth is the minimum required terminal width.
	MinTermWidth = 60
	// MinTermHeight is the minimum required terminal height.
	MinTermHeight = 12
	// minPreviewTotal is the narrowest terminal that still shows the preview.
	minPreviewTotal = 90

	tablineHeight = 1
	footerHeight  = 2 // statusline + command line
)

// Layout holds the screen geometry of every pane. X offsets are absolute
// columns; all body panes start at row BodyY and are BodyH rows tall.
type Layout struct {
	SidebarW int
	EditorX  int
	EditorW  int
	PreviewX int
	PreviewW int
	BodyY    int
	BodyH    int
}

// CalculateLayout splits the window into explorer, editor and preview. A
// one-column divider separates neighbouring panes.
func CalculateLayout(totalWidth, totalHeight int, sidebarOpen, previewOpen bool) Layout {
	l := Layout{BodyY: tablineHeight, BodyH: max(totalHeight-tablineHeight-footerHeight, 1)}
	if totalWidth < MinTermWidth || totalHeight < MinTermHeight {
		return l
	}

	x := 0
	if sidebarOpen {
		l.SidebarW = SidebarWidth
		x = SidebarWidth + 1
	}
	remain := totalWidth - x

	if previewOpen && totalWidth >= minPreviewTotal {
		l.EditorW = (remain - 1) / 2
		l.PreviewW = remain - 1 - l.EditorW
		l.EditorX = x
		l.PreviewX = x + l.EditorW + 1
		return l
	}
	l.EditorX = x
	l.EditorW = remain
	return l
}
