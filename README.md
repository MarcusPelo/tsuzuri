<div align="center">

```text
████████╗███████╗██╗   ██╗███████╗██╗   ██╗██████╗ ██╗
╚══██╔══╝██╔════╝██║   ██║╚══███╔╝██║   ██║██╔══██╗██║
   ██║   ███████╗██║   ██║  ███╔╝ ██║   ██║██████╔╝██║
   ██║   ╚════██║██║   ██║ ███╔╝  ██║   ██║██╔══██╗██║
   ██║   ███████║╚██████╔╝███████╗╚██████╔╝██║  ██║██║
   ╚═╝   ╚══════╝ ╚═════╝ ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝
```

### ~ 綴り • Terminal Markdown Notebook ~

</div>

<p align="center">
  <a href="https://github.com/jaisuriya-11/tsuzuri/actions/workflows/ci.yml"><img src="https://github.com/jaisuriya-11/tsuzuri/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/jaisuriya-11/tsuzuri/releases/latest"><img src="https://img.shields.io/github/v/release/jaisuriya-11/tsuzuri" alt="Release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"></a>
</p>

**Tsuzuri** (綴り — *spelling, binding, writing*) is a Notion-style notebook for the terminal with an NvChad-inspired look. It edits plain Markdown files on disk with a Vim-style editor, a nested page tree, buffer tabs and a live preview, built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Your files, not a database.** Every note is a `.md` file in a normal folder. Use git, sync it, grep it, open it in any other editor.
- **NvChad look and feel.** nvdash-style start screen, nvim-tree explorer with indent guides, a clickable buffer tabline and an NvChad statusline (mode, file, folder, word count, cursor position).
- **VSCode-style unsaved tabs.** `Ctrl+N` opens an `Untitled-1` tab that lives only in memory. The first save opens **Save As**: pick a folder (or type a new one) and a name. `.md` is added automatically and nothing else can be written.
- **Edits are never lost by accident.** Unsaved changes survive tab switches, closing a dirty tab asks *Save / Don't Save / Cancel*, quitting offers *Save All*, and deleting a file asks first.
- **Vim editing.** `NORMAL`, `INSERT` and `COMMAND` modes with motions (`hjkl`, `w`/`b`, `0`/`$`, `gg`/`G`, `Ctrl+D`/`Ctrl+U`), `x`, `dd`, `o`/`O`, and `:w`, `:w name`, `:wq`, `:q!`, `:bd`, `:enew`.
- **Mouse everywhere.** Click tabs, close buttons, the `+` button, tree rows and the editor (places the cursor). The wheel scrolls whichever pane is under the pointer.
- **Live preview.** Headings, checklists, callouts, tables and code blocks render as you type. Code is syntax-highlighted for 250+ languages (also in the editor), and local images (PNG, JPEG, GIF, WebP, BMP) are drawn right in the terminal. Toggle it with `Space p`.
- **96 NvChad themes.** Every base46 theme (onedark, catppuccin, gruvbox, tokyonight, rosepine, nord, everforest, kanagawa, the light ones …). `Space t` opens a picker that previews as you move; the choice is remembered.
- **Notion-style `/` menu.** In INSERT mode, type `/` at the start of a line (or after a space) to insert headings, lists, to-dos, toggles, callouts, quotes, tables, dividers, code, a new sub-page or a link to another note. Keep typing to filter.
- **Attach images and files.** `/image`, `/video`, `/audio` and `/file` open your system's file dialog (Finder on macOS, zenity/kdialog on Linux, the Windows file picker). The file is copied into an `assets/` folder next to the note and linked. Over SSH or without a desktop, a built-in browser opens instead (set `TSUZURI_NATIVE_PICKER=0` to always use it).
- **Views from plain text.** `/board`, `/calendar`, `/timeline`, `/chart` and `/form` insert a fenced block you edit as text (on a board, lines under a `- card` are its description); the preview draws it as a Kanban board, a month calendar, a Gantt-style timeline, a pixel-art chart (bar, hbar, line, pie, donut) or a form:

  ````markdown
  ```chart
  type: bar
  title: Sales
  Jan: 12
  Feb: 20
  ```
  ````
- **Interactive views.** Click in the preview to work with them: add or move board cards, add events by clicking a calendar day, shift or stretch timeline bars, edit chart values, and fill in forms (answers are saved under each question as `= answer`). Every change edits the note's text, so it stays plain Markdown.
- **Page covers.** `/cover` picks a banner image; it's stored as front matter and drawn across the top of the preview as pixel art, Notion-style. Add `icon: ☁️` for a page icon:

  ```markdown
  ---
  cover: assets/gcp.png
  icon: ☁️
  ---
  # GCP- ACE
  ```
- **Global finder.** `Ctrl+P` (or `f` on the start screen) searches every note from the workspace root down: file names (fuzzy) and the text inside notes, with a preview. `Enter` opens the note in the current tab at the matching line, `Ctrl+T` in a new one.
- **Notion-style nesting.** A note can have sub-notes: `Project.md` plus a sibling `Project/` folder.

## Install

### Prebuilt binaries

Download the archive for your OS and CPU from the [latest release](https://github.com/jaisuriya-11/tsuzuri/releases/latest), extract it and put `tsuzuri` somewhere on your `PATH`:

```bash
# example: macOS on Apple Silicon (replace X.Y.Z with the release version)
tar -xzf tsuzuri_X.Y.Z_darwin_arm64.tar.gz
sudo mv tsuzuri /usr/local/bin/
```

Builds are published for Linux, macOS and Windows on `amd64` and `arm64`. Verify downloads against `checksums.txt`.

### With Go

```bash
go install github.com/jaisuriya-11/tsuzuri/cmd/tsuzuri@latest
```

### From source

```bash
git clone https://github.com/jaisuriya-11/tsuzuri.git
cd tsuzuri
make build        # binary at ./bin/tsuzuri
```

Requires the Go version listed in `go.mod`. Icons need a [Nerd Font](https://www.nerdfonts.com/) in your terminal.

## Usage

```bash
tsuzuri                      # open the current directory as the workspace
tsuzuri --dir ~/notes        # open a specific folder (created if missing)
TSUZURI_WORKSPACE=~/notes tsuzuri
tsuzuri --theme catppuccin   # theme for this session
tsuzuri --list-themes
tsuzuri --version
```

Your theme choice is saved to `~/.config/tsuzuri/config.json` (or your OS's config directory):

```json
{ "theme": "tokyonight" }
```

The workspace is picked in this order: `--dir`, then `$TSUZURI_WORKSPACE`, then the current directory. Hidden files and folders (names starting with `.`) are not shown. The debug log goes to your user cache directory (for example `~/Library/Caches/tsuzuri/tsuzuri.log` on macOS, `~/.cache/tsuzuri/tsuzuri.log` on Linux), never into your notes.

### How notes map to files

```text
notes/
├── Inbox.md                 note
├── Project.md               note with sub-notes…
├── Project/                 …which live in this folder
│   ├── Roadmap.md
│   └── Meeting notes.md
└── Archive/                 plain folder
    └── 2025.md
```

Only `.md` files are shown and written, and folders appear in the explorer only when they contain a note somewhere inside. Renaming a note also renames its sub-note folder. Deleting a note also deletes its sub-notes; Tsuzuri asks before deleting anything.

## Keybindings

Leader is `Space`, as in NvChad. Press `?` or `Space h` in the app for the cheatsheet.

### Anywhere

| Key | Action |
| :-- | :-- |
| `Ctrl+N` | New note (unsaved `Untitled-N` tab) |
| `Ctrl+S` | Save; new notes open Save As |
| `\` / `Ctrl+P` | Find a note (Telescope-style; `Enter` opens here, `Ctrl+T` in a new tab) |
| `Ctrl+B` | Toggle explorer |
| `Ctrl+C` | Quit (offers to save unsaved tabs) |

### Normal mode / explorer / preview

| Key | Action |
| :-- | :-- |
| `Tab` / `Shift+Tab` | Next / previous pane |
| `Ctrl+H` / `Ctrl+L` | Pane to the left / right |
| `Space Tab` / `Space Shift+Tab` | Next / previous tab |
| `[` / `]` | Previous / next tab |
| `Space x` | Close tab |
| `Space e` | Focus explorer |
| `Space f` | Find a note |
| `Space n` | New note |
| `Space p` | Toggle preview |
| `Space t` | Theme picker |
| `Space d` | Start screen |
| `Space w` | Save |

### Start screen

| Key | Action |
| :-- | :-- |
| `n` | New note |
| `f` / `\` | Find note |
| `t` | Themes |
| `e` | Open explorer |
| `1`–`5` | Open recent note |
| `j` / `k`, `Enter` | Move, select |
| `q` | Quit |

### Explorer

| Key | Action |
| :-- | :-- |
| `j` / `k` | Move |
| `Enter` / `l` | Open note, or expand / collapse folder |
| `h` | Collapse, or jump to parent |
| `o` | Open a note that has sub-notes |
| `n` | New note in the selected folder |
| `a` | New sub-note under the selection |
| `r` | Rename (`Enter` confirm, `Esc` cancel) |
| `d` | Delete (asks first) |
| `\` | Find a note |
| `W` | Collapse all |

### Editor

| Mode | Key | Action |
| :-- | :-- | :-- |
| NORMAL | `i` `a` `A` `I` | Insert at cursor / after / line end / line start |
| NORMAL | `o` / `O` | Open line below / above |
| NORMAL | `h` `j` `k` `l`, `w` `b`, `0` `$` | Move |
| NORMAL | `gg` / `G` | Top / bottom |
| NORMAL | `Ctrl+D` / `Ctrl+U`, `Ctrl+E` / `Ctrl+Y` | Scroll |
| NORMAL | `x` / `dd` | Delete character / line |
| NORMAL | `yy` / `p` | Copy line / paste from the clipboard |
| NORMAL | `v` / `V` | Visual (character / line) selection: `y` copy, `d` cut, `Esc` cancel |
| Mouse | drag | Select text; it's copied on release (a toast confirms) |
| INSERT | `/` | Block menu (at line start or after a space; `↑↓`, `Enter`, `Esc`) |
| INSERT | `Tab` / `Shift+Tab` | Indent (nests list items) / outdent; in a table, next / previous cell (Tab on the last cell adds a row) |
| COMMAND | `:addrow` `:addcol` `:delrow` `:delcol` `:tablefmt` | Edit the table under the cursor (also in the `/` menu inside a table); columns are re-aligned |
| INSERT | `Esc` | Back to NORMAL |
| COMMAND | `:w` | Save (Save As for new notes) |
| COMMAND | `:w name`, `:saveas name` | Save under a new name (`folder/name` works) |
| COMMAND | `:wq` / `:x` | Save and quit |
| COMMAND | `:q` / `:q!` | Quit / quit discarding changes |
| COMMAND | `:bd` / `:bd!` | Close tab / discard and close |
| COMMAND | `:enew` | New note |
| COMMAND | `:colorscheme name` | Switch theme (`:colo` with no name opens the picker) |

### Save As dialog

| Key | Action |
| :-- | :-- |
| `Tab` | Switch between Folder and Name |
| `↑` / `↓` | Choose a folder (typing filters; unknown folders are created) |
| `Enter` | Save |
| `Esc` | Cancel |

### Preview

| Key | Action |
| :-- | :-- |
| `j` / `k` | Scroll one line |
| `d` / `u` | Half page |
| `g` / `G` | Top / bottom |
| `<` / `>` (or `H` / `L`), `T` | Previous / next month, today, in calendar blocks (or click `‹  Today  ›`) |

## Development

```bash
make            # fmt, vet, test, build
make test
make run
make snapshot   # local release dry run (needs goreleaser)
```

Code layout:

| Package | Role |
| :-- | :-- |
| `cmd/tsuzuri` | Entry point, flags, workspace resolution |
| `internal/core` | Filesystem-backed page store and app messages (no UI code) |
| `internal/app` | Root model: layout, focus, keys, mouse, tabs/buffers, tabline, statusline, dialogs |
| `internal/dashboard` | nvdash-style start screen |
| `internal/sidebar` | nvim-tree style explorer, search, inline rename |
| `internal/content` | Vim editor pane with breadcrumb winbar |
| `internal/textarea` | Vendored bubbles textarea with scrolling and click-to-position additions |
| `internal/preview` | Markdown compiler and preview viewport |
| `internal/ui` | Exact-size pane fitting and modal overlays |
| `internal/theme` | NvChad base46 palettes (`palettes_gen.go` is generated) |
| `internal/config` | Saved preferences |

### Releasing

Releases are built by [GoReleaser](https://goreleaser.com) in GitHub Actions. Push a semver tag and the workflow publishes binaries, archives and checksums:

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

## Credits

- Colour themes are generated from [NvChad base46](https://github.com/NvChad/base46) (MIT).
- Syntax highlighting by [chroma](https://github.com/alecthomas/chroma) (MIT); image decoding by [golang.org/x/image](https://pkg.go.dev/golang.org/x/image) (BSD-3-Clause).
- `internal/textarea` is adapted from [charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) (MIT).

## License

[MIT](LICENSE)
