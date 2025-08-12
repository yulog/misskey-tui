package main

import (
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Model ---

type model struct {
	config        *Config
	client        *http.Client
	list          list.Model
	detailList    list.Model
	textarea      textarea.Model
	spinner       spinner.Model
	timeline      string // "home", "local", "social", "global"
	mode          string // "timeline", "posting", "detail"
	replyToId     string // ID of the note being replied to
	replyToNote   *Note  // The note being replied to
	selectedNote  *Note
	parentNote    *Note // The parent of the selected note
	statusMessage string
	width         int
	height        int
	loading       bool
	err           error
}

// --- Initialization ---

func newModel(config *Config) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	ta := textarea.New()
	ta.Placeholder = "What's on your mind?"
	ta.Focus()

	mainList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	mainList.SetShowTitle(false)
	mainList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "post")),
			key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reply")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "react")),
			key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "renote")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "detail")),
			key.NewBinding(key.WithKeys("h/l/s/g"), key.WithHelp("h/l/s/g", "switch")),
		}
	}

	detailList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	detailList.SetShowTitle(false)
	detailList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reply")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "react")),
			key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "renote")),
		}
	}

	return model{
		config:     config,
		client:     &http.Client{Timeout: 10 * time.Second},
		list:       mainList,
		detailList: detailList,
		textarea:   ta,
		spinner:    s,
		timeline:   "home",
		mode:       "timeline",
		loading:    true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchTimelineCmd())
}
