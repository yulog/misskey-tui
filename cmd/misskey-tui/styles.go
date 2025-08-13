package main

import "github.com/charmbracelet/lipgloss"

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
