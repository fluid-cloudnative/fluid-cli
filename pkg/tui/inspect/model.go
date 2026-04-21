package inspect

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	inspectpkg "github.com/fluid-cloudnative/fluid-cli/pkg/inspect"
	"github.com/fluid-cloudnative/fluid-cli/pkg/tui/common"
)

type model struct {
	report     *inspectpkg.DatasetReport
	wide       bool
	activeTab  int
	resources  table.Model
	dataOps    table.Model
	overview   string
	termWidth  int
	termHeight int
}

func Run(in io.Reader, out io.Writer, report *inspectpkg.DatasetReport, wide bool) error {
	m := newModel(report, wide)
	p := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func newModel(report *inspectpkg.DatasetReport, wide bool) model {
	return model{
		report:    report,
		wide:      wide,
		activeTab: 0,
		resources: resourceTable(report, wide),
		dataOps:   dataOpsTable(report),
		overview:  overviewText(report),
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = typed.Width
		m.termHeight = typed.Height
		height := typed.Height - 10
		if height < 5 {
			height = 5
		}
		m.resources.SetHeight(height)
		m.dataOps.SetHeight(height)
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
		m.resources, cmd = m.resources.Update(msg)
	case 2:
		m.dataOps, cmd = m.dataOps.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	title := common.Title(fmt.Sprintf("Fluid Inspect: %s/%s", m.report.Identity.Namespace, m.report.Identity.Name))
	header := common.Subtle(fmt.Sprintf("Phase: %s | Runtimes: %d | DataOps: %d", m.report.Status.Phase, len(m.report.Runtimes), len(m.report.DataOps)))
	tabs := common.Tabs([]string{"Overview", "Resources", "DataOps"}, m.activeTab)

	content := ""
	switch m.activeTab {
	case 0:
		content = common.Panel(m.overview, innerWidth(m.termWidth))
	case 1:
		content = common.Panel(m.resources.View(), innerWidth(m.termWidth))
	case 2:
		content = common.Panel(m.dataOps.View(), innerWidth(m.termWidth))
	}
	help := common.Subtle("tab/shift+tab: switch section | arrows/jk: navigate rows | q: quit")
	return lipgloss.JoinVertical(lipgloss.Left, title, header, tabs, content, help)
}

func innerWidth(total int) int {
	if total <= 0 {
		return 0
	}
	w := total - 4
	if w < 40 {
		return 40
	}
	return w
}

func overviewText(report *inspectpkg.DatasetReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Dataset: %s/%s\n", report.Identity.Namespace, report.Identity.Name)
	fmt.Fprintf(&b, "API: %s (%s)\n", report.Identity.APIVersion, report.Identity.Kind)
	fmt.Fprintf(&b, "Status: phase=%s, files=%s, ufsTotal=%s\n\n", report.Status.Phase, report.Status.FileNum, report.Status.UfsTotal)

	if len(report.Spec.Mounts) == 0 {
		b.WriteString("Mounts: <none>\n")
	} else {
		b.WriteString("Mounts:\n")
		for _, mount := range report.Spec.Mounts {
			name := mount.Name
			if name == "" {
				name = "-"
			}
			fmt.Fprintf(&b, "- %s (%s)\n", mount.MountPoint, name)
		}
	}
	b.WriteString("\nConditions:\n")
	if len(report.Status.Conditions) == 0 {
		b.WriteString("- <none>\n")
	} else {
		for _, c := range report.Status.Conditions {
			fmt.Fprintf(&b, "- %s=%s (%s)\n", c.Type, c.Status, c.Reason)
			if c.Message != "" {
				fmt.Fprintf(&b, "  %s\n", c.Message)
			}
		}
	}

	if len(report.Status.Runtimes) > 0 {
		b.WriteString("\nRuntime Status:\n")
		for _, rt := range report.Status.Runtimes {
			fmt.Fprintf(&b, "- %s [%s] ns=%s type=%s\n", rt.Name, rt.Category, rt.Namespace, rt.Type)
		}
	}
	return b.String()
}

func resourceTable(report *inspectpkg.DatasetReport, wide bool) table.Model {
	columns := []table.Column{
		{Title: "RUNTIME", Width: 16},
		{Title: "KIND", Width: 12},
		{Title: "NAMESPACE", Width: 14},
		{Title: "NAME", Width: 26},
		{Title: "STATUS", Width: 14},
		{Title: "AGE", Width: 8},
	}
	if wide {
		columns = append(columns,
			table.Column{Title: "NODE", Width: 16},
			table.Column{Title: "RESTARTS", Width: 10},
		)
	}

	rows := make([]table.Row, 0)
	for _, runtime := range report.Runtimes {
		for _, resource := range runtime.Resources {
			row := table.Row{runtime.RuntimeName, resource.Kind, orDefault(resource.Namespace, "-"), resource.Name, resource.Status, resource.Age}
			if wide {
				row = append(row, orDefault(resource.Node, "-"), orDefault(resource.Restarts, "-"))
			}
			rows = append(rows, row)
		}
	}
	return common.NewTable(columns, rows)
}

func dataOpsTable(report *inspectpkg.DatasetReport) table.Model {
	columns := []table.Column{
		{Title: "KIND", Width: 14},
		{Title: "NAMESPACE", Width: 14},
		{Title: "NAME", Width: 28},
		{Title: "STATUS", Width: 14},
		{Title: "AGE", Width: 8},
	}
	ops := append([]inspectpkg.ResourceRow(nil), report.DataOps...)
	sort.Slice(ops, func(i, j int) bool {
		if ops[i].Namespace == ops[j].Namespace {
			return ops[i].Name < ops[j].Name
		}
		return ops[i].Namespace < ops[j].Namespace
	})
	rows := make([]table.Row, 0, len(ops))
	for _, op := range ops {
		rows = append(rows, table.Row{op.Kind, orDefault(op.Namespace, "-"), op.Name, op.Status, op.Age})
	}
	return common.NewTable(columns, rows)
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
