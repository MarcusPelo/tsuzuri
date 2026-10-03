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

- **Your files, not a database.** Every page is a `.md` file in a normal folder. Use git, sync it, grep it, open it in any other editor.
- **Notion-style nesting.** A page can have sub-pages: `Project.md` plus a sibling `Project/` folder. Plain folders show up as folders.
- **NvChad-style UI.** Dashboard landing screen, tree sidebar, editor, live Markdown preview and a status line.
- **Vim editing.** `NORMAL`, `INSERT` and `COMMAND` modes with `:w`, `:q`, `:wq`, `:x`, `:q!`.
- **Buffer tabs.** Open several pages at once, switch with `[` / `]`, close with `x`. Unsaved tabs show a `●`.
- **No surprise files.** New pages stay in memory until you `:w`, like an unsaved Vim buffer.
- **Live preview.** Headings, checklists, callouts, code blocks and tables render as you type.
- **Search** the tree with `/`, plus mouse support.

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
tsuzuri --version
```

The workspace is picked in this order: `--dir`, then `$TSUZURI_WORKSPACE`, then the current directory. Hidden files and folders (names starting with `.`) are not shown. The debug log goes to your user cache directory (for example `~/Library/Caches/tsuzuri/tsuzuri.log` on macOS, `~/.cache/tsuzuri/tsuzuri.log` on Linux), never into your notes.

### How pages map to files

```text
notes/
├── Inbox.md                 page
├── Project.md               page with sub-pages…
├── Project/                 …which live in this folder
│   ├── Roadmap.md
│   └── Meeting notes.md
└── Archive/                 plain folder
    └── 2025.md
```

Renaming a page renames both its file and its sub-page folder. Deleting a page removes its `.md` file; if it had sub-pages, its folder stays as a plain folder so nothing is lost.

## Keybindings

Press `Space` `h` or `?` inside the app for the cheatsheet.

### Dashboard

| Key | Action |
| :-- | :-- |
| `n` | New page |
| `f` / `Tab` | Open workspace (focus sidebar) |
| `1`–`9` | Open recent page |
| `j` / `k` | Move selection |
| `Enter` | Run selected action |
| `q` / `Ctrl+C` | Quit |

### Global

| Key | Action |
| :-- | :-- |
| `Tab` | Cycle focus: sidebar → editor → preview |
| `Ctrl+B` | Toggle sidebar |
| `Ctrl+N` | New page |
| `Ctrl+D` | Back to dashboard |
| `[` / `]` | Previous / next tab |
| `x` | Close tab (editor, NORMAL mode) |
| `Space` `h` / `?` | Shortcut cheatsheet |
| `Ctrl+C` | Quit |

### Sidebar

| Key | Action |
| :-- | :-- |
| `j` / `k`, `↓` / `↑` | Move cursor |
| `Enter` / `l` / `→` | Open page, or expand / collapse a folder |
| `o` | Open a page that has sub-pages |
| `h` / `←` | Collapse folder |
| `z` / `Space` | Toggle expand / collapse |
| `n` | New top-level page |
| `a` | New sub-page under selection |
| `r` / `e` | Rename (`Enter` confirm, `Esc` cancel) |
| `d` / `x` | Delete |
| `/` | Search (`Enter` / `Esc` to leave) |

### Editor

| Mode | Key | Action |
| :-- | :-- | :-- |
| NORMAL | `i` / `a` | Enter INSERT mode |
| NORMAL | `:` | Enter COMMAND mode |
| NORMAL | `j` / `k` | Move cursor |
| INSERT | `Esc` | Back to NORMAL |
| COMMAND | `:w` / `:write` | Save (a new page becomes a file here) |
| COMMAND | `:wq` / `:x` | Save and quit |
| COMMAND | `:q` / `:quit` / `:q!` | Quit without saving |
| COMMAND | `Esc` | Cancel |

### Preview

| Key | Action |
| :-- | :-- |
| `j` / `k` | Scroll one line |
| `d` / `u`, `Ctrl+D` / `Ctrl+U` | Half-page down / up |
| `g` / `G` | Top / bottom |
| Mouse wheel | Scroll |

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
| `internal/app` | Root model: layout, focus, keymap, tabs, drafts, modal |
| `internal/dashboard` | Landing screen |
| `internal/sidebar` | Page tree, search, inline rename |
| `internal/content` | Vim editor and tab strip |
| `internal/preview` | Markdown compiler and preview viewport |
| `internal/header` | Header bar |
| `internal/theme` | Colors and styles |

### Releasing

Releases are built by [GoReleaser](https://goreleaser.com) in GitHub Actions. Push a semver tag and the workflow publishes binaries, archives and checksums:

```bash
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

## License

[MIT](LICENSE)
