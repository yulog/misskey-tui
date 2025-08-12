package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle           = lipgloss.NewStyle().Margin(0, 2)
	tabStyle           = lipgloss.NewStyle().Padding(0, 1)
	activeTabStyle     = tabStyle.Foreground(lipgloss.Color("205")).Bold(true).Underline(true)
	inactiveTabStyle   = tabStyle.Foreground(lipgloss.Color("240"))
	statusMessageStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	dialogBoxStyle     = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 0)
)

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
	replyToId     string // ID of the note being replied to
	statusMessage string
	width         int
	height        int
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
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "post")),
			key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "reply")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "react")),
			key.NewBinding(key.WithKeys("h/l/s/g"), key.WithHelp("h/l/s/g", "switch")),
		}
	}

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
	return tea.Batch(m.spinner.Tick, m.fetchTimelineCmd())
}

// --- Update ---

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v-3) // Adjust for tabs and status
		m.textarea.SetWidth(msg.Width - h - 4)      // Adjust for dialog padding
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
				m.textarea.Placeholder = "What's on your mind?"
				return m, m.textarea.Focus()
			case "R":
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					m.mode = "posting"
					m.replyToId = selectedItem.note.ID
					m.textarea.Placeholder = fmt.Sprintf("Replying to @%s...", selectedItem.note.User.Username)
					return m, m.textarea.Focus()
				}
			case "r":
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					cmds = append(cmds, m.createReactionCmd(selectedItem.note.ID, "❤️"))
				}
			case "h", "l", "s", "g":
				key := msg.String()
				timelineMap := map[string]string{"h": "home", "l": "local", "s": "social", "g": "global"}
				if m.timeline != timelineMap[key] {
					m.timeline = timelineMap[key]
					m.loading = true
					cmds = append(cmds, m.spinner.Tick, m.fetchTimelineCmd())
				}
			}
		case "posting":
			switch msg.String() {
			case "ctrl+s":
				m.loading = true
				cmds = append(cmds, m.spinner.Tick, m.createNoteCmd(m.textarea.Value(), m.replyToId))
				return m, tea.Batch(cmds...)
			case "esc":
				m.mode = "timeline"
				m.textarea.Reset()
				m.replyToId = ""
				return m, nil // Stop the event from propagating.
			}
		}

	case timelineLoadedMsg:
		m.loading = false
		m.list.SetItems(msg.items)

	case notePostedMsg:
		m.loading = false
		m.mode = "timeline"
		m.textarea.Reset()
		m.replyToId = ""
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to post note: %v", msg.err)
		} else {
			m.statusMessage = "Note posted successfully!"
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchTimelineCmd())
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
		// Use the placeholder for the question/prompt
	help := "(Ctrl+S to post, Esc to cancel)"
		ui := fmt.Sprintf("%s\n\n%s\n\n%s", m.textarea.Placeholder, m.textarea.View(), help)
		dialog := dialogBoxStyle.Render(ui)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
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

// --- Commands ---

func (m model) fetchTimelineCmd() tea.Cmd {
	return func() tea.Msg {
		notes, err := fetchTimeline(m.client, m.config, m.timeline)
		if err != nil {
			return errorMsg{err: err}
		}
		items := make([]list.Item, len(notes))
		for i, note := range notes {
			items[i] = item{note: note}
		}
		return timelineLoadedMsg{items: items}
	}
}

func (m model) createNoteCmd(text string, replyId string) tea.Cmd {
	return func() tea.Msg {
		err := createNote(m.client, m.config, text, replyId)
		return notePostedMsg{err: err}
	}
}

func (m model) createReactionCmd(noteId string, reaction string) tea.Cmd {
	return func() tea.Msg {
		err := createReaction(m.client, m.config, noteId, reaction)
		return reactionResultMsg{err: err}
	}
}
