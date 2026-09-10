package app

const (
	// SidebarWidth defines fixed character width of the sidebar pane.
	SidebarWidth = 28
	// MinTermWidth is the minimum required terminal width.
	MinTermWidth = 40
	// MinTermHeight is the minimum required terminal height.
	MinTermHeight = 10
)

// CalculateLayout splits window bounds into header, sidebar, and content dimensions,
// returning zeros if terminal dimensions are below minimum thresholds.
func CalculateLayout(totalWidth, totalHeight, headerHeight int) (sidebarW, contentW, contentH int) {
	if totalWidth < MinTermWidth || totalHeight < MinTermHeight {
		return 0, 0, 0
	}

	sidebarW = SidebarWidth
	contentW = totalWidth - sidebarW
	if contentW < 10 {
		contentW = 10
	}
	contentH = totalHeight - headerHeight
	if contentH < 3 {
		contentH = 3
	}
	return sidebarW, contentW, contentH
}
