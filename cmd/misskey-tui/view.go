package main

import (
	"fmt"
	"strings"
	"time"

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
	quoteBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color("240")).
			PaddingLeft(1).
			MarginLeft(1)

	detailContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("205")).
				Padding(1, 1)

	metadataStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	repliesHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Padding(0, 1).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true)
)

func (m *model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\nAn error occurred: %s\n\nPress any key to return.", m.err)
	}

	if m.loading {
		return fmt.Sprintf("\n\n   %s Loading...\n\n", m.spinner.View())
	}

	if m.mode == "posting" {
		var viewContent strings.Builder
		if m.replyToNote != nil {
			quoteAuthor := fmt.Sprintf("@%s", m.replyToNote.User.Username)
			quoteText := m.replyToNote.Text
			quote := fmt.Sprintf("%s\n%s", quoteAuthor, quoteText)
			viewContent.WriteString(quoteBoxStyle.Render(quote))
			viewContent.WriteString("\n")
		}
		viewContent.WriteString(m.textarea.View())
		viewContent.WriteString("\n\n")
		help := "(Ctrl+S to post, Esc to cancel)"
		viewContent.WriteString(help)
		dialog := dialogBoxStyle.Render(viewContent.String())
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.mode == "detail" {
		// 1. Parent Note (if it exists)
		var parentView string
		if m.parentNote != nil {
			parentAuthor := fmt.Sprintf("Replying to @%s", m.parentNote.User.Username)
			parentInfo := metadataStyle.Render(parentAuthor)
			parentText := m.parentNote.Text
			quote := fmt.Sprintf("%s\n%s", parentInfo, parentText)
			parentView = quoteBoxStyle.Render(quote)
		}

		// 2. Main Note
		var noteContent strings.Builder
		noteContent.WriteString(lipgloss.NewStyle().Bold(true).Render(item{note: *m.selectedNote}.Title()))
		noteContent.WriteString("\n")
		noteContent.WriteString(m.selectedNote.Text)
		noteContent.WriteString("\n\n")

		var reactions []string
		for r, c := range m.selectedNote.Reactions {
			reactions = append(reactions, fmt.Sprintf("%s %d", r, c))
		}
		reactionsStr := strings.Join(reactions, " | ")

		t, err := time.Parse(time.RFC3339, m.selectedNote.CreatedAt)
		var timeStr string
		if err == nil {
			timeStr = t.Local().Format("2006-01-02 15:04:05")
		}

		countsStr := fmt.Sprintf("Replies: %d, Renotes: %d", m.selectedNote.RepliesCount, m.selectedNote.RenoteCount)

		metaData := lipgloss.JoinVertical(lipgloss.Left,
			reactionsStr,
			metadataStyle.Render(countsStr),
			metadataStyle.Render(timeStr),
		)
		noteContent.WriteString(metaData)
		mainNoteView := detailContainerStyle.Render(noteContent.String())

		// 3. Replies
		repliesView := lipgloss.JoinVertical(lipgloss.Left,
			repliesHeaderStyle.Render("Replies"),
			m.detailList.View(),
		)

		// 4. Join them all together
		finalView := lipgloss.JoinVertical(lipgloss.Left,
			parentView,
			mainNoteView,
			repliesView,
		)

		return docStyle.Render(finalView)
	}

	// Timeline view
	timelineTabs := []string{"home", "local", "social", "global"}
	var renderedTabs []string
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

	mainContent := docStyle.Render(m.list.View())

	status := statusMessageStyle.Render(m.statusMessage)
	return tabHeader + "\n" + mainContent + "\n" + status
}
