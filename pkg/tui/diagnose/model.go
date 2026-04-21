package diagnose

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fluid-cloudnative/fluid-cli/pkg/tui/common"
)

type ManifestEntry struct {
	Path   string
	Status string
	Reason string
}

type ViewData struct {
	Dataset             string
	Namespace           string
	OutputPath          string
	ArchivePath         string
	PartialFailureCount int
	Summary             string
	WarningEvents       []string
	Artifacts           []ManifestEntry
}

type model struct {
	data       ViewData
	activeTab  int
	artifacts  table.Model
	eventTable table.Model
	width      int
	height     int
}

func Run(in io.Reader, out io.Writer, data ViewData) error {
	m := model{
		data:       data,
		activeTab:  0,
		artifacts:  artifactTable(data.Artifacts),
		eventTable: warningEventTable(data.WarningEvents),
	}
	p := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		height := typed.Height - 10
		if height < 5 {
			height = 5
		}
		m.artifacts.SetHeight(height)
		m.eventTable.SetHeight(height)
		return m, nil
	case tea.KeyMsg:
		switch typed.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "tab", "right", "l":
			m.activeTab = (m.activeTab + 1) % 3
			return m, nil
		case "shift+tab", "left", "h":
			m.activeTab--
			if m.activeTab < 0 {
				m.activeTab = 2
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.activeTab {
	case 1:
		m.artifacts, cmd = m.artifacts.Update(msg)
	case 2:
		m.eventTable, cmd = m.eventTable.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	title := common.Title(fmt.Sprintf("Fluid Diagnose: %s/%s", m.data.Namespace, m.data.Dataset))
	head := common.Subtle(fmt.Sprintf("Artifacts: %s | Archive: %s | Partial failures: %d", valueOr(m.data.OutputPath, "-"), valueOr(m.data.ArchivePath, "-"), m.data.PartialFailureCount))
	tabs := common.Tabs([]string{"Overview", "Artifacts", "Warnings"}, m.activeTab)

	var content string
	switch m.activeTab {
	case 0:
		content = common.Panel(m.data.Summary, contentWidth(m.width))
	case 1:
		content = common.Panel(m.artifacts.View(), contentWidth(m.width))
	case 2:
		content = common.Panel(m.eventTable.View(), contentWidth(m.width))
	}
	help := common.Subtle("tab/shift+tab: switch section | arrows/jk: navigate rows | q: quit")
	return lipgloss.JoinVertical(lipgloss.Left, title, head, tabs, content, help)
}

func artifactTable(entries []ManifestEntry) table.Model {
	columns := []table.Column{
		{Title: "STATUS", Width: 12},
		{Title: "PATH", Width: 48},
		{Title: "REASON", Width: 44},
	}
	rows := make([]table.Row, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, table.Row{e.Status, e.Path, e.Reason})
	}
	return common.NewTable(columns, rows)
}

func warningEventTable(events []string) table.Model {
	columns := []table.Column{
		{Title: "WARNING EVENT", Width: 108},
	}
	rows := make([]table.Row, 0, len(events))
	for _, event := range events {
		rows = append(rows, table.Row{event})
	}
	return common.NewTable(columns, rows)
}

func contentWidth(total int) int {
	if total <= 0 {
		return 0
	}
	width := total - 4
	if width < 40 {
		return 40
	}
	return width
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
