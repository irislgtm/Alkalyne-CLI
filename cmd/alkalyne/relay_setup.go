package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type relaySetupStep int

const (
	stepName relaySetupStep = iota
	stepHost
	stepPort
	stepResult
	stepDone
)

type relaySetupModel struct {
	step   relaySetupStep
	name   string
	host   string
	port   string
	input  textinput.Model
	output string
	err    string
}

var (
	rsStylePrompt = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	rsStyleTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color("76")).Bold(true)
	rsStyleOutput = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	rsStyleError  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

func initialRelaySetupModel() relaySetupModel {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50
	return relaySetupModel{
		step:  stepName,
		input: ti,
		err:   "",
	}
}

func (m relaySetupModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m relaySetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			return m.advanceStep(), nil
		case "tab":
			return m.advanceStep(), nil
		default:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		return m, nil
	}
	return m, nil
}

func (m relaySetupModel) advanceStep() relaySetupModel {
	switch m.step {
	case stepName:
		m.name = m.input.Value()
		if m.name == "" {
			m.err = "relay name cannot be empty"
			return m
		}
		m.err = ""
		m.step = stepHost
		m.input.SetValue("")
		m.input.Placeholder = "e.g. 203.0.113.42 or relay.example.com"
	case stepHost:
		m.host = m.input.Value()
		if m.host == "" {
			m.err = "host cannot be empty"
			return m
		}
		m.err = ""
		m.step = stepPort
		m.input.SetValue("9000")
		m.input.Placeholder = "default: 9000"
	case stepPort:
		m.port = m.input.Value()
		if m.port == "" {
			m.port = "9000"
		}
		m.err = ""
		m.step = stepResult
		m.generateConfig()
	case stepResult:
		m.step = stepDone
		return m
	}
	return m
}

func (m *relaySetupModel) generateConfig() {
	m.output = fmt.Sprintf(`Relay configuration generated for "%s":

  relay name:  %s
  host:        %s
  port:        %s

Add this to your config.toml:

  [relays]
    [relays.%s]
    peer_id = ""
    addrs = ["/ip4/%s/tcp/%s"]
    enabled = true

On the target machine, run:

  alkalyne daemon --port %s --relay

Share this peer's multiaddr with your contacts once it's running:
  alkalyne daemon will print its peer ID and addresses on startup.
`, m.name, m.name, m.host, m.port, m.name, m.host, m.port, m.port)
}

func (m relaySetupModel) View() string {
	title := rsStyleTitle.Render("Alkalyne Relay Setup Wizard\n\n")

	switch m.step {
	case stepName:
		prompt := rsStylePrompt.Render("Enter a name for this relay:")
		errStr := ""
		if m.err != "" {
			errStr = rsStyleError.Render("\n" + m.err + "\n")
		}
		return title + prompt + "\n" + m.input.View() + errStr + "\n\n[Enter] continue  [Esc/Ctrl+C] quit"

	case stepHost:
		prompt := rsStylePrompt.Render("Enter the relay hostname or IP address:")
		errStr := ""
		if m.err != "" {
			errStr = rsStyleError.Render("\n" + m.err + "\n")
		}
		return title + prompt + "\n" + m.input.View() + errStr + "\n\n[Enter] continue  [Esc/Ctrl+C] quit"

	case stepPort:
		prompt := rsStylePrompt.Render("Enter the relay port (default 9000):")
		return title + prompt + "\n" + m.input.View() + "\n\n[Enter] continue  [Esc/Ctrl+C] quit"

	case stepResult:
		return title + rsStyleOutput.Render(m.output) + "\n[Enter] done  [Esc/Ctrl+C] quit"

	case stepDone:
		return title + "Wizard complete. Press any key to exit.\n"
	}

	return title + "unknown step"
}

func runRelaySetup() {
	p := tea.NewProgram(initialRelaySetupModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "relay setup: %v\n", err)
		os.Exit(1)
	}
}
