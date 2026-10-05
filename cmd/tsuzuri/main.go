// Command tsuzuri is the entry point for the Tsuzuri terminal notebook application.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jaisuriya-11/tsuzuri/internal/app"
	"github.com/jaisuriya-11/tsuzuri/internal/config"
	"github.com/jaisuriya-11/tsuzuri/internal/core"
	"github.com/jaisuriya-11/tsuzuri/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

// version is stamped at build time via -ldflags "-X main.version=...".
// GoReleaser sets this automatically for tagged release builds.
var version = "dev"

func main() {
	var (
		dir        string
		themeName  string
		showVer    bool
		listThemes bool
	)
	flag.StringVar(&dir, "dir", "", "workspace directory to open (defaults to $TSUZURI_WORKSPACE, then the current directory)")
	flag.BoolVar(&showVer, "version", false, "print the Tsuzuri version and exit")
	flag.StringVar(&themeName, "theme", "", "colour theme for this session (see --list-themes)")
	flag.BoolVar(&listThemes, "list-themes", false, "list the available colour themes and exit")
	flag.Parse()

	if showVer {
		fmt.Printf("tsuzuri %s\n", version)
		return
	}
	if listThemes {
		for _, n := range theme.Names() {
			fmt.Println(n)
		}
		return
	}

	cfgPath := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ignoring unreadable config %s: %v\n", cfgPath, err)
	}
	if themeName == "" {
		themeName = cfg.Theme
	}
	if themeName != "" {
		if _, ok := theme.Get(themeName); !ok {
			fmt.Fprintf(os.Stderr, "Unknown theme %q (run tsuzuri --list-themes)\n", themeName)
			os.Exit(2)
		}
	}

	workspace := resolveWorkspace(dir)

	store, err := core.NewStore(workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open workspace %q: %v\n", workspace, err)
		os.Exit(1)
	}

	logPath := logFilePath()
	f, err := tea.LogToFile(logPath, "debug")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	log.Printf("Starting Tsuzuri (workspace: %s)\n", store.Root())

	appModel := app.New(store, app.WithTheme(themeName), app.WithConfigPath(cfgPath))

	p := tea.NewProgram(appModel, tea.WithAltScreen(), tea.WithMouseAllMotion())
	if _, err := p.Run(); err != nil {
		log.Printf("Fatal error running program: %v\n", err)
		fmt.Fprintf(os.Stderr, "Error running Tsuzuri application: %v\n", err)
		os.Exit(1)
	}

	log.Println("Tsuzuri exited cleanly.")
}

// resolveWorkspace picks the notes directory to open: an explicit --dir flag
// wins, then $TSUZURI_WORKSPACE, falling back to the current directory.
func resolveWorkspace(flagDir string) string {
	if flagDir != "" {
		return flagDir
	}
	if envDir := os.Getenv("TSUZURI_WORKSPACE"); envDir != "" {
		return envDir
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return cwd
}

// logFilePath keeps the debug log out of the user's notes workspace,
// writing it to the OS user cache directory instead.
func logFilePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "tsuzuri.log"
	}
	dir := filepath.Join(cacheDir, "tsuzuri")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "tsuzuri.log"
	}
	return filepath.Join(dir, "tsuzuri.log")
}
