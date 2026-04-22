package tui

import (
	"bufio"
	"image/color"
	"net"
	"time"

	"charm.land/bubbles/v2/table"
	"github.com/jwil007/roamctl/internal/ipc"
)

type client struct {
	conn      *net.UnixConn
	localPath string
}

type model struct {
	iface      *string
	client     *client
	scanner    *bufio.Scanner
	procState  *ipc.ProcessState
	ringBuffer []int
	width      int
	height     int
	apTable    table.Model
	roamTable  table.Model
	roamLogs   []roamLog
	rssiColors []colorStop
	lastRoam   roamLog
	lastBSSID  string
}

type roamLog struct {
	status      string
	fromBSSID   string
	targetBSSID string
	finalBSSID  string
	fromChan    int
	toChan      int
	fromBand    string
	toBand      string
	trigRSSI    int
	finalRSSI   int
	scoreDelta  int
	duration    time.Duration
	completedAt time.Time
	message     string
}

type colorStop struct {
	threshold int
	color     color.Color
}

type socketMsg string
type reconnectMsg bool

type clientMsg *client

var apTableColumns = []table.Column{
	{Title: "  BSSID", Width: 21},
	{Title: "Ch", Width: 4},
	{Title: "CW", Width: 5},
	{Title: "Band", Width: 4},
	{Title: "RSSI", Width: 5},
	{Title: "SNR", Width: 4},
	{Title: "PHY", Width: 4},
	{Title: "Util", Width: 5},
	{Title: "Scr", Width: 4},
	{Title: "ScrΔ", Width: 4},
}

var roamTableColumns = []table.Column{
	{Title: "    Time", Width: 14},
	{Title: "Status", Width: 9},
	{Title: "From BSSID", Width: 19},
	{Title: "To BSSID", Width: 19},
	{Title: "Duration", Width: 8},
}

const tuiWidth = 82
