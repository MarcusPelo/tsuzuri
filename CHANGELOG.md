# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- All 96 NvChad base46 themes with a live-preview picker (`Space t`, home
  screen, `:colorscheme`), `--theme` / `--list-themes` flags, and the choice
  saved in the user config file. The whole screen is painted with the theme,
  so light themes work on dark terminals.
- Notion-style `/` block menu in INSERT mode (headings, lists, to-do, toggle,
  callout, quote, table, divider, code, media, sub-page, link to page).
- Syntax highlighting (chroma, 250+ languages) for fenced code in the editor
  and preview, plus Markdown colouring in the editor.
- Local images are drawn in the preview with half-block characters.
- Copy: drag to select (auto-copied with a toast), `yy`, visual `v`/`V` +
  `y`/`d`, `p` to paste. Uses the system clipboard, or OSC 52 over SSH.
- `\` opens the finder (instead of `/`).
- `/image`, `/video`, `/audio`, `/file` open the system file dialog (or a
  built-in browser) and copy the file into the note's `assets/` folder.
- Tab indents in INSERT mode (nests list items); `Space Tab` switches tabs.
- Global Telescope-style finder (`Ctrl+P`, `f` on home) searching file names
  (fuzzy) and note text across every folder from the root, with preview; opens in the current tab unless it has unsaved changes.
- NvChad onedark look: nvdash start screen, nvim-tree explorer with indent
  guides, buffer tabline, statusline with mode, file, folder, words and cursor.
- Save As dialog for new notes: choose a folder from a list (or type a new
  one) and a name; `.md` is enforced. `:w name` and `:saveas` too.
- Unsaved buffers kept per tab; prompts before closing a dirty tab, quitting
  with unsaved work (Save All) and deleting files.
- Mouse: clickable tabs, close and `+` buttons, explorer rows, click to place
  the cursor; wheel scrolls the pane under the pointer.
- Vim motions in NORMAL mode (`w b 0 $ gg G Ctrl+D/U Ctrl+E/Y x dd o O A I`).
- Space-leader shortcuts, `Ctrl+S`, `Ctrl+P`, preview toggle.

### Fixed
- Markdown preview: raw escape codes leaking into text, raw HTML, long lines
  cut off instead of wrapped, piles of blank lines.
- Explorer: clicking a file stole keyboard focus; `Ctrl+B` now focuses it
  from any mode.
- Files longer than 99 lines were truncated on load (and the rest lost on save).
- Renaming a note from the explorer erased its content.
- Line numbers of 100+ broke the editor layout.
- Editor only scrolled while typing; now scrolls with motions and the wheel.
- Explorer search box borders were misaligned; panes could overflow and break
  the divider lines.
- Switching tabs discarded unsaved edits.

## [0.1.0] - 2026-10-03

First public release.

### Added
- Pages are plain `.md` files in a real workspace folder, chosen with `--dir`,
  `$TSUZURI_WORKSPACE` or the current directory.
- Notion-style sub-pages stored in a sibling folder named after the page.
- VSCode-style buffer tabs: `[` / `]` to switch, `x` to close, `●` for unsaved.
- New pages stay as unsaved drafts until the first `:w`.
- Sidebar navigation modelled on nvim-tree (`l`/`h` expand/collapse, `o` open).
- File-type icons, empty-state screens, redesigned dashboard, mouse support.
- `--version` flag, prebuilt binaries for Linux, macOS and Windows via GoReleaser.

### Changed
- Module path is now `github.com/jaisuriya-11/tsuzuri`, so `go install` works.
- Debug log moved to the user cache directory, out of the notes folder.
- CI uses the Go version from `go.mod`.

### Fixed
- Notes were lost on quit; everything is now saved to disk.
- Sample pages were injected on every launch.
- Deleting then creating a page could reuse an ID and overwrite another page.
- Deleting a parent page hid its children permanently.
- Sidebar search and selected-row highlighting.
- `.gitignore` rule that ignored the `cmd/tsuzuri` source directory.

[Unreleased]: https://github.com/jaisuriya-11/tsuzuri/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/jaisuriya-11/tsuzuri/releases/tag/v0.1.0
