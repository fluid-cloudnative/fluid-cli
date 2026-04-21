package common

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	subtleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	tabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("244"))

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62"))
)

func Title(text string) string {
	return titleStyle.Render(text)
}

func Subtle(text string) string {
	return subtleStyle.Render(text)
}

func Panel(text string, width int) string {
	st := panelStyle
	if width > 0 {
		st = st.Width(width)
	}
	return st.Render(text)
}

func Tabs(labels []string, active int) string {
	if len(labels) == 0 {
		return ""
	}
	out := make([]string, 0, len(labels))
	for i, label := range labels {
		if i == active {
			out = append(out, activeTabStyle.Render(label))
			continue
		}
		out = append(out, tabStyle.Render(label))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, out...)
}
