// Command tsuzuri is the entry point for the Tsuzuri terminal notebook application.
package main

import (
	"fmt"
	"log"
	"os"

	"tsuzuri/internal/app"
	"tsuzuri/internal/core"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Setup file logging via tea.LogToFile
	f, err := tea.LogToFile("tsuzuri.log", "debug")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open log file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	log.Println("Starting Tsuzuri application...")

	store := core.NewStore()
	appModel := app.New(store)

	p := tea.NewProgram(appModel, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Printf("Fatal error running program: %v\n", err)
		fmt.Fprintf(os.Stderr, "Error running Tsuzuri application: %v\n", err)
		os.Exit(1)
	}

	log.Println("Tsuzuri exited cleanly.")
}
