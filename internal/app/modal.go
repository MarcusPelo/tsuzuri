package app

import (
	"strings"

	"github.com/jaisuriya-11/tsuzuri/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

// RenderKeymapModal renders the floating shortcuts cheatsheet modal in the center of the terminal.
func RenderKeymapModal(th theme.Theme, termWidth, termHeight int) string {
	modalWidth := 74
	if modalWidth > termWidth-4 {
		modalWidth = termWidth - 4
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.DarkFg).
		Background(th.NormalBg).
		Padding(0, 2)

	catHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.SidebarBg)

	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.SelectedFg)

	descStyle := lipgloss.NewStyle().
		Foreground(th.TitleFg)

	var b strings.Builder

	// Header
	b.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Width(modalWidth-4).Render(titleStyle.Render("  TSUZURI SHORTCUTS CHEATSHEET")) + "\n\n")

	// Section 1: Navigation & Layout
	b.WriteString(catHeaderStyle.Render("󰕭 NAVIGATION & LAYOUT") + "\n")
	b.WriteString(fmtShortcut("Tab", "Cycle Focus (Sidebar → Editor → Preview)", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("Ctrl+B", "Toggle Sidebar (Show / Hide)", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("Ctrl+D", "Return to NvChad Landing Page", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("Ctrl+N", "New page (unsaved until :w)", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("[ / ]", "Previous / next buffer tab", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("x", "Close active tab (editor NORMAL mode)", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("SPC h / ?", "Open this Shortcuts Modal", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("Ctrl+C", "Quit Application", keyStyle, descStyle) + "\n\n")

	// Section 2: Notion Sidebar & Nested Tree
	b.WriteString(catHeaderStyle.Render("󰉋 NOTION SIDEBAR & TREE") + "\n")
	b.WriteString(fmtShortcut("j / k", "Navigate items up / down", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("Enter / l", "Open page / expand folder", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("o", "Open page that has sub-pages", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("h", "Collapse folder", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("z / Space", "Toggle expand / collapse", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("r / e", "Rename selected document", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("n", "New top-level document", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("a", "New nested sub-page under current doc", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("d / x", "Delete document", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("/", "Search & filter documents in tree", keyStyle, descStyle) + "\n\n")

	// Section 3: Markdown Editor
	b.WriteString(catHeaderStyle.Render(" MARKDOWN EDITOR (VIM)") + "\n")
	b.WriteString(fmtShortcut("i / a", "Enter INSERT mode (compiles live on typing)", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("Esc", "Return to NORMAL mode", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut(":w", "Save document", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut(":wq / :x", "Save and quit application", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut(":q / :q!", "Quit (discard unsaved changes)", keyStyle, descStyle) + "\n\n")

	// Section 4: Live Compiled Preview
	b.WriteString(catHeaderStyle.Render("󰈈 LIVE COMPILED PREVIEW") + "\n")
	b.WriteString(fmtShortcut("j / k", "Scroll preview line up / down", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("d / u", "Half-page down / up (also Ctrl+D / Ctrl+U)", keyStyle, descStyle) + "\n")
	b.WriteString(fmtShortcut("g / G", "Jump to top / bottom", keyStyle, descStyle) + "\n\n")

	// Footer dismissal hint
	dismissStyle := lipgloss.NewStyle().
		Italic(true).
		Foreground(th.MutedFg).
		Align(lipgloss.Center).
		Width(modalWidth - 4)
	b.WriteString(dismissStyle.Render("Esc / q / Space / Enter to dismiss"))

	box := lipgloss.NewStyle().
		Width(modalWidth).
		BorderStyle(lipgloss.DoubleBorder()).
		BorderForeground(th.SelectedFg).
		Background(th.SelectedBg).
		Padding(1, 2).
		Render(b.String())

	return lipgloss.Place(termWidth, termHeight, lipgloss.Center, lipgloss.Center, box)
}

func fmtShortcut(key, desc string, kStyle, dStyle lipgloss.Style) string {
	return "  " + kStyle.Render(fmtPadRight(key, 12)) + " " + dStyle.Render(desc)
}

func fmtPadRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
