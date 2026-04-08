package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/jwil007/roamctl/internal/ipc"
)

func Tui() error {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tea.NewProgram: %w", err)
	}
	return nil
}

func initialModel() model {
	c, err := connect()
	if err != nil {
		slog.Error("Failed to connect to roamctl", "err", err)
	}
	sc := bufio.NewScanner(c.conn)
	t := table.New(
		table.WithColumns(apTableColumns),
		table.WithStyles(apTableStyles()),
	)
	return model{
		scanner:    sc,
		ringBuffer: nil,
		procState:  &ipc.ProcessState{},
		apTable:    t,
		rssiColors: makeRSSIColors(-78, -30),
	}
}

func (m model) Init() tea.Cmd {
	cmd := readCmd(m.scanner)
	return cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.apTable.SetWidth(m.width)
		m.apTable.SetHeight(m.height / 3)
	case socketMsg:
		err := json.Unmarshal([]byte(msg), m.procState)
		if err != nil {
			slog.Error("Error parsing JSON", "err", err)
		}
		m.apTable.SetRows(m.makeRows())
		rssi := m.procState.ConnState.RSSI
		m.ringBuffer = append(m.ringBuffer, rssi)
		if m.width > 0 && len(m.ringBuffer) > m.width {
			m.ringBuffer = m.ringBuffer[1:]
		}
		return m, readCmd(m.scanner)
	case reconnectMsg:
		m.client.close()
		return m, reconnectCmd()
	case clientMsg:
		m.client = msg
		m.scanner = bufio.NewScanner(m.client.conn)
		return m, readCmd(m.scanner)
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m model) View() tea.View {
	return tea.View{
		Content:   m.viewDashboard(),
		AltScreen: true,
	}
}
