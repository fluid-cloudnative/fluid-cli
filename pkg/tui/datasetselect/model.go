package datasetselect

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fluid-cloudnative/fluid-cli/pkg/tui/common"
)

type item string

func (i item) Title() string       { return string(i) }
func (i item) Description() string { return "Dataset" }
func (i item) FilterValue() string { return string(i) }

type model struct {
	list      list.Model
	namespace string
	selected  string
	cancelled bool
}

func Run(in io.Reader, out io.Writer, datasetNames []string, namespace string) (string, error) {
	items := make([]list.Item, 0, len(datasetNames))
	for _, name := range datasetNames {
		items = append(items, item(name))
	}

	l := list.New(items, list.NewDefaultDelegate(), 60, 12)
	l.Title = fmt.Sprintf("Select Dataset (%s)", namespace)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	m := model{
		list:      l,
		namespace: namespace,
	}

	program := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out), tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return "", err
	}

	resolved, ok := finalModel.(model)
	if !ok {
		return "", fmt.Errorf("unexpected selector model type")
	}
	if resolved.cancelled {
		return "", fmt.Errorf("dataset selection canceled")
	}
	if resolved.selected == "" {
		return "", fmt.Errorf("no dataset selected")
	}
	return resolved.selected, nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			selected := m.list.SelectedItem()
			if selected == nil {
				return m, nil
			}
			m.selected = strings.TrimSpace(selected.FilterValue())
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	title := common.Title("Fluid Inspect")
	subtitle := common.Subtle("Choose a dataset to inspect")
	help := common.Subtle("type to filter | enter: select | q/esc: quit")
	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, m.list.View(), help)
}
