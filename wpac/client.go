package wpac

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	Conn      *net.UnixConn
	Iface     string
	LocalPath string
}

func Connect(iface string) (*Client, error) {
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
		return &Client{}, fmt.Errorf("net.DialUnix: %w", err)
	}

	return &Client{
		Conn:      conn,
		Iface:     iface,
		LocalPath: localPath,
	}, nil
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

func (c *Client) ListenEvents(ctx context.Context) (<-chan string, <-chan error) {
	events := make(chan string)
	errc := make(chan error, 1)
	go func() {
		_, err := c.cmd("ATTACH")
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
				if errors.Is(err, os.ErrDeadlineExceeded) {
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
	}()
	return events, errc
}

func (c *Client) WaitForEvent(ctx context.Context, match string, timeout time.Duration) error {
	events, errc := c.ListenEvents(ctx)
	errw := make(chan error, 1)
	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				errw <- fmt.Errorf("timed out waiting for event")
				return
			case event := <-events:
				if strings.Contains(event, match) {
					errw <- nil
					return
				}
			case err := <-errc:
				errw <- err
				return
			}
		}
	}()
	return <-errw
}

func (c *Client) cmd(command string) ([]byte, error) {

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
