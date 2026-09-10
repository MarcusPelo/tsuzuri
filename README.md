# Tsuzuri (綴り)

[![CI](https://github.com/jaisuriya-11/tsuzuri/actions/workflows/ci.yml/badge.svg)](https://github.com/jaisuriya-11/tsuzuri/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Tsuzuri** (綴|り — *spelling, binding, writing*) is a terminal-based notebook and Markdown workspace application written in **Go**, powered by **Bubble Tea** and **Lip Gloss**.

It combines a React-like component architecture with a Vim-inspired text editor, allowing you to manage multi-page document notes, tasks, and code snippets directly inside your terminal without leaving the keyboard.

---

## 🌟 What the Project Is

Tsuzuri brings a Notion-like multi-document workspace experience to terminal environments with modal Vim editing.

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
