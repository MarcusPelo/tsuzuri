
                                                                                        ████████╗███████╗██╗   ██╗███████╗██╗   ██╗██████╗ ██╗
                                                                                        ╚══██╔══╝██╔════╝██║   ██║╚══███╔╝██║   ██║██╔══██╗██║
                                                                                           ██║   ███████╗██║   ██║  ███╔╝ ██║   ██║██████╔╝██║
                                                                                           ██║   ╚════██║██║   ██║ ███╔╝  ██║   ██║██╔══██╗██║
                                                                                           ██║   ███████║╚██████╔╝███████╗╚██████╔╝██║  ██║██║
                                                                                           ╚═╝   ╚══════╝ ╚═════╝ ╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝

**Tsuzuri** (綴|り — *spelling, binding, writing*) is a terminal-based notebook and Markdown workspace application written in **Go**, powered by **Bubble Tea** and **Lip Gloss**.

It combines a React-like component architecture with a Vim-inspired text editor, allowing you to manage multi-page document notes, tasks, and code snippets directly inside your terminal without leaving the keyboard.

- **`internal/core`**: Domain entities (`Page`, `Block`), thread-safe `Store`, and decoupled `Msg` event definitions (`PageSelectedMsg`, `PageCreatedMsg`, etc.). Zero UI dependencies.
- **`internal/theme`**: Shared design system tokens, color palettes, and Lip Gloss styles.
- **`internal/dashboard`**: Landing page component (`<Dashboard/>`) with NvChad-style ASCII banner, quick navigation buttons, and recent notes.
- **`internal/header`**: Header component (`<Header/>`) rendering status badges, 3-pane focus indicators, and navigation hints.
- **`internal/sidebar`**: Notion-style sidebar component (`<Sidebar/>`) rendering workspace header, emoji-tagged document lists, cursor navigation, and inline title renaming.
- **`internal/content`**: Content component (`<Body/>`) hosting the multi-line Vim Markdown code editor (`NORMAL`, `INSERT`, `COMMAND`).
- **`internal/preview`**: Live Markdown compilation engine and viewport component (`<Preview/>`), compiling and rendering rich Markdown styles (headings, checklists, callouts, code blocks, tables) in real time on every keystroke.
- **`internal/app`**: Root orchestrator container (`<App/>`) handling 3-pane window resize calculations, keymap shortcuts, view mode switching, and child message routing.

## 🌟 What the Project Is

### Landing Page (NvChad Dashboard)

| Shortcut | Description |
| :--- | :--- |
| `n` | Create a new document and open editor |
| `f` / `Tab` | Open workspace and focus Sidebar |
| `1` - `9` | Directly open recent document by index |
| `j` / `k` or `↓` / `↑` | Move menu selection cursor |
| `Enter` | Trigger selected dashboard action |
| `q` / `Ctrl+C` | Quit application |

### Workspace (Notion Mode)

| Shortcut | Description |
| :--- | :--- |
| `Ctrl+B` | Toggle sidebar on/off |
| `Space + h` or `?` | Open **Keymap Modal** cheatsheet popup |
| `Tab` | Cycle pane focus between **Sidebar** $\rightarrow$ **Markdown Editor** $\rightarrow$ **Live Preview** |
| `Ctrl+D` | Return to NvChad Landing Page |
| `Ctrl+N` | Create a new workspace document |
| `Ctrl+C` | Quit application |
| `Esc` | Close modal / Return to Normal mode / Clear search |

#### Sidebar Navigation & Nested Tree
| Shortcut | Description |
| :--- | :--- |
| `/` | Focus search input to filter documents in real-time |
| `j` / `k` or `↓` / `↑` | Move selection cursor up / down |
| `Enter` / `r` / `e` | Inline edit / rename selected document |
| `z` / `Space` / `l` / `h` | Expand / collapse nested folder or sub-pages |
| `a` | Add sub-page under currently selected page |
| `n` | Create new top-level page |
| `d` / `x` | Delete selected page |

#### Markdown Editor & Live Preview
| Shortcut | Description |
| :--- | :--- |
| `i` | Enter Vim Insert mode (live compiles to Preview on keystroke) |
| `Esc` | Return to Normal mode |
| `:` | Open Vim Command mode (`:w`, `:q`, `:wq`, `:q!`) |
| `j` / `k` (Preview) | Scroll preview down / up line by line |
| `d` / `u` (Preview) | Half-page scroll down / up |
| `g` / `G` (Preview) | Jump to top / bottom |

### Key Features
- **React-Style Component Architecture**: Clean separation between root container (`<App/>`), header (`<Header/>`), sidebar (`<Sidebar/>`), editor (`<Body/>`), and domain state (`core.Store`).
- **Vim Modal Text Editor**: Full support for `NORMAL`, `INSERT` (press `i`), and `COMMAND` (press `:`) modes including Vim commands (`:w`, `:q`, `:wq`, `:q!`).
- **Dynamic Document Workspace**: Create, rename, delete, and switch between workspace documents dynamically.
- **Production Hardened**: Guarded slice indexing, bounds-checked line math, small terminal window protection, and zero-panic error handling.
- **Gruvbox Dark Aesthetic Palette**: Styled with curated terminal colors via Lip Gloss.

---

## 📦 Install

### Prerequisites
- **Go 1.22** or higher installed on your machine.

### Option 1: Build from Source
Clone the repository and build the executable binary:

```bash
git clone https://github.com/jaisuriya-11/tsuzuri.git
cd tsuzuri
make build
```

The compiled binary will be placed at `./bin/tsuzuri`.

### Option 2: Go Install
Install directly using the Go toolchain:

```bash
go install tsuzuri/cmd/tsuzuri@latest
```

---

## 🚀 Usage

### Running Tsuzuri

Start the interactive terminal workspace:

```bash
./bin/tsuzuri
```

Or run directly with Go without compiling:

```bash
make run
# or
go run ./cmd/tsuzuri
```

---

## ⌨️ Keybindings

### Global Shortcuts

| Keybinding | Action |
| :--- | :--- |
| `Tab` | Switch focus between Sidebar pane and Editor pane |
| `Ctrl+N` | Create a new document page |
| `Ctrl+C` | Exit application |

### Sidebar Navigation & Actions (Sidebar Focus)

| Keybinding | Action |
| :--- | :--- |
| `j` / `k` or `↓` / `↑` | Move selection cursor up / down |
| `Enter` | Enter inline title rename mode for selected document |
| `n` or `a` | Create a new workspace document |
| `d` or `x` | Delete selected document |

### Editor & Vim Modes (Editor Focus)

| Mode | Keybinding | Action |
| :--- | :--- | :--- |
| **Normal** | `i` or `a` | Enter **INSERT** mode to begin typing |
| **Normal** | `:` | Open **COMMAND** prompt (`:w`, `:q`, `:wq`, `:q!`) |
| **Normal** | `j` / `k` or `↓` / `↑` | Move editor cursor down / up |
| **Insert** | `Esc` | Return to **NORMAL** mode |
| **Command** | `Enter` | Execute Vim command (`:w` save, `:q` quit, `:wq` write & quit) |
| **Command** | `Esc` | Cancel command prompt and return to **NORMAL** mode |

---

## 📜 License

This project is licensed under the [MIT License](LICENSE).
