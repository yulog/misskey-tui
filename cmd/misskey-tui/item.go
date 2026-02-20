package main

import "fmt"

type item struct {
	note Note
}

func (i item) Title() string {
	note := i.note
	isRenote := note.Renote != nil && note.Text == ""
	
	username := note.User.Username
	name := note.User.Name
	
	title := ""
	if name != "" {
		title = fmt.Sprintf("%s (@%s)", name, username)
	} else {
		title = fmt.Sprintf("@%s", username)
	}

	if isRenote {
		return fmt.Sprintf("%s renoted", title)
	}
	return title
}

func (i item) Description() string {
	if i.note.Renote != nil && i.note.Text == "" {
		return i.note.Renote.Text
	}
	return i.note.Text
}

func (i item) FilterValue() string {
	if i.note.Renote != nil && i.note.Text == "" {
		return i.note.User.Username + " " + i.note.Renote.User.Username
	}
	return i.note.User.Username
}
