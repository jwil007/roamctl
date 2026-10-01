package tui

import (
	"bufio"
	"fmt"
	"net"
	"time"

	tea "charm.land/bubbletea/v2"
)

func connect(iface *string) (*client, error) {
	remotePath := "/run/roamctl/" + *iface + ".sock"
	raddr := &net.UnixAddr{Name: remotePath, Net: "unix"}
	c, err := net.DialUnix("unix", nil, raddr)
	if err != nil {
		return nil, fmt.Errorf("net.DialUnix: %w", err)
	}
	return &client{
		conn: c,
	}, nil
}

func (c *client) close() {
	if c == nil {
		return
	}
	_ = c.conn.Close()
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
