package tui

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"github.com/jwil007/roamctl/internal/ipc"
)

func Tui(iface *string) error {
	p := tea.NewProgram(initialModel(iface))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tea.NewProgram: %w", err)
	}
	return nil
}

func initialModel(iface *string) model {
	at := table.New(
		table.WithColumns(apTableColumns),
		table.WithStyles(apTableStyle()),
	)
	rt := table.New(
		table.WithColumns(roamTableColumns),
	)
	rt.Focus()
	return model{
		iface:      iface,
		ringBuffer: nil,
		procState:  &ipc.ProcessState{},
		apTable:    at,
		roamTable:  rt,
		rssiColors: makeRSSIColors(-78, -30),
	}
}

func (m model) Init() tea.Cmd {
	return reconnectCmd(m.iface)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.roamTable.SetStyles(m.roamTableStyle())
		m.roamTable.SetWidth(m.width)
		m.roamTable.SetHeight(8)
		m.apTable.SetWidth(m.width)
		m.apTable.SetHeight(11)
		return m, nil
	case socketMsg:
		err := json.Unmarshal([]byte(msg), m.procState)
		if err != nil {
			slog.Error("Error parsing JSON", "err", err)
		}

		//handle rssi bar
		rssi := m.procState.ConnState.RSSI
		m.ringBuffer = append(m.ringBuffer, rssi)
		if m.width > 0 && len(m.ringBuffer) > tuiWidth {
			m.ringBuffer = m.ringBuffer[1:]
		}

		//handle roam table
		if m.isNewRoam() {
			m.lastRoam = m.logRoam()
			m.roamLogs = append(m.roamLogs, m.lastRoam)
		}
		slices.SortFunc(m.roamLogs, func(a, b roamLog) int {
			return int(b.completedAt.Sub(a.completedAt))
		})
		m.roamTable.SetRows(m.makeRoamTableRows())

		//handle ap table
		rows, lastBSSID := m.makeAPRows()
		m.apTable.SetRows(rows)
		m.lastBSSID = lastBSSID
		return m, readCmd(m.scanner)
	case reconnectMsg:
		m.client.close()
		return m, reconnectCmd(m.iface)
	case clientMsg:
		m.client = msg
		m.scanner = bufio.NewScanner(m.client.conn)
		return m, readCmd(m.scanner)
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "down":
			m.roamTable, _ = m.roamTable.Update(msg)
			return m, nil
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
