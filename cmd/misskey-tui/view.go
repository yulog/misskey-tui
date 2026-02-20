package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) statusBarView() string {
	userInfo := fmt.Sprintf("%s@%s", m.username, m.hostname)
	statusLeft := statusMessageStyle.Render(m.statusMessage)
	statusRight := statusMessageStyle.Render(userInfo)

	spacerWidth := max(m.width-lipgloss.Width(statusLeft)-lipgloss.Width(statusRight), 0)

	status := lipgloss.JoinHorizontal(
		lipgloss.Top,
		statusLeft,
		lipgloss.NewStyle().Width(spacerWidth).Render(""),
		statusRight,
	)

	return status
}

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
		viewContent.WriteString(m.help.View(m.keys))
		dialog := dialogBoxStyle.Render(viewContent.String())
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	if m.mode == "detail" {
		// 1. Parent Note (if it exists)
		var parentView string
		if m.parentNote != nil {
			parentAuthor := fmt.Sprintf("Replying to @%s", m.parentNote.User.Username)
			parentInfo := metadataStyle.Render(parentAuthor)

			textWidth := max(m.width-7, 0)
			wrappedParentText := lipgloss.NewStyle().Width(textWidth).Render(m.parentNote.Text)

			quote := fmt.Sprintf("%s\n%s", parentInfo, wrappedParentText)
			parentView = quoteBoxStyle.Render(quote)
		}

		// 2. Main Note
		var noteContent strings.Builder

		displayNote := m.selectedNote
		isRenote := m.selectedNote.Renote != nil && m.selectedNote.Text == ""

		if isRenote {
			renoterName := m.selectedNote.User.Name
			if renoterName == "" {
				renoterName = m.selectedNote.User.Username
			}
			noteContent.WriteString(metadataStyle.Render(fmt.Sprintf("Renoted by %s", renoterName)))
			noteContent.WriteString("\n")
			displayNote = m.selectedNote.Renote
		}

		noteContent.WriteString(lipgloss.NewStyle().Bold(true).Render(item{note: *displayNote}.Title()))
		noteContent.WriteString("\n")

		textWidth := max(m.width-8, 0)
		wrappedText := lipgloss.NewStyle().Width(textWidth).Render(displayNote.Text)
		noteContent.WriteString(wrappedText)

		noteContent.WriteString("\n\n")

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
		mainNoteView := detailContainerStyle.Render(noteContent.String())

		// Calculate heights and set list height
		status := m.statusBarView()
		parentHeight := lipgloss.Height(parentView)
		mainNoteHeight := lipgloss.Height(mainNoteView)
		repliesHeaderHeight := lipgloss.Height(repliesHeaderStyle.Render("Replies"))
		statusHeight := lipgloss.Height(status)
		listHeight := max(m.height-parentHeight-mainNoteHeight-repliesHeaderHeight-statusHeight, 0)
		m.detailList.SetHeight(listHeight)

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

		return lipgloss.JoinVertical(lipgloss.Left, docStyle.Render(finalView), status)
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

	status := m.statusBarView()

	return tabHeader + "\n" + mainContent + "\n" + status
}
