# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
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
