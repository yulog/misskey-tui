package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Messages ---

type timelineLoadedMsg struct{ items []list.Item }
type parentNoteLoadedMsg struct{ note *Note }
type childrenNotesLoadedMsg struct{ notes []Note }
type notePostedMsg struct{ err error }
type noteRenotedMsg struct{ err error }
type reactionResultMsg struct{ err error }
type clearStatusMsg struct{}
type errorMsg struct{ err error }

func (e errorMsg) Error() string { return e.err.Error() }

// --- Update ---

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.onWindowSizeChanged(msg)
		m.help.Width = msg.Width
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
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Post):
				m.mode = "posting"
				m.textarea.Placeholder = "What's on your mind?"
				return m, m.textarea.Focus()
			case key.Matches(msg, m.keys.Reply):
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					m.mode = "posting"
					m.replyToId = selectedItem.note.ID
					m.replyToNote = &selectedItem.note
					m.textarea.Placeholder = fmt.Sprintf("Replying to @%s...", selectedItem.note.User.Username)
					return m, m.textarea.Focus()
				}
			case key.Matches(msg, m.keys.React):
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					cmds = append(cmds, m.createReactionCmd(selectedItem.note.ID, "❤️"))
				}
			case key.Matches(msg, m.keys.Renote):
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					cmds = append(cmds, m.createRenoteCmd(selectedItem.note.ID))
				}
			case key.Matches(msg, m.keys.Detail):
				if selectedItem, ok := m.list.SelectedItem().(item); ok {
					m.loading = true
					m.selectedNote = &selectedItem.note

					// Use target note for children/parent fetching (handle Renote)
					targetNote := m.selectedNote
					if targetNote.Renote != nil && targetNote.Text == "" {
						targetNote = targetNote.Renote
					}

					var batchCmds []tea.Cmd
					batchCmds = append(batchCmds, m.spinner.Tick, m.fetchNoteChildrenCmd(targetNote.ID))
					if targetNote.ReplyId != "" {
						batchCmds = append(batchCmds, m.fetchParentNoteCmd(targetNote.ReplyId))
					}
					cmds = append(cmds, tea.Batch(batchCmds...))
				}
			case key.Matches(msg, m.keys.Switch):
				key := msg.String()
				timelineMap := map[string]string{"h": "home", "l": "local", "s": "social", "g": "global"}
				if m.timeline != timelineMap[key] {
					m.timeline = timelineMap[key]
					m.loading = true
					cmds = append(cmds, m.spinner.Tick, m.fetchTimelineCmd())
				}
			}
		case "posting":
			switch {
			case key.Matches(msg, m.keys.PostSubmit):
				m.loading = true
				cmds = append(cmds, m.spinner.Tick, m.createNoteCmd(m.textarea.Value(), m.replyToId))
				return m, tea.Batch(cmds...)
			case key.Matches(msg, m.keys.PostCancel):
				m.mode = "timeline"
				m.textarea.Reset()
				m.replyToId = ""
				m.replyToNote = nil
				return m, nil
			}
		case "detail":
			switch {
			case key.Matches(msg, m.keys.DetailQuit):
				m.mode = "timeline"
				m.selectedNote = nil
				m.parentNote = nil
				return m, nil
			case key.Matches(msg, m.keys.DetailReply):
				m.mode = "posting"
				m.replyToId = m.selectedNote.ID
				m.replyToNote = m.selectedNote
				m.textarea.Placeholder = fmt.Sprintf("Replying to @%s...", m.selectedNote.User.Username)
				return m, m.textarea.Focus()
			case key.Matches(msg, m.keys.DetailReact):
				cmds = append(cmds, m.createReactionCmd(m.selectedNote.ID, "❤️"))
			case key.Matches(msg, m.keys.DetailRenote):
				cmds = append(cmds, m.createRenoteCmd(m.selectedNote.ID))
			case msg.String() == "tab":
				if m.detailFocus == "note" {
					m.detailFocus = "replies"
				} else {
					m.detailFocus = "note"
				}
			}
		}

	case timelineLoadedMsg:
		m.loading = false
		m.list.SetItems(msg.items)

	case parentNoteLoadedMsg:
		m.parentNote = msg.note
		return m, nil

	case childrenNotesLoadedMsg:
		m.loading = false
		var items []list.Item
		for _, note := range msg.notes {
			items = append(items, item{note: note})
		}
		m.detailList.SetItems(items)
		m.mode = "detail"
		m.detailFocus = "note"

		// Initialize viewport content
		displayNote := m.selectedNote
		if displayNote.Renote != nil && displayNote.Text == "" {
			displayNote = displayNote.Renote
		}

		var noteContent strings.Builder
		if m.selectedNote.Renote != nil && m.selectedNote.Text == "" {
			renoterName := m.selectedNote.User.Name
			if renoterName == "" {
				renoterName = m.selectedNote.User.Username
			}
			noteContent.WriteString(metadataStyle.Render(fmt.Sprintf("Renoted by %s", renoterName)))
			noteContent.WriteString("\n")
		}

		noteContent.WriteString(lipgloss.NewStyle().Bold(true).Render(item{note: *displayNote}.Title()))
		noteContent.WriteString("\n\n")
		noteContent.WriteString(displayNote.Text)
		noteContent.WriteString("\n\n")

		// Metadata
		heartCount := 0
		otherReactions := []string{}
		for _, r := range slices.Sorted(maps.Keys(displayNote.Reactions)) {
			isCustomEmoji := strings.HasPrefix(r, ":") && strings.HasSuffix(r, ":")
			if r == "❤️" || isCustomEmoji {
				heartCount += displayNote.Reactions[r]
			} else {
				otherReactions = append(otherReactions, fmt.Sprintf("%s %d", r, displayNote.Reactions[r]))
			}
		}

		var reactions []string
		if heartCount > 0 {
			reactions = append(reactions, fmt.Sprintf("❤️ %d", heartCount))
		}
		reactions = append(reactions, otherReactions...)
		reactionsStr := strings.Join(reactions, " | ")

		t, err := time.Parse(time.RFC3339, displayNote.CreatedAt)
		var timeStr string
		if err == nil {
			timeStr = t.Local().Format("2006-01-02 15:04:05")
		}

		countsStr := fmt.Sprintf("Replies: %d, Renotes: %d", displayNote.RepliesCount, displayNote.RenoteCount)

		metaData := lipgloss.JoinVertical(lipgloss.Left,
			reactionsStr,
			metadataStyle.Render(countsStr),
			metadataStyle.Render(timeStr),
		)
		noteContent.WriteString(metaData)

		m.viewport.SetContent(noteContent.String())
		m.viewport.YOffset = 0

	case notePostedMsg:
		m.loading = false
		m.mode = "timeline"
		m.textarea.Reset()
		m.replyToId = ""
		m.replyToNote = nil
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to post note: %v", msg.err)
		} else {
			m.statusMessage = "Note posted successfully!"
			m.loading = true
			cmds = append(cmds, m.spinner.Tick, m.fetchTimelineCmd())
		}
		cmds = append(cmds, tea.Tick(3*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} }))

	case noteRenotedMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to renote: %v", msg.err)
		} else {
			m.statusMessage = "Renoted successfully!"
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
	} else {
		switch m.mode {
		case "timeline":
			m.list, cmd = m.list.Update(msg)
			cmds = append(cmds, cmd)
		case "posting":
			m.textarea, cmd = m.textarea.Update(msg)
			m.help, cmd = m.help.Update(msg)
			cmds = append(cmds, cmd)
		case "detail":
			if m.detailFocus == "note" {
				m.viewport, cmd = m.viewport.Update(msg)
			} else {
				m.detailList, cmd = m.detailList.Update(msg)
			}
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *model) onWindowSizeChanged(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height
	h, v := docStyle.GetFrameSize()
	m.list.SetSize(msg.Width-h, msg.Height-v-3)
	m.textarea.SetWidth(msg.Width - h - 4)

	// Detail view adjustments
	m.viewport.Width = msg.Width - h - 4
	m.viewport.Height = (msg.Height - v) / 2
	m.detailList.SetSize(msg.Width-h, msg.Height-v-(m.viewport.Height+8))
}
