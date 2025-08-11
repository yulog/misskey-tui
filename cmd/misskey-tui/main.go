package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// --- Misskey API Structs ---

type Note struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	User User   `json:"user"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// --- Model ---

type item struct {
	note Note
}

func (i item) Title() string {
	if i.note.User.Name != "" {
		return fmt.Sprintf("%s (@%s)", i.note.User.Name, i.note.User.Username)
	}
	return fmt.Sprintf("@%s", i.note.User.Username)
}
func (i item) Description() string { return i.note.Text }
func (i item) FilterValue() string { return i.note.User.Username }

type model struct {
	config   *Config
	client   *http.Client
	list     list.Model
	spinner  spinner.Model
	timeline string // "home", "local", "social", "global"
	loading  bool
	err      error
}

// --- Messages ---

type timelineLoadedMsg struct {
	items []list.Item
}

type errorMsg struct {
	err error
}

func (e errorMsg) Error() string {
	return e.err.Error()
}

// --- Initialization ---

func newModel(config *Config) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		config:   config,
		client:   &http.Client{Timeout: 10 * time.Second},
		list:     list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		spinner:  s,
		timeline: "home", // Start with home timeline
		loading:  true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchTimeline(m.client, m.config, m.timeline))
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var listCmd tea.Cmd
	var spinnerCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		return m, nil

	case tea.KeyMsg:
		// Don't do anything if we're loading
		if m.loading {
			return m, nil
		}

		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "h", "l", "s", "g":
			key := msg.String()
			timelineMap := map[string]string{
				"h": "home",
				"l": "local",
				"s": "social",
				"g": "global",
			}
			if m.timeline != timelineMap[key] {
				m.timeline = timelineMap[key]
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, fetchTimeline(m.client, m.config, m.timeline))
			}
		}

	case timelineLoadedMsg:
		m.loading = false
		m.list.SetItems(msg.items)

	case errorMsg:
		m.loading = false
		m.err = msg
		return m, nil
	}

	if m.loading {
		m.spinner, spinnerCmd = m.spinner.Update(msg)
	} else {
		m.list, listCmd = m.list.Update(msg)
	}

	return m, tea.Batch(listCmd, spinnerCmd)
}

// --- View ---

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nAn error occurred: %s\n\nPress q to quit.", m.err)
	}

	if m.loading {
		return fmt.Sprintf("\n\n   %s Loading %s timeline...\n\n", m.spinner.View(), m.timeline)
	}

	// Set the title of the list based on the current timeline
	m.list.Title = strings.ToTitle(m.timeline) + " Timeline (h/l/s/g)"
	return docStyle.Render(m.list.View())
}

// --- I/O ---

type Config struct {
	InstanceURL string `json:"instance_url"`
	AccessToken string `json:"access_token"`
}

func loadConfig() (*Config, error) {
	file, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func fetchTimeline(client *http.Client, config *Config, timelineType string) tea.Cmd {
	return func() tea.Msg {
		endpointMap := map[string]string{
			"home":   "/api/notes/timeline",
			"local":  "/api/notes/local-timeline",
			"social": "/api/notes/hybrid-timeline",
			"global": "/api/notes/global-timeline",
		}

		endpoint, err := url.JoinPath(config.InstanceURL, endpointMap[timelineType])
		if err != nil {
			return errorMsg{err}
		}

		reqBody, err := json.Marshal(map[string]interface{}{
			"i":     config.AccessToken,
			"limit": 30,
		})
		if err != nil {
			return errorMsg{err}
		}

		req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
		if err != nil {
			return errorMsg{err}
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return errorMsg{err}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return errorMsg{fmt.Errorf("API request failed with status: %s", resp.Status)}
		}

		var notes []Note
		if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
			return errorMsg{err}
		}

		items := make([]list.Item, len(notes))
		for i, note := range notes {
			items[i] = item{note: note}
		}

		return timelineLoadedMsg{items: items}
	}
}

// --- Main ---

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