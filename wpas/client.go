// Package wpas: UnixSocket interface to wpa_supplicant
package wpas

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

type Client struct {
	Conn      *net.UnixConn
	Iface     string
	LocalPath string
}

func Connect(iface string) (Client, error) {
	remotePath := "/var/run/wpa_supplicant/" + iface
	localPath := "/tmp/wpa_ctrl_" + strconv.Itoa(os.Getpid())

	laddr := &net.UnixAddr{
		Name: localPath,
		Net:  "unixgram",
	}
	raddr := &net.UnixAddr{
		Name: remotePath,
		Net:  "unixgram",
	}

	conn, err := net.DialUnix("unixgram", laddr, raddr)
	if err != nil {
		return Client{}, fmt.Errorf("net.DialUnix: %w", err)
	}

	return Client{
		Conn:      conn,
		Iface:     iface,
		LocalPath: localPath,
	}, nil
}

func (c *Client) Cmd(command string) ([]byte, error) {

	buf := make([]byte, 4096)

	_, wErr := c.Conn.Write([]byte(command))
	if wErr != nil {
		return nil, fmt.Errorf("n.Write: %v", wErr)
	}

	out, err := c.Conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("conn.Read: %v", err)
	}
	return buf[:out], nil
}

func (c *Client) Close() error {
	errClose := c.Conn.Close()
	if errClose != nil {
		return fmt.Errorf("c.Conn.Close: %w", errClose)
	}
	errRemove := os.Remove(c.LocalPath)
	if errRemove != nil {
		return fmt.Errorf("os.Remove %v: %w", c.LocalPath, errRemove)
	}
	return nil
}
