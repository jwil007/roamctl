package tui

import (
	"bufio"
	"image/color"
	"net"

	"charm.land/bubbles/v2/table"
	"github.com/jwil007/roamctl/internal/ipc"
)

type client struct {
	conn      *net.UnixConn
	localPath string
}

type model struct {
	client     *client
	scanner    *bufio.Scanner
	procState  *ipc.ProcessState
	ringBuffer []int
	width      int
	height     int
	apTable    table.Model
	rssiColors []colorStop
}

type colorStop struct {
	threshold int
	color     color.Color
}

type socketMsg string
type reconnectMsg bool

type clientMsg *client

var apTableColumns = []table.Column{
	{Title: "BSSID", Width: 19},
	{Title: "Ch", Width: 4},
	{Title: "CW", Width: 5},
	{Title: "Band", Width: 4},
	{Title: "RSSI", Width: 5},
	{Title: "SNR", Width: 4},
	{Title: "PHY", Width: 4},
	{Title: "Util", Width: 5},
	{Title: "Scr", Width: 4},
	{Title: "ΔScr", Width: 4},
}
