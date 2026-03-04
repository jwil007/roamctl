// Package wpas: UnixSocket interface to wpa_supplicant
package wpas

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	Conn      *net.UnixConn
	Iface     string
	LocalPath string
}

func Connect(iface string) (Client, error) {
	myUUID := uuid.New()
	remotePath := "/var/run/wpa_supplicant/" + iface
	localPath := "/tmp/wpa_ctrl_" + myUUID.String()

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

func (c *Client) ListenEvents(ctx context.Context, events chan string, errc chan error) {
	_, err := c.Cmd("ATTACH")
	if err != nil {
		errc <- err
	}

	buf := make([]byte, 4096)

	for {
		errDeadline := c.Conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		if errDeadline != nil {
			errc <- errDeadline
			return
		}
		n, err := c.Conn.Read(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			errc <- err
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
			events <- string(buf[:n])
		}
	}
}
