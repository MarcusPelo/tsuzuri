# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Tsuzuri is a terminal Markdown notebook (Go, Bubble Tea + Lip Gloss). This repo is a fork of `jaisuriya-11/tsuzuri`; the Go module path is still `github.com/jaisuriya-11/tsuzuri`, so imports use that path.

## Commands

```sh
make            # gofmt -w + go vet + go test + build to ./bin/tsuzuri (run before pushing)
make test       # go test ./...
make run        # go run ./cmd/tsuzuri (opens the current folder as the workspace)
go run ./cmd/tsuzuri --dir /tmp/tsuzuri-notes   # try changes against a scratch notes folder, not the repo
go test ./internal/app -run TestName            # single test
go run ./cmd/tsuzuri export note.md -o out.pdf  # PDF export path, no TUI
```

CI (`.github/workflows/ci.yml`) fails on `gofmt -l .` output, `go vet`, or tests on Linux/macOS/Windows, plus a GoReleaser snapshot and install-script tests. The build must stay pure Go (`CGO_ENABLED=0`), single static binary. Debug log goes to the OS cache dir (`~/.cache/tsuzuri/tsuzuri.log` on Linux), never the workspace.

## Architecture

- **`cmd/tsuzuri`**: flags, config, theme registration (custom themes path must be set before any theme lookup), the `export` subcommand, then `app.New(store, ...)` runs in a `tea.Program`.
- **`internal/core`**: filesystem `Store` and the message types (`events.go`) panes use to talk to the app. No UI imports allowed. A `Page.ID` is the slash-separated path of the file relative to the workspace root, recomputed from disk on every `List`/`Get`; there is no database.
- **`internal/app`**: the root Bubble Tea model. It owns layout, focus, tabs/buffers, dialogs, the finder and theme switching, and routes `core.*Msg` messages emitted by child panes (e.g. `VimSaveMsg`, `PageSelectedMsg`, `ExportMsg`). Files are split by concern: `app_update.go` / `app_keys.go` / `app_mouse.go` / `app_view.go`.
- **`internal/content`**: the modal (vim-like) editor pane, the `/` slash menu and table editing. Built on `internal/textarea`, a vendored copy of bubbles' textarea; local changes there are marked with `tsuzuri:` comments.
- **`internal/preview`**: compiles Markdown source into terminal output (`compiler*.go`, `CompileBlocks`). Alongside the rendered text it returns `Hit` regions (screen rect + source line / fence lines) and `Block` ranges. Rich views (board, calendar, timeline, chart, form, flowchart, math, columns, covers) are fenced code blocks rendered by `block_*.go`, `flow*.go`, `math.go`.
- **Interactive views round-trip through the Markdown source**: a click in the preview becomes a `preview.HitMsg`; `internal/app/viewaction*.go` interprets it and edits the note's text (e.g. moving a board card rewrites lines in the fence). Notes must stay plain, hand-editable Markdown, so any new feature stores its state as text in the note.
- **`internal/export`**: PDF via `go-pdf/fpdf`, reusing preview rendering for boards/charts/flowcharts.
- **`internal/theme`**: `palettes_gen.go` is generated from NvChad base46 (do not edit by hand); add hand-written palettes in `palettes_extra.go`. User themes are JSON files loaded from the config dir or `themes_dir`.
- **`internal/ui`**: exact-size helpers. Every pane must render at exactly its given width and height (`ui.Fit` / `ui.FitLine`); one extra cell breaks the dividers.

## Conventions (from CONTRIBUTING.md)

- Never discard unsaved work without asking; files are written only when the user saves.
- When adding or changing a key binding, update the in-app cheatsheet in `internal/app/modal.go` (the README does not list keys).
- Cross-platform: use `path/filepath` for paths, `/` in note IDs, no OS-specific shell commands without a fallback.
- UI flows are tested by driving the real model with the `harness` in `internal/app/app_test.go` (send keys/clicks, assert on the rendered frame and saved file). Tests must not touch the real clipboard, config or home: use `app.WithClipboard`, `t.TempDir()`, `t.Setenv`. Bug fixes come with a test that fails without the fix.
- New dependencies need a permissive license (MIT/BSD/Apache-2.0).
- Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, ...). Stage files by name; it's easy to commit test notes or images by accident.
