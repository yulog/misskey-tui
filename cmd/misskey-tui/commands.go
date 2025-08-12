package main

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

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

func (m model) fetchParentNoteCmd(noteId string) tea.Cmd {
	return func() tea.Msg {
		note, err := fetchSingleNote(m.client, m.config, noteId)
		if err != nil {
			return errorMsg{err: err}
		}
		return parentNoteLoadedMsg{note: note}
	}
}

func (m model) fetchNoteChildrenCmd(noteId string) tea.Cmd {
	return func() tea.Msg {
		notes, err := fetchNoteChildren(m.client, m.config, noteId)
		if err != nil {
			return errorMsg{err: err}
		}
		return childrenNotesLoadedMsg{notes: notes}
	}
}

func (m model) createNoteCmd(text string, replyId string) tea.Cmd {
	return func() tea.Msg {
		err := createNote(m.client, m.config, text, replyId)
		return notePostedMsg{err: err}
	}
}

func (m model) createRenoteCmd(noteId string) tea.Cmd {
	return func() tea.Msg {
		err := createRenote(m.client, m.config, noteId)
		return noteRenotedMsg{err: err}
	}
}

func (m model) createReactionCmd(noteId string, reaction string) tea.Cmd {
	return func() tea.Msg {
		err := createReaction(m.client, m.config, noteId, reaction)
		return reactionResultMsg{err: err}
	}
}
