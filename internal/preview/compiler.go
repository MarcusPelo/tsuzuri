package preview

import (
	"fmt"
	"regexp"
	"strings"

	"tsuzuri/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

var (
	boldRegex       = regexp.MustCompile(`\*\*(.*?)\*\*`)
	italicRegex     = regexp.MustCompile(`\*(.*?)\*`)
	codeRegex       = regexp.MustCompile("`([^`]+)`")
	linkRegex       = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
	strikeRegex     = regexp.MustCompile(`~~(.*?)~~`)
	checkboxRegex   = regexp.MustCompile(`^(\s*)[-*+]\s+\[([ xX])\]\s*(.*)$`)
	bulletRegex     = regexp.MustCompile(`^(\s*)[-*+]\s+(.*)$`)
	numberedRegex   = regexp.MustCompile(`^(\s*)(\d+)\.\s+(.*)$`)
	blockquoteRegex = regexp.MustCompile(`^>\s?(.*)$`)
	dividerRegex    = regexp.MustCompile(`^(\-{3,}|\*{3,}|_{3,})$`)
)

// Compile parses raw Markdown text and returns a styled ANSI string using the given Theme.
func Compile(input string, th theme.Theme, contentWidth int) string {
	if contentWidth < 10 {
		contentWidth = 40
	}

	lines := strings.Split(input, "\n")
	var compiled []string

	inCodeBlock := false
	codeBlockLang := ""
	var codeLines []string

	h1Style := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.SelectedFg)

	h2Style := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.CommandBg)

	h3Style := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.SidebarBg)

	quoteStyle := lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(th.NormalBg).
		PaddingLeft(1).
		Foreground(th.TitleFg).
		Italic(true)

	calloutStyle := lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(th.InsertBg).
		PaddingLeft(1).
		Foreground(th.TitleFg)

	codeBoxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(th.Border).
		Background(th.SelectedBg).
		Padding(0, 1)

	codeLangBadge := lipgloss.NewStyle().
		Bold(true).
		Foreground(th.DarkFg).
		Background(th.NormalBg).
		Padding(0, 1)

	dividerStyle := lipgloss.NewStyle().
		Foreground(th.Border)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Handle Code Blocks (```lang ... ```)
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				inCodeBlock = true
				codeBlockLang = strings.TrimPrefix(trimmed, "```")
				codeBlockLang = strings.TrimSpace(codeBlockLang)
				codeLines = nil
				continue
			} else {
				inCodeBlock = false
				boxWidth := contentWidth - 4
				if boxWidth < 10 {
					boxWidth = 10
				}

				headerBadge := ""
				if codeBlockLang != "" {
					headerBadge = codeLangBadge.Render(codeBlockLang) + " "
				}

				rawCode := strings.Join(codeLines, "\n")
				renderedBox := codeBoxStyle.Width(boxWidth).Render(rawCode)
				compiled = append(compiled, headerBadge, renderedBox, "")
				continue
			}
		}

		if inCodeBlock {
			codeLines = append(codeLines, line)
			continue
		}

		// Horizontal Dividers (---, ***)
		if dividerRegex.MatchString(trimmed) {
			divWidth := contentWidth - 2
			if divWidth < 10 {
				divWidth = 10
			}
			compiled = append(compiled, dividerStyle.Render(strings.Repeat("─", divWidth)), "")
			continue
		}

		// Headings (#, ##, ###)
		if strings.HasPrefix(line, "# ") {
			title := strings.TrimPrefix(line, "# ")
			compiled = append(compiled, "", h1Style.Render("█ "+renderInline(title, th)), "")
			continue
		}
		if strings.HasPrefix(line, "## ") {
			title := strings.TrimPrefix(line, "## ")
			compiled = append(compiled, "", h2Style.Render("▎ "+renderInline(title, th)), "")
			continue
		}
		if strings.HasPrefix(line, "### ") {
			title := strings.TrimPrefix(line, "### ")
			compiled = append(compiled, "", h3Style.Render("▸ "+renderInline(title, th)), "")
			continue
		}

		// Notion Checklists (- [ ] or - [x])
		if matches := checkboxRegex.FindStringSubmatch(line); len(matches) == 4 {
			indent := matches[1]
			checked := matches[2]
			text := matches[3]

			if checked == "x" || checked == "X" {
				checkIcon := lipgloss.NewStyle().Foreground(th.InsertBg).Render("󰄳")
				itemText := lipgloss.NewStyle().Foreground(th.MutedFg).Strikethrough(true).Render(text)
				compiled = append(compiled, fmt.Sprintf("%s%s  %s", indent, checkIcon, itemText))
			} else {
				checkIcon := lipgloss.NewStyle().Foreground(th.SelectedFg).Render("󰄱")
				itemText := renderInline(text, th)
				compiled = append(compiled, fmt.Sprintf("%s%s  %s", indent, checkIcon, itemText))
			}
			continue
		}

		// Bullet lists (- or * or +)
		if matches := bulletRegex.FindStringSubmatch(line); len(matches) == 3 {
			indent := matches[1]
			text := matches[2]
			bulletIcon := lipgloss.NewStyle().Foreground(th.SelectedFg).Render("•")
			compiled = append(compiled, fmt.Sprintf("%s%s %s", indent, bulletIcon, renderInline(text, th)))
			continue
		}

		// Numbered lists (1., 2., etc.)
		if matches := numberedRegex.FindStringSubmatch(line); len(matches) == 4 {
			indent := matches[1]
			num := matches[2]
			text := matches[3]
			numBadge := lipgloss.NewStyle().Bold(true).Foreground(th.SelectedFg).Render(num + ".")
			compiled = append(compiled, fmt.Sprintf("%s%s %s", indent, numBadge, renderInline(text, th)))
			continue
		}

		// Notion Callouts & Blockquotes (> ...)
		if matches := blockquoteRegex.FindStringSubmatch(line); len(matches) == 2 {
			quoteContent := matches[1]
			if strings.Contains(quoteContent, "💡") || strings.Contains(quoteContent, "📌") ||
				strings.Contains(quoteContent, "⚠️") || strings.Contains(quoteContent, "[!NOTE]") ||
				strings.Contains(quoteContent, "[!TIP]") {
				compiled = append(compiled, calloutStyle.Render(renderInline(quoteContent, th)))
			} else {
				compiled = append(compiled, quoteStyle.Render(renderInline(quoteContent, th)))
			}
			continue
		}

		// Empty line
		if trimmed == "" {
			compiled = append(compiled, "")
			continue
		}

		// Regular Paragraph text with inline formatting
		compiled = append(compiled, renderInline(line, th))
	}

	// If codeblock was not closed
	if inCodeBlock && len(codeLines) > 0 {
		rawCode := strings.Join(codeLines, "\n")
		compiled = append(compiled, codeBoxStyle.Render(rawCode))
	}

	return strings.Join(compiled, "\n")
}

// renderInline styles inline Markdown tags: **bold**, *italic*, `code`, [link](url), ~~strike~~.
func renderInline(text string, th theme.Theme) string {
	boldStyle := lipgloss.NewStyle().Bold(true).Foreground(th.TitleFg)
	italicStyle := lipgloss.NewStyle().Italic(true).Foreground(th.TitleFg)
	codeStyle := lipgloss.NewStyle().
		Background(th.SelectedBg).
		Foreground(th.NormalBg).
		Padding(0, 1)
	linkStyle := lipgloss.NewStyle().
		Underline(true).
		Foreground(th.CommandBg)
	strikeStyle := lipgloss.NewStyle().
		Strikethrough(true).
		Foreground(th.MutedFg)

	// Code snippets `code`
	text = codeRegex.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.Trim(m, "`")
		return codeStyle.Render(inner)
	})

	// Bold **text**
	text = boldRegex.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.TrimPrefix(m, "**")
		inner = strings.TrimSuffix(inner, "**")
		return boldStyle.Render(inner)
	})

	// Italic *text*
	text = italicRegex.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.TrimPrefix(m, "*")
		inner = strings.TrimSuffix(inner, "*")
		return italicStyle.Render(inner)
	})

	// Strikethrough ~~text~~
	text = strikeRegex.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.TrimPrefix(m, "~~")
		inner = strings.TrimSuffix(inner, "~~")
		return strikeStyle.Render(inner)
	})

	// Links [title](url)
	text = linkRegex.ReplaceAllStringFunc(text, func(m string) string {
		parts := linkRegex.FindStringSubmatch(m)
		if len(parts) == 3 {
			return linkStyle.Render(parts[1]) + lipgloss.NewStyle().Foreground(th.MutedFg).Render(" ("+parts[2]+")")
		}
		return m
	})

	return text
}
