package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load config.json: %v\nPlease make sure the file exists and is correct.", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	user, err := fetchMe(client, config)
	if err != nil {
		fmt.Printf("Failed to fetch user info: %v\n", err)
		os.Exit(1)
	}

	model := newModel(config, user)
	model.client = client

	p := tea.NewProgram(&model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
