package main

import (
	"net/http"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Keys ---

type keyMap struct {
	// For timeline
	Post   key.Binding
	Reply  key.Binding
	React  key.Binding
	Renote key.Binding
	Detail key.Binding
	Switch key.Binding
	Quit   key.Binding

	// For posting
	PostSubmit key.Binding
	PostCancel key.Binding

	// For detail
	DetailReply  key.Binding
	DetailReact  key.Binding
	DetailRenote key.Binding
	DetailQuit   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.PostSubmit, k.PostCancel}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.PostSubmit, k.PostCancel},
	}
}

func newKeyMap() keyMap {
	return keyMap{
		Post: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "post"),
		),
		Reply: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reply"),
		),
		React: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "react"),
		),
		Renote: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "renote"),
		),
		Detail: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "detail"),
		),
		Switch: key.NewBinding(
			key.WithKeys("h", "l", "s", "g"),
			key.WithHelp("h/l/s/g", "switch"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		PostSubmit: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "post"),
		),
		PostCancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		DetailReply: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reply"),
		),
		DetailReact: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "react"),
		),
		DetailRenote: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "renote"),
		),
		DetailQuit: key.NewBinding(
			key.WithKeys("ctrl+c", "q", "esc"),
			key.WithHelp("q/esc", "quit"),
		),
	}
}

// --- Model ---

type model struct {
	config        *Config
	client        *http.Client
	keys          keyMap
	help          help.Model
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
	keys := newKeyMap()

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
			keys.Post,
			keys.Reply,
			keys.React,
			keys.Renote,
			keys.Detail,
			keys.Switch,
		}
	}

	detailList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	detailList.SetShowTitle(false)
	detailList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			keys.DetailReply,
			keys.DetailReact,
			keys.DetailRenote,
		}
	}

	h := help.New()
	h.ShowAll = true

	return model{
		config:     config,
		client:     &http.Client{Timeout: 10 * time.Second},
		keys:       keys,
		help:       h,
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
