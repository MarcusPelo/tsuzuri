package content

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"
)

// VimMode represents the operational mode of the editor.
type VimMode int

const (
	ModeNormal VimMode = iota
	ModeInsert
	ModeCommand
)

func configureTextareaStyles(ta *textarea.Model, th theme.Theme) {
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().Background(th.DarkFg)
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(th.MutedFg)
	ta.FocusedStyle.Text = lipgloss.NewStyle().Foreground(th.TitleFg)
	ta.FocusedStyle.LineNumber = lipgloss.NewStyle().Foreground(th.Border)
	ta.FocusedStyle.CursorLineNumber = lipgloss.NewStyle().Foreground(th.NormalBg)

	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.CursorLine = lipgloss.NewStyle()
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(th.Border)
	ta.BlurredStyle.Text = lipgloss.NewStyle().Foreground(th.MutedFg)
	ta.BlurredStyle.LineNumber = lipgloss.NewStyle().Foreground(th.SelectedBg)
	ta.BlurredStyle.CursorLineNumber = lipgloss.NewStyle().Foreground(th.Border)
}

func parseVimCommand(cmdStr string) (core.VimSaveMsg, core.VimQuitMsg, string, bool, bool) {
	cmdStr = strings.TrimSpace(cmdStr)
	switch cmdStr {
	case "w", "write":
		return core.VimSaveMsg{}, core.VimQuitMsg{}, "written", true, false
	case "q", "quit":
		return core.VimSaveMsg{}, core.VimQuitMsg{Save: false}, "", false, true
	case "wq", "x":
		return core.VimSaveMsg{}, core.VimQuitMsg{Save: true}, "written", true, true
	case "q!":
		return core.VimSaveMsg{}, core.VimQuitMsg{Save: false}, "", false, true
	default:
		if cmdStr != "" {
			return core.VimSaveMsg{}, core.VimQuitMsg{}, "Not an editor command: :" + cmdStr, false, false
		}
		return core.VimSaveMsg{}, core.VimQuitMsg{}, "", false, false
	}
}

func createCommandInput(th theme.Theme) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ":"
	ti.CharLimit = 64
	ti.PromptStyle = lipgloss.NewStyle().Foreground(th.NormalBg).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(th.TitleFg)
	return ti
}
