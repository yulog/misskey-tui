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
	"github.com/charmbracelet/bubbles/textarea"
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
	textarea textarea.Model
	spinner  spinner.Model
	timeline string // "home", "local", "social", "global"
	mode     string // "timeline", "posting"
	loading  bool
	err      error
}

// --- Messages ---

type timelineLoadedMsg struct {
	items []list.Item
}

type notePostedMsg struct {
	err error
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

	ta := textarea.New()
	ta.Placeholder = "What's on your mind?"
	ta.Focus()

	return model{
		config:   config,
		client:   &http.Client{Timeout: 10 * time.Second},
		list:     list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		textarea: ta,
		spinner:  s,
		timeline: "home",
		mode:     "timeline",
		loading:  true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchTimeline(m.client, m.config, m.timeline))
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
		m.textarea.SetWidth(msg.Width - h)
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case "timeline":
			if m.loading {
				return m, nil
			}
			if m.list.FilterState() == list.Filtering {
				break
			}
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "p":
				m.mode = "posting"
				return m, m.textarea.Focus()
			case "h", "l", "s", "g":
				key := msg.String()
				timelineMap := map[string]string{"h": "home", "l": "local", "s": "social", "g": "global"}
				if m.timeline != timelineMap[key] {
					m.timeline = timelineMap[key]
					m.loading = true
					cmds = append(cmds, m.spinner.Tick, fetchTimeline(m.client, m.config, m.timeline))
				}
			}
		case "posting":
			switch msg.String() {
			case "ctrl+s":
				m.loading = true
				cmds = append(cmds, m.spinner.Tick, createNote(m.client, m.config, m.textarea.Value()))
				return m, tea.Batch(cmds...)
			case "esc":
				m.mode = "timeline"
				m.textarea.Reset()
			}
		}

	case timelineLoadedMsg:
		m.loading = false
		m.list.SetItems(msg.items)

	case notePostedMsg:
		m.loading = false
		m.mode = "timeline"
		m.err = msg.err // Use the general error field
		m.textarea.Reset()
		if msg.err == nil {
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, fetchTimeline(m.client, m.config, m.timeline))
		}

	case errorMsg:
		m.loading = false
		m.err = msg.err
	}

	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.mode == "timeline" {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.mode == "posting" {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// --- View ---

func (m model) View() string {
	if m.err != nil {
		// Simple error view
		return fmt.Sprintf("\nAn error occurred: %s\n\nPress any key to return to the timeline.", m.err)
	}

	if m.mode == "posting" {
		return fmt.Sprintf(
			"\n%s\n\n%s",
			m.textarea.View(),
			"(Ctrl+S to post, Esc to cancel)",
		) + "\n"
	}

	if m.loading {
		return fmt.Sprintf("\n\n   %s Loading...\n\n", m.spinner.View())
	}

	m.list.Title = strings.ToTitle(m.timeline) + " Timeline (p:post, h/l/s/g)"
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
			"home": "/api/notes/timeline", "local": "/api/notes/local-timeline",
			"social": "/api/notes/hybrid-timeline", "global": "/api/notes/global-timeline",
		}
		endpoint, _ := url.JoinPath(config.InstanceURL, endpointMap[timelineType])
		reqBody, _ := json.Marshal(map[string]interface{}{"i": config.AccessToken, "limit": 30})
		req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil { return errorMsg{err: err} }
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK { return errorMsg{err: fmt.Errorf("API request failed: %s", resp.Status)} }
		var notes []Note
		if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil { return errorMsg{err: err} }
		items := make([]list.Item, len(notes))
		for i, note := range notes { items[i] = item{note: note} }
		return timelineLoadedMsg{items: items}
	}
}

func createNote(client *http.Client, config *Config, text string) tea.Cmd {
	return func() tea.Msg {
		endpoint, _ := url.JoinPath(config.InstanceURL, "/api/notes/create")
		reqBody, _ := json.Marshal(map[string]interface{}{"i": config.AccessToken, "text": text})
		req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil { return notePostedMsg{err: err} }
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK { return notePostedMsg{err: fmt.Errorf("API request failed: %s", resp.Status)} }
		return notePostedMsg{err: nil}
	}
}

// --- Main ---

func main() {
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Failed to load config.json: %v", err)
		os.Exit(1)
	}
	model := newModel(config)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}