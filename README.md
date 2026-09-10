# Tsuzuri (綴)

A terminal-based notebook and markdown workspace application built with **Go**, **Bubble Tea**, and **Lip Gloss**.

## 🏗 Architecture

Tsuzuri is structured following **The Elm Architecture** mapped onto a React-like component model:

- **`internal/core`**: Domain entities (`Page`, `Block`), thread-safe `Store`, and decoupled `Msg` event definitions (`PageSelectedMsg`, `PageCreatedMsg`, etc.). Zero UI dependencies.
- **`internal/theme`**: Shared design system tokens, color palettes, and Lip Gloss styles.
- **`internal/header`**: Header component (`<Header/>`) rendering status badges and navigation hints.
- **`internal/sidebar`**: Sidebar component (`<Sidebar/>`) rendering document lists, cursor navigation, and inline title renaming.
- **`internal/content`**: Content component (`<Body/>`) hosting the multi-line Vim Markdown editor (`NORMAL`, `INSERT`, `COMMAND`).
- **`internal/app`**: Root orchestrator container (`<App/>`) handling window resize calculations, keymap shortcuts, and child message routing.

## ⌨️ Controls & Shortcuts

| Shortcut | Description |
| :--- | :--- |
| `Tab` | Switch pane focus between Sidebar and Content Editor |
| `Ctrl+N` | Create a new workspace document |
| `Ctrl+C` | Quit application |
| `j` / `k` or `↓` / `↑` | Move selection cursor in Sidebar or Editor |
| `Enter` | Rename selected document in Sidebar |
| `n` / `a` | Create new document (Sidebar focus) |
| `d` / `x` | Delete selected document (Sidebar focus) |
| `i` | Enter Vim Insert mode (Editor focus) |
| `Esc` | Return to Vim Normal mode (Editor focus) |
| `:` | Open Vim Command mode (`:w`, `:q`, `:wq`, `:q!`) |

## 🚀 Getting Started

### Build
```bash
make build
```

### Run
```bash
make run
```

### Test
```bash
make test
```
