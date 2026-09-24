package app

const (
	// SidebarWidth defines character width of the sidebar pane when open.
	SidebarWidth = 26
	// MinTermWidth is the minimum required terminal width.
	MinTermWidth = 50
	// MinTermHeight is the minimum required terminal height.
	MinTermHeight = 10
)

// CalculateLayout splits window bounds into header, sidebar, editor, and preview dimensions.
// If sidebarOpen is false, the sidebar collapses and editor/preview take the full window.
func CalculateLayout(totalWidth, totalHeight, headerHeight, footerHeight int, sidebarOpen bool) (sidebarW, editorW, previewW, bodyH int) {
	if totalWidth < MinTermWidth || totalHeight < MinTermHeight {
		return 0, 0, 0, 0
	}

	bodyH = totalHeight - headerHeight - footerHeight
	if bodyH < 3 {
		bodyH = 3
	}

	if !sidebarOpen {
		sidebarW = 0
		editorW = totalWidth / 2
		previewW = totalWidth - editorW
		return sidebarW, editorW, previewW, bodyH
	}

	sidebarW = SidebarWidth
	remainW := totalWidth - sidebarW
	if remainW < 20 {
		return sidebarW, remainW, 0, bodyH
	}

	editorW = remainW / 2
	previewW = remainW - editorW
	return sidebarW, editorW, previewW, bodyH
}
