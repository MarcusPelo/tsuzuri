<div align="center">

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

Tsuzuri is a single self-contained binary with no dependencies. It runs on
**macOS** (Intel and Apple Silicon), **Windows** (x64, ARM64) and **any Linux
distribution** (x86-64, ARM64, ARMv7 such as Raspberry Pi, x86), plus FreeBSD.

### macOS and Linux

```sh
curl -fsSL https://raw.githubusercontent.com/jaisuriya-11/tsuzuri/main/install.sh | sh
```

The script picks the right build for your system, verifies its checksum and
installs to `/usr/local/bin` (or `~/.local/bin` without sudo). No curl? Use
`wget -qO- … | sh`. Set `TSUZURI_VERSION=v0.1.0` to pin a version or
`TSUZURI_INSTALL_DIR` to choose the folder.

### Windows

In PowerShell:

```powershell
irm https://raw.githubusercontent.com/jaisuriya-11/tsuzuri/main/install.ps1 | iex
```

It installs to `%LOCALAPPDATA%\Programs\tsuzuri` and adds it to your `PATH`.
Use [Windows Terminal](https://aka.ms/terminal) for full colour and mouse support.

### Linux packages

Every [release](https://github.com/jaisuriya-11/tsuzuri/releases/latest) also ships native packages:

| Distro | Command |
| :-- | :-- |
| Debian, Ubuntu, Mint, Pop!_OS | `sudo dpkg -i tsuzuri_*_linux_amd64.deb` |
| Fedora, RHEL, CentOS, openSUSE | `sudo rpm -i tsuzuri_*_linux_amd64.rpm` |
| Alpine | `sudo apk add --allow-untrusted tsuzuri_*_linux_amd64.apk` |
| Arch, Manjaro, EndeavourOS | `sudo pacman -U tsuzuri_*_linux_amd64.pkg.tar.zst` |
| Anything else (NixOS, Void, Gentoo, …) | use the install script or unpack the `.tar.gz` |

Replace `amd64` with `arm64`, `armv7` or `386` for other CPUs.

### Manual download

Grab the `.tar.gz` (or `.zip` on Windows) for your system from the
[latest release](https://github.com/jaisuriya-11/tsuzuri/releases/latest),
unpack it and put `tsuzuri` on your `PATH`. Checksums are in `checksums.txt`.

### With Go

```sh
go install github.com/jaisuriya-11/tsuzuri/cmd/tsuzuri@latest
```

### Requirements and tips

- A terminal with true colour: Windows Terminal, iTerm2, WezTerm, Kitty, Alacritty, Ghostty, GNOME Terminal, Konsole… (macOS Terminal.app works with fewer colours).
- A [Nerd Font](https://www.nerdfonts.com/) for icons (everything works without one; icons just show as boxes).
- Clipboard: works out of the box on macOS and Windows. On Linux install `wl-clipboard` (Wayland) or `xclip` / `xsel` (X11); over SSH Tsuzuri falls back to the terminal's OSC 52 clipboard.
- File dialogs for `/image` use Finder, the Windows picker, or `zenity` / `kdialog` on Linux; otherwise a built-in browser opens.

### Uninstall

Uninstalling never touches your notes: they're ordinary `.md` files in your
own folders. Remove Tsuzuri the same way you installed it.

**Install script (macOS / Linux)**

```sh
rm -f /usr/local/bin/tsuzuri ~/.local/bin/tsuzuri   # add sudo for /usr/local/bin if needed
```

**Install script (Windows, PowerShell)**

```powershell
$dir = "$env:LOCALAPPDATA\Programs\tsuzuri"
Remove-Item $dir -Recurse -Force
$path = ([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_ -and $_ -ne $dir }) -join ';'
[Environment]::SetEnvironmentVariable('Path', $path, 'User')
```

**Linux packages**

| Distro | Command |
| :-- | :-- |
| Debian, Ubuntu, Mint | `sudo apt remove tsuzuri` (or `sudo dpkg -r tsuzuri`) |
| Fedora, RHEL, CentOS | `sudo dnf remove tsuzuri` (or `sudo rpm -e tsuzuri`) |
| openSUSE | `sudo zypper remove tsuzuri` |
| Alpine | `sudo apk del tsuzuri` |
| Arch, Manjaro | `sudo pacman -R tsuzuri` |

**Go**

```sh
rm -f "$(go env GOPATH)/bin/tsuzuri"
```

**Settings and logs (optional)**

Tsuzuri keeps your theme choice and a debug log outside your notes. Delete
them for a completely clean removal:

| OS | Settings | Log |
| :-- | :-- | :-- |
| macOS | `~/Library/Application Support/tsuzuri` | `~/Library/Caches/tsuzuri` |
| Linux | `~/.config/tsuzuri` | `~/.cache/tsuzuri` |
| Windows | `%APPDATA%\tsuzuri` | `%LOCALAPPDATA%\tsuzuri` |

```sh
# macOS
rm -rf ~/Library/Application\ Support/tsuzuri ~/Library/Caches/tsuzuri
# Linux
rm -rf ~/.config/tsuzuri ~/.cache/tsuzuri
```

```powershell
# Windows
Remove-Item "$env:APPDATA\tsuzuri", "$env:LOCALAPPDATA\tsuzuri" -Recurse -Force -ErrorAction SilentlyContinue
```

Images you attached with `/image` or `/cover` live in `assets/` folders next
to your notes; they're part of your notes, so they stay.

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
| Preview | click a cell, `+ Add row`, `+ Add column` | Edit cells, add or delete rows and columns with the mouse |
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
| click `▾` / `▸` | Fold / unfold a heading's section or a code block |
| `zM` / `zR` | Fold / unfold everything |
| `<` / `>` (or `H` / `L`), `T` | Previous / next month, today, in calendar blocks (or click `‹  Today  ›`) |

## Development

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, guidelines and how to send a pull request.

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

Releases are built by [GoReleaser](https://goreleaser.com) in GitHub Actions. Every push runs the tests on Linux, macOS and Windows and test-installs a snapshot build on Ubuntu, Debian, Fedora, Alpine, Arch, openSUSE, macOS and Windows. Push a semver tag to publish binaries, Linux packages and checksums:

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
