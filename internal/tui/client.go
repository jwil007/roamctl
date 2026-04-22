package tui

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"
)

func connect(iface *string) (*client, error) {
	localPath := "/tmp/roamctl-tui_" + strconv.Itoa(os.Getpid())
	remotePath := "/run/roamctl/" + *iface + ".sock"
	laddr := &net.UnixAddr{Name: localPath, Net: "unix"}
	raddr := &net.UnixAddr{Name: remotePath, Net: "unix"}
	_ = os.Remove(localPath)
	c, err := net.DialUnix("unix", laddr, raddr)
	if err != nil {
		return nil, fmt.Errorf("net.DialUnix: %w", err)
	}
	return &client{
		conn:      c,
		localPath: localPath,
	}, nil
}

func (c *client) close() {
	if c == nil {
		return
	}
	_ = c.conn.Close()
	_ = os.Remove(c.localPath)
}

func readCmd(scanner *bufio.Scanner) tea.Cmd {
	return func() tea.Msg {
		if scanner.Scan() {
			return socketMsg(scanner.Text())
		}
		return reconnectMsg(true)
	}
}

func reconnectCmd(iface *string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		c, err := connect(iface)
		if err != nil {
			return reconnectMsg(true)
		}
		return clientMsg(c)
	}
}
