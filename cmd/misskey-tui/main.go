package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sirupsen/logrus"
	"github.com/yitsushi/go-misskey"
	"github.com/yitsushi/go-misskey/models"
	"github.com/yitsushi/go-misskey/services/notes/timeline"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

// --- Model ---

type item struct {
	note *models.Note
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
	client  *misskey.Client
	list    list.Model
	spinner spinner.Model
	loading bool
	err     error
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

func newModel(client *misskey.Client) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		client:  client,
		list:    list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		spinner: s,
		loading: true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchTimeline(m.client))
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
		if m.list.FilterState() == list.Filtering {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

	case timelineLoadedMsg:
		m.loading = false
		m.list.Title = "Home Timeline"
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
		return fmt.Sprintf("\n\n   %s Loading timeline...\n\n", m.spinner.View())
	}

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

func fetchTimeline(client *misskey.Client) tea.Cmd {
	return func() tea.Msg {
		notes, err := client.Notes().Timeline().Get(timeline.GetRequest{
			Limit: 10,
		})
		if err != nil {
			return errorMsg{err}
		}

		items := make([]list.Item, len(notes))
		for i, note := range notes {
			items[i] = item{note: &note}
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

	parsedURL, err := url.Parse(config.InstanceURL)
	if err != nil {
		fmt.Printf("Failed to parse instance URL: %v", err)
		os.Exit(1)
	}

	client, err := misskey.NewClientWithOptions(
		misskey.WithAPIToken(config.AccessToken),
		misskey.WithBaseURL(parsedURL.Scheme, parsedURL.Host, parsedURL.Path),
		misskey.WithLogLevel(logrus.ErrorLevel), // Be less verbose
	)
	if err != nil {
		fmt.Printf("Failed to create Misskey client: %v", err)
		os.Exit(1)
	}

	model := newModel(client)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
