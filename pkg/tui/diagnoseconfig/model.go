// Copyright 2026 The Fluid Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diagnoseconfig

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	diagpkg "github.com/fluid-cloudnative/fluid-cli/pkg/diagnose"
	"github.com/fluid-cloudnative/fluid-cli/pkg/tui/common"
)

const (
	fieldEndpoint = iota
	fieldAPIKey
	fieldModel
	fieldCount
)

var fieldLabels = []string{
	"LLM endpoint (base URL)",
	"API key",
	"Model",
}

type model struct {
	defaults  diagpkg.LLMFormDefaults
	inputs    []textinput.Model
	focused   int
	status    string
	saved     bool
	cancelled bool
	err       error
}

// Run starts the interactive LLM config form.
func Run(in io.Reader, out io.Writer) error {
	defaults, err := diagpkg.LoadLLMFormDefaults()
	if err != nil {
		return err
	}

	m := newModel(defaults)
	p := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out), tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return err
	}
	resolved, ok := final.(model)
	if !ok {
		return fmt.Errorf("unexpected config model type")
	}
	if resolved.err != nil {
		return resolved.err
	}
	if resolved.cancelled {
		return fmt.Errorf("config setup canceled")
	}
	return nil
}

func newModel(defaults diagpkg.LLMFormDefaults) model {
	inputs := make([]textinput.Model, fieldCount)

	endpoint := textinput.New()
	endpoint.Placeholder = "https://api.openai.com/v1"
	endpoint.CharLimit = 512
	endpoint.Width = 64
	endpoint.SetValue(defaults.Endpoint)
	inputs[fieldEndpoint] = endpoint

	apiKey := textinput.New()
	apiKey.Placeholder = apiKeyPlaceholder(defaults)
	apiKey.CharLimit = 512
	apiKey.Width = 64
	apiKey.EchoMode = textinput.EchoPassword
	apiKey.EchoCharacter = '•'
	inputs[fieldAPIKey] = apiKey

	modelInput := textinput.New()
	modelInput.Placeholder = diagpkg.DefaultLLMModel()
	modelInput.CharLimit = 128
	modelInput.Width = 64
	if defaults.Model != "" {
		modelInput.SetValue(defaults.Model)
	}
	inputs[fieldModel] = modelInput

	m := model{
		defaults: defaults,
		inputs:   inputs,
	}
	m.focusField(0)
	return m
}

func apiKeyPlaceholder(d diagpkg.LLMFormDefaults) string {
	switch {
	case d.APIKeyInEnv:
		return "set via FLUID_LLM_API_KEY (leave blank to keep)"
	case d.APIKeyInFile:
		return "configured in file (leave blank to keep)"
	default:
		return "sk-... (optional; prefer FLUID_LLM_API_KEY)"
	}
}

func (m *model) focusField(idx int) tea.Cmd {
	if idx < 0 || idx >= fieldCount {
		return nil
	}
	m.focused = idx
	cmds := make([]tea.Cmd, 0, fieldCount)
	for i := range m.inputs {
		if i == idx {
			cmds = append(cmds, m.inputs[i].Focus())
			continue
		}
		m.inputs[i].Blur()
	}
	return tea.Batch(cmds...)
}

func (m model) Init() tea.Cmd {
	return m.focusField(m.focused)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "tab", "down":
			return m, m.focusField((m.focused + 1) % fieldCount)
		case "shift+tab", "up":
			return m, m.focusField((m.focused - 1 + fieldCount) % fieldCount)
		case "ctrl+s":
			return m.save()
		case "enter":
			if m.focused < fieldCount-1 {
				return m, m.focusField(m.focused + 1)
			}
			return m.save()
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

func (m *model) save() (tea.Model, tea.Cmd) {
	values := diagpkg.LLMFormValues{
		Endpoint: m.inputs[fieldEndpoint].Value(),
		APIKey:   m.inputs[fieldAPIKey].Value(),
		Model:    m.inputs[fieldModel].Value(),
	}
	if err := diagpkg.ApplyLLMFormValues(values); err != nil {
		m.status = common.Subtle("Error: " + err.Error())
		m.err = err
		return m, nil
	}
	m.saved = true
	m.status = common.Subtle("Saved to " + m.defaults.ConfigPath)
	return m, tea.Quit
}

func (m model) View() string {
	title := common.Title("Fluid Diagnose — LLM Config")
	subtitle := common.Subtle("Configure OpenAI-compatible API settings stored in ~/.fluid/config")
	hint := common.Subtle("Runtime overrides: FLUID_LLM_ENDPOINT, FLUID_LLM_API_KEY, FLUID_LLM_MODEL")

	var b strings.Builder
	for i, label := range fieldLabels {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		if i == m.focused {
			style = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
		}
		fmt.Fprintf(&b, "%s\n%s\n\n", style.Render(label), m.inputs[i].View())
	}

	help := common.Subtle("tab/↑↓: field | enter: next/save | ctrl+s: save | esc: cancel")
	status := m.status
	if status == "" {
		status = help
	}

	return lipgloss.JoinVertical(lipgloss.Left, title, subtitle, hint, "", b.String(), status)
}
