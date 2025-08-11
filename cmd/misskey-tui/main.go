package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load config.json: %v\nPlease make sure the file exists and is correct.", err)
		os.Exit(1)
	}

	model := newModel(config)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}