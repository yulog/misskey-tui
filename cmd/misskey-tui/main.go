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

var (
	docStyle           = lipgloss.NewStyle().Margin(0, 2)
	tabStyle           = lipgloss.NewStyle().Padding(0, 1)
	activeTabStyle     = tabStyle.Copy().Foreground(lipgloss.Color("205")).Bold(true).Underline(true)
	inactiveTabStyle   = tabStyle.Copy().Foreground(lipgloss.Color("240"))
	statusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
)

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
	config        *Config
	client        *http.Client
	list          list.Model
	textarea      textarea.Model
	spinner       spinner.Model
	timeline      string // "home", "local", "social", "global"
	mode          string // "timeline", "posting"
	statusMessage string
	loading       bool
	err           error
}

// --- Messages ---

type timelineLoadedMsg struct{ items []list.Item }
type notePostedMsg struct{ err error }
type reactionResultMsg struct{ err error }
type clearStatusMsg struct{}
type errorMsg struct{ err error }

func (e errorMsg) Error() string { return e.err.Error() }

// --- Initialization ---

func newModel(config *Config) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ta := textarea.New()
	ta.Placeholder = "What's on your mind?"
	ta.Focus()

	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false) // We'll render our own header

	return model{
		config:   config,
		client:   &http.Client{Timeout: 10 * time.Second},
		list:     l,
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
		m.list.SetSize(msg.Width-h, msg.Height-v-3) // Adjust for tabs and status
		m.textarea.SetWidth(msg.Width - h)
		return m, nil

	case tea.KeyMsg:
		if m.err != nil {
			m.err = nil
			return m, nil
		}

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
			case "r":
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					cmds = append(cmds, createReaction(m.client, m.config, selectedItem.note.ID, "❤️"))
				}
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
				return m, nil
			}
		}

	case timelineLoadedMsg:
		m.loading = false
		m.list.SetItems(msg.items)

	case notePostedMsg:
		m.loading = false
		m.mode = "timeline"
		m.textarea.Reset()
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to post note: %v", msg.err)
		} else {
			m.statusMessage = "Note posted successfully!"
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, fetchTimeline(m.client, m.config, m.timeline))
		}
		cmds = append(cmds, tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} }))

	case reactionResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to react: %v", msg.err)
		} else {
			m.statusMessage = "Reacted with ❤️"
		}
		cmds = append(cmds, tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} }))

	case clearStatusMsg:
		m.statusMessage = ""

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
		return fmt.Sprintf("\nAn error occurred: %s\n\nPress any key to return.", m.err)
	}

	if m.mode == "posting" {
		return fmt.Sprintf("\n%s\n\n%s", m.textarea.View(), "(Ctrl+S to post, Esc to cancel)") + "\n"
	}

	// Tabs
	timelineTabs := []string{"home", "local", "social", "global"}
	renderedTabs := []string{}
	for _, t := range timelineTabs {
		var style lipgloss.Style
		if t == m.timeline {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		renderedTabs = append(renderedTabs, style.Render(strings.ToTitle(t)))
	}
	tabHeader := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	var mainContent string
	if m.loading {
		mainContent = fmt.Sprintf("\n\n   %s Loading...\n\n", m.spinner.View())
	} else {
		mainContent = docStyle.Render(m.list.View())
	}

	status := statusMessageStyle.Render(m.statusMessage)
	return tabHeader + "\n" + mainContent + "\n" + status
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
		if err != nil {
			return errorMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return errorMsg{err: fmt.Errorf("API request failed: %s", resp.Status)}
		}
		var notes []Note
		if err := json.NewDecoder(resp.Body).Decode(&notes); err != nil {
			return errorMsg{err: err}
		}
		items := make([]list.Item, len(notes))
		for i, note := range notes {
			items[i] = item{note: note}
		}
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
		if err != nil {
			return notePostedMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return notePostedMsg{err: fmt.Errorf("API request failed: %s", resp.Status)}
		}
		return notePostedMsg{err: nil}
	}
}

func createReaction(client *http.Client, config *Config, noteId string, reaction string) tea.Cmd {
	return func() tea.Msg {
		endpoint, _ := url.JoinPath(config.InstanceURL, "/api/notes/reactions/create")
		reqBody, _ := json.Marshal(map[string]interface{}{"i": config.AccessToken, "noteId": noteId, "reaction": reaction})
		req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return reactionResultMsg{err: err}
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			return reactionResultMsg{err: fmt.Errorf("API request failed: %s", resp.Status)}
		}
		return reactionResultMsg{err: nil}
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
