package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"github.com/jwil007/roamctl/internal/ipc"
)

func (m model) viewDashboard() string {
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		m.borderStyle().Render(m.titleView()),
		m.borderStyle().Render(m.rssiBarView()),
		m.borderStyle().Render(m.connStatView()),
		m.borderStyle().Render(m.statePanelView()),
		m.borderStyle().Render(m.roamTableView()),
		m.borderStyle().Render(m.apTableView()))
	return lipgloss.PlaceHorizontal(m.width, lipgloss.Center, content)
}

func (m model) titleView() string {
	left := lipgloss.NewStyle().Bold(true).Render(" roamctl-tui")
	center := time.Now().Format("15:04:05")
	right := "q to quit"

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	centerWidth := tuiWidth - leftWidth - rightWidth

	return titleStyle.Render(left) +
		lipgloss.NewStyle().Width(centerWidth).
			Align(lipgloss.Center).Faint(true).Render(center) +
		titleStyle.Render(right)
}

func (m model) connStatView() string {
	if m.procState.SSID == "" {
		return m.alignCenter().
			Render("Waiting for data from roamctl... \n" +
				"roamctl-tui may require root. Run with: sudo roamctl-tui\n" +
				"Also, ensure roamctl is running. " +
				"Run: sudo systemctl status roamctl")
	}
	txPHY := m.procState.ConnState.TxPHY
	rxPHY := m.procState.ConnState.RxPHY
	if m.procState.ConnState.TxPHY == "" {
		txPHY = "Lgcy"
	}
	if m.procState.ConnState.RxPHY == "" {
		rxPHY = "Lgcy"
	}
	header := m.alignLeft().
		Render(titleStyle.Render(" CONNECTION STATUS"))
	band, channel := getBandandChanfromFreq(m.procState.ConnState.Freq)
	l := titleStyle.Render

	line1 := l("ssid: ") + fmt.Sprintf("%-15v", m.procState.SSID) +
		l(" bssid: ") + fmt.Sprintf("%-19v", m.procState.ConnState.BSSID) +
		l("Duration: ") + fmt.Sprintf("%-10v", m.procState.ConnState.ConnDuration)

	line2 := l("rssi: ") + fmt.Sprintf("%-5v", m.procState.ConnState.RSSI) +
		l("channel: ") + fmt.Sprintf("%-5v", channel) +
		l("width: ") + fmt.Sprintf("%-7v", m.procState.ConnState.ChannelWidth) +
		l("band: ") + fmt.Sprintf("%-6v", band)

	line3 := l("tx: ") + fmt.Sprintf(
		"%-4v", m.procState.ConnState.TxBitrate/1000000) + l("Mbps  ") +
		l("PHY: ") + fmt.Sprintf("%-4v", txPHY) +
		l(" MCS: ") + fmt.Sprintf("%-2v", m.procState.ConnState.TxMCS) +
		l(" | rx: ") + fmt.Sprintf(
		"%-4v", m.procState.ConnState.RxBitrate/1000000) + l("Mbps  ") +
		l("PHY: ") + fmt.Sprintf("%-4v", rxPHY) +
		l(" MCS: ") + fmt.Sprintf("%-2v", m.procState.ConnState.RxMCS)

	s := lipgloss.JoinVertical(lipgloss.Left, line1, line2, line3)
	return lipgloss.JoinVertical(lipgloss.Center, header, s)
}

func (m model) apTableView() string {
	l := titleStyle.Render
	header := m.alignLeft().Render(l(" SCAN DATA"))
	footer := l("* current AP")
	content := m.apTable.View()
	detail := l(" Scan time: ") +
		m.procState.LastScanTime.Format("15:04:05")

	bottom := lipgloss.NewStyle().Width(tuiWidth).Render(
		detail + lipgloss.NewStyle().Width(tuiWidth-lipgloss.Width(detail)-
			lipgloss.Width(footer)).Render("") +
			lipgloss.NewStyle().Faint(true).Render(footer))
	return lipgloss.JoinVertical(lipgloss.Left, m.alignCenter().
		Render(header), content, bottom)
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
			s += lipgloss.NewStyle().
				Foreground(m.rssiColors[len(m.rssiColors)-1].color).
				Render("█")
		}
	}
	return s
}

func (m model) roamTableView() string {
	header := m.alignLeft().Render(titleStyle.Render(" ROAM RESULTS"))
	content := m.roamTable.View()
	footer := "↑↓ to select"
	info := ""
	detail := ""
	if len(m.roamLogs) == 0 {
		detail = titleStyle.Render(" Waiting for roam events...")
	}
	if len(m.roamLogs) > 0 && m.roamTable.
		Cursor() >= 0 && m.roamTable.Cursor() < len(m.roamLogs) {
		selected := m.roamLogs[m.roamTable.Cursor()]
		info = fmt.Sprintf(" Msg: %v",
			selected.message)
		if selected.message == "" {
			info = ""
		}
		detail = fmt.Sprintf(" Score Δ: %v | From: Ch%v %v %vdBm | "+
			"To: Ch%v %v %vdBm",
			selected.scoreDelta,
			selected.fromChan,
			selected.fromBand,
			selected.trigRSSI,
			selected.toChan,
			selected.toBand,
			selected.finalRSSI)
	}
	bottom := lipgloss.NewStyle().Width(tuiWidth).Render(
		detail + lipgloss.NewStyle().Width(tuiWidth-lipgloss.Width(detail)-
			lipgloss.Width(footer)).Render("") +
			lipgloss.NewStyle().Faint(true).Render(footer))
	return lipgloss.JoinVertical(lipgloss.Left, m.alignCenter().
		Render(header), content, info, bottom)
}

func (m model) isNewRoam() bool {
	if m.lastRoam.targetBSSID == m.procState.RoamStats.TargetBSSID &&
		m.lastRoam.status == m.procState.RoamResultFlag &&
		m.lastRoam.completedAt == m.procState.RoamStats.CompletedAt {
		return false
	}
	if m.procState.RoamResultFlag == "success" ||
		m.procState.RoamResultFlag == "failure" {
		return true
	}
	return false
}

func (m model) logRoam() roamLog {
	bssidMap := make(map[string]ipc.BSS)
	var fromBSS ipc.BSS
	var toBSS ipc.BSS
	for _, b := range m.procState.BSSList {
		bssidMap[b.BSSID] = b
	}
	fromBSS = bssidMap[m.lastBSSID]
	toBSS = bssidMap[m.procState.RoamStats.FinalBSSID]
	return roamLog{
		status:      m.procState.RoamResultFlag,
		fromBSSID:   m.lastBSSID,
		targetBSSID: m.procState.RoamStats.TargetBSSID,
		finalBSSID:  m.procState.RoamStats.FinalBSSID,
		fromChan:    fromBSS.ChannelNum,
		toChan:      toBSS.ChannelNum,
		fromBand:    fromBSS.Band,
		toBand:      toBSS.Band,
		trigRSSI:    m.procState.LastTriggerRSSI,
		finalRSSI:   toBSS.RSSI,
		scoreDelta:  toBSS.FinalScore - fromBSS.FinalScore,
		duration:    m.procState.RoamStats.Duration,
		completedAt: m.procState.RoamStats.CompletedAt,
		message:     m.procState.RoamStats.Message,
	}
}

func (m model) makeRoamTableRows() []table.Row {
	var rows []table.Row
	for i, log := range m.roamLogs {
		row := make(table.Row, 5)
		if i == m.roamTable.Cursor() {
			row[0] = "[*] " + log.completedAt.Format("15:04:05")
		} else {
			row[0] = "[ ] " + log.completedAt.Format("15:04:05")
		}
		switch log.status {
		case "success":
			row[1] = greenText().Render("Success")
		case "failure":
			row[1] = redText().Render("Failure")
		}
		if log.fromBSSID == "" {
			row[2] = "—"
		} else {
			row[2] = log.fromBSSID
		}
		row[3] = log.targetBSSID
		row[4] = fmt.Sprintf("%.3fs", log.duration.Seconds())
		rows = append(rows, row)
	}
	return rows
}

func (m model) makeAPRows() ([]table.Row, string) {
	var rows []table.Row
	var currSc int
	var lastBSSID string
	for _, b := range m.procState.BSSList {
		if b.IsCurrentAP {
			currSc = b.FinalScore
			break
		}
	}
	for _, b := range m.procState.BSSList {
		row := make([]string, 10)
		if b.IsCurrentAP {
			lastBSSID = b.BSSID
			row[0] = currentAPStyle().Render("* " + b.BSSID)
			row[1] = currentAPStyle().Render(strconv.Itoa(b.ChannelNum))
			row[2] = currentAPStyle().
				Render(strings.TrimSuffix(b.ChannelWidth, "MHz"))
			row[3] = currentAPStyle().
				Render(strings.TrimSuffix(b.Band, "Hz"))
			row[4] = currentAPStyle().Render(strconv.Itoa(b.RSSI))
			row[5] = currentAPStyle().Render(strconv.Itoa(b.SNR))
			if b.PHYType == "Legacy a/b/g" {
				b.PHYType = "abg"
			} else {
				b.PHYType = strings.TrimPrefix(b.PHYType, "802.11")
			}
			row[6] = currentAPStyle().Render(b.PHYType)
			row[7] = currentAPStyle().
				Render(strconv.Itoa(int(b.QBSSUtil)*100/255) + "%")
			row[8] = currentAPStyle().Render(strconv.Itoa(b.FinalScore))
			row[9] = currentAPStyle().Render("—")
		} else {
			row[0] = "  " + b.BSSID
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
	return rows, lastBSSID
}

func (m model) statePanelView() string {
	firstRow := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.roamingTierContainer(),
		m.roamStateContainer(),
		m.roamResultContainer(),
		m.scanModeContainer(),
		m.scanFlagsContainer())
	secondRow := lipgloss.JoinHorizontal(
		lipgloss.Left,
		m.unhealthyContainer(),
		m.bssStableContainer(),
		m.hysteresisContainer(),
		m.scanInProgContainer())
	return m.alignCenter().
		Render(lipgloss.JoinVertical(lipgloss.Center, firstRow, secondRow))
}

func (m model) roamingTierContainer() string {
	switch m.procState.RoamingTier {
	case "unknown":
		return containerStyle(lipgloss.Color("#808080")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Roaming Tier"), "Unknown"))
	case "roam_disabled":
		return containerStyle(lipgloss.Color("#808080")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Roaming Tier"), "Disabled"))
	case "opportunistic":
		return containerStyle(lipgloss.Color("#00ff00")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Roaming Tier"), "Opportunistic"))
	case "active_roaming":
		return containerStyle(lipgloss.Color("#ffee00")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Roaming Tier"), "Active"))
	case "critical":
		return containerStyle(lipgloss.Color("#ff2200")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Roaming Tier"), "Critical"))
	}
	return ""
}

func (m model) scanModeContainer() string {
	switch m.procState.ScanMode {
	case "scan_disabled":
		return containerStyle(lipgloss.Color("#808080")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Scan Mode"), "Disabled"))
	case "fast_scan":
		return containerStyle(lipgloss.Color("#00aaff")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Scan Mode"), "Fast"))
	case "full_scan":
		return containerStyle(lipgloss.Color("#aa00ff")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Scan Mode"), "Full"))
	}
	return ""
}

func (m model) roamResultContainer() string {
	title := "Roam Result"
	switch m.procState.RoamResultFlag {
	case "unknown":
		return containerStyle(lipgloss.Color("#808080")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				title), "Unknown"))
	case "success":
		return containerStyle(lipgloss.Color("#00ff00")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				title), "Success"))
	case "failure":
		return containerStyle(lipgloss.Color("#ff2200")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				title), "Failure"))
	case "no_candidates":
		return containerStyle(lipgloss.Color("#808080")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				title), "No candidates"))
	}
	return ""
}

func (m model) roamStateContainer() string {
	if m.procState.RoamInProgress {
		return containerStyle(lipgloss.Color("#00aaff")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Roam State"), "In progress"))
	}
	return containerStyle(lipgloss.Color("#808080")).Render(
		lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
			"Roam State"), "Idle"))
}

func (m model) scanFlagsContainer() string {
	check := func(b bool) string {
		if b {
			return "✓ "
		}
		return "_ "
	}
	markers := check(m.procState.EntryScanned) +
		check(m.procState.EntryScannedCrit) +
		check(m.procState.FullScannedCrit)

	switch {
	case m.procState.FullScannedCrit:
		return containerStyle(lipgloss.Color("#ff2200")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Active Scans"), markers))
	case m.procState.EntryScannedCrit:
		return containerStyle(lipgloss.Color("#ff8800")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Active Scans"), markers))
	case m.procState.EntryScanned:
		return containerStyle(lipgloss.Color("#ffee00")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Active Scans"), markers))
	default:
		return containerStyle(lipgloss.Color("#808080")).Render(
			lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render(
				"Active Scans"), markers))
	}
}

func (m model) unhealthyContainer() string {
	if m.procState.UnhealthyConn {
		return containerStyle(lipgloss.Color("#ff2200")).Render(
			"Connection unhealthy")
	}
	return containerStyle(lipgloss.Color("#808080")).Render(
		"Connection stable")
}

func (m model) hysteresisContainer() string {
	if m.procState.HysteresisActive {
		return containerStyle(lipgloss.Color("#ffee00")).Render(
			"Hysteresis on")
	}
	return containerStyle(lipgloss.Color("#808080")).Render(
		"Hysteresis off")
}

func (m model) scanInProgContainer() string {
	if m.procState.ScanInProgress {
		return containerStyle(lipgloss.Color("#00aaff")).Render(
			"Scan running")
	}
	return containerStyle(lipgloss.Color("#808080")).Render(
		"Scan idle")
}

func (m model) bssStableContainer() string {
	if m.procState.BSSListStable {
		return containerStyle(lipgloss.Color("#808080")).Render(
			"BSS list stable")
	}
	return containerStyle(lipgloss.Color("#ff8800")).Render(
		"BSS list changed")
}

func getBandandChanfromFreq(freq int) (string, int) {
	var channel int
	switch {
	case freq == 2484:
		channel = 14
		return "2.4GHz", channel
	case freq >= 2412 && freq <= 2472:
		channel = (freq - 2407) / 5
		return "2.4GHz", channel
	case freq >= 5180 && freq <= 5825:
		channel = (freq - 5000) / 5
		return "5GHz", channel
	case freq >= 5955 && freq <= 7115:
		channel = (freq - 5950) / 5
		return "6GHz", channel
	}
	return "unknown", 0
}
