package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

func (m model) viewDashboard() string {
	return baseStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		m.borderStyle().Render(m.titleView()),
		m.borderStyle().Render(m.rssiBarView()),
		m.borderStyle().Render(m.headerView()),
		m.borderStyle().Render(m.statePanelView()),
		m.borderStyle().Render(m.roamStatsView()),
		m.apTableView()))
}

func (m model) titleView() string {
	left := lipgloss.NewStyle().Bold(true).Render("roamctl-tui")
	center := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(time.Now().Format("15:04:05"))
	right := lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render("q to quit")

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	centerWidth := m.width - leftWidth - rightWidth

	return left +
		lipgloss.NewStyle().Width(centerWidth).Align(lipgloss.Center).Render(center) +
		right
}

func (m model) headerView() string {
	if m.procState.SSID == "" {
		return m.allignCenter().Render("Waiting for data from roamctl... \n" +
			"Ensure roamctl is running. Run: sudo systemctl status roamctl")
	}
	header := m.allignCenter().
		Render(lipgloss.NewStyle().Bold(true).
			Render("Current Connection Status"))
	s := fmt.Sprintf("ssid: %v   bssid: %v wpa_state: %v\n"+
		"rssi: %v   tx: %vMbps rx: %vMbps   duration: %v",
		m.procState.SSID,
		m.procState.ConnState.BSSID,
		m.procState.ConnState.WPAState,
		m.procState.ConnState.RSSI,
		m.procState.ConnState.STAInfo.TxBitrate/1000000,
		m.procState.ConnState.STAInfo.RxBitrate/1000000,
		m.procState.ConnState.STAInfo.ConnDuration,
	)
	return lipgloss.JoinVertical(lipgloss.Center, header, s)
}

func (m model) roamStatsView() string {
	header := m.allignCenter().
		Render(lipgloss.NewStyle().Bold(true).
			Render("Last Roam Stats"))
	s := fmt.Sprintf("target bssid: %v final bssid: %v\n"+
		"duration: %v message: %v",
		m.procState.RoamStats.TargetBSSID,
		m.procState.RoamStats.FinalBSSID,
		fmt.Sprintf("%.2fs", m.procState.RoamStats.Duration.Seconds()),
		m.procState.RoamStats.Message,
	)
	return lipgloss.JoinVertical(lipgloss.Center, header, s)
}

func (m model) apTableView() string {
	header := m.allignCenter().
		Render(lipgloss.NewStyle().Bold(true).
			Render("Last Scan Data"))
	return lipgloss.JoinVertical(lipgloss.Left, header, m.apTable.View())
}

func (m model) rssiBarView() string {
	var s string
	for _, r := range m.ringBuffer {
		matched := false
		for _, c := range m.rssiColors {
			if r <= c.threshold {
				s += lipgloss.NewStyle().Foreground(c.color).Render("█")
				matched = true
				break
			}
		}
		if !matched {
			s += lipgloss.NewStyle().Foreground(m.rssiColors[len(m.rssiColors)-1].color).Render("█")
		}
	}
	return s
}

func (m model) makeRows() []table.Row {
	var rows []table.Row
	var currSc int
	for _, b := range m.procState.BSSList {
		if b.IsCurrentAP {
			currSc = b.FinalScore
			break
		}
	}
	for _, b := range m.procState.BSSList {
		row := make([]string, 10)
		if b.IsCurrentAP {
			row[0] = currentAPStyle().Render(b.BSSID)
			row[1] = currentAPStyle().Render(strconv.Itoa(b.ChannelNum))
			row[2] = currentAPStyle().Render(strings.TrimSuffix(b.ChannelWidth, "MHz"))
			row[3] = currentAPStyle().Render(strings.TrimSuffix(b.Band, "Hz"))
			row[4] = currentAPStyle().Render(strconv.Itoa(b.RSSI))
			row[5] = currentAPStyle().Render(strconv.Itoa(b.SNR))
			if b.PHYType == "Legacy a/b/g" {
				b.PHYType = "abg"
			} else {
				b.PHYType = strings.TrimPrefix(b.PHYType, "802.11")
			}
			row[6] = currentAPStyle().Render(b.PHYType)
			row[7] = currentAPStyle().Render(strconv.Itoa(int(b.QBSSUtil)*100/255) + "%")
			row[8] = currentAPStyle().Render(strconv.Itoa(b.FinalScore))
			row[9] = currentAPStyle().Render("—")
		} else {
			row[0] = b.BSSID
			row[1] = strconv.Itoa(b.ChannelNum)
			row[2] = strings.TrimSuffix(b.ChannelWidth, "MHz")
			row[3] = strings.TrimSuffix(b.Band, "Hz")
			row[4] = strconv.Itoa(b.RSSI)
			row[5] = strconv.Itoa(b.SNR)
			if b.PHYType == "Legacy a/b/g" {
				b.PHYType = "abg"
			} else {
				b.PHYType = strings.TrimPrefix(b.PHYType, "802.11")
			}
			row[6] = b.PHYType
			row[7] = strconv.Itoa(int(b.QBSSUtil)*100/255) + "%"
			row[8] = strconv.Itoa(b.FinalScore)
			row[9] = strconv.Itoa(b.FinalScore - currSc)
		}
		rows = append(rows, row)
	}
	return rows
}

func (m model) statePanelView() string {
	firstRow := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.roamingTierContainer(),
		m.roamResultContainer(),
		m.scanModeContainer(),
		m.scanFlagsContainer())
	secondRow := lipgloss.JoinHorizontal(
		lipgloss.Center,
		m.unhealthyContainer(),
		m.bssStableContainer(),
		m.hysteresisContainer(),
		m.scanInProgContainer())
	return lipgloss.JoinVertical(lipgloss.Center, firstRow, secondRow)
}

func (m model) roamingTierContainer() string {
	txt := "Roaming Tier\n" + m.procState.RoamingTier
	switch m.procState.RoamingTier {
	case "unknown":
		return containerStyle(lipgloss.Color("#808080")).Render(txt)
	case "roam_disabled":
		return containerStyle(lipgloss.Color("#00ff00")).Render(txt)
	case "opportunistic":
		return containerStyle(lipgloss.Color("#ffee00")).Render(txt)
	case "active_roaming":
		return containerStyle(lipgloss.Color("#ff8800")).Render(txt)
	case "critical":
		return containerStyle(lipgloss.Color("#ff2200")).Render(txt)
	}
	return ""
}

func (m model) scanModeContainer() string {
	txt := "Scan Mode\n" + m.procState.ScanMode
	switch m.procState.ScanMode {
	case "scan_disabled":
		return containerStyle(lipgloss.Color("#808080")).Render(txt)
	case "fast_scan":
		return containerStyle(lipgloss.Color("#00aaff")).Render(txt)
	case "full_scan":
		return containerStyle(lipgloss.Color("#aa00ff")).Render(txt)
	}
	return ""
}

func (m model) roamResultContainer() string {
	txt := "Last Roam Result\n" + m.procState.RoamResultFlag
	switch m.procState.RoamResultFlag {
	case "unknown":
		return containerStyle(lipgloss.Color("#808080")).Render(txt)
	case "success":
		return containerStyle(lipgloss.Color("#00ff00")).Render(txt)
	case "failure":
		return containerStyle(lipgloss.Color("#ff2200")).Render(txt)
	case "no_candidates":
		return containerStyle(lipgloss.Color("#808080")).Render(txt)
	}
	return ""
}

func (m model) scanFlagsContainer() string {
	check := func(b bool) string {
		if b {
			return "✓"
		}
		return "✗"
	}
	txt := "Active Scans: " +
		"Entry:" + check(m.procState.EntryScanned) + "\n" +
		" CritEntry:" + check(m.procState.EntryScannedCrit) +
		" CritFull:" + check(m.procState.FullScannedCrit)
	switch {
	case m.procState.FullScannedCrit:
		return containerStyle(lipgloss.Color("#ff2200")).Render(txt)
	case m.procState.EntryScannedCrit:
		return containerStyle(lipgloss.Color("#ff8800")).Render(txt)
	case m.procState.EntryScanned:
		return containerStyle(lipgloss.Color("#ffee00")).Render(txt)
	default:
		return containerStyle(lipgloss.Color("#808080")).Render(txt)
	}
}

func (m model) unhealthyContainer() string {
	if m.procState.UnhealthyConn {
		return containerStyle(lipgloss.Color("#ff2200")).Render(
			"Connection unhealthy")
	}
	return containerStyle(lipgloss.Color("#00ff00")).Render(
		"Connection stable")
}

func (m model) hysteresisContainer() string {
	if m.procState.HysteresisActive {
		return containerStyle(lipgloss.Color("#ffee00")).Render(
			"Hysteresis active")
	}
	return containerStyle(lipgloss.Color("#808080")).Render(
		"Hysteresis inactive")
}

func (m model) scanInProgContainer() string {
	if m.procState.ScanInProgress {
		return containerStyle(lipgloss.Color("#00aaff")).Render(
			"Scan active")
	}
	return containerStyle(lipgloss.Color("#808080")).Render(
		"Scan idle")
}

func (m model) bssStableContainer() string {
	if m.procState.BSSListStable {
		return containerStyle(lipgloss.Color("#00ff00")).Render(
			"BSS list stable")
	}
	return containerStyle(lipgloss.Color("#ff8800")).Render(
		"BSS list changed")
}
