package wpac

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

func Connect(iface string) (*Client, error) {
	remotePath := "/var/run/wpa_supplicant/" + iface
	localPathCmd := "/tmp/wpa_ctrl_" + strconv.Itoa(os.Getpid()) + "command"
	localPathEvent := "/tmp/wpa_ctrl_" + strconv.Itoa(os.Getpid()) + "event"

	laddrC := &net.UnixAddr{
		Name: localPathCmd,
		Net:  "unixgram",
	}
	laddrE := &net.UnixAddr{
		Name: localPathEvent,
		Net:  "unixgram",
	}
	raddr := &net.UnixAddr{
		Name: remotePath,
		Net:  "unixgram",
	}

	cc, err := net.DialUnix("unixgram", laddrC, raddr)
	if err != nil {
		return nil, fmt.Errorf("net.DialUnix: %w", err)
	}

	ec, err := net.DialUnix("unixgram", laddrE, raddr)
	if err != nil {
		errClose := cc.Close()
		if errClose != nil {
			return nil, fmt.Errorf("cc.Close: %w", err)
		}
		return nil, fmt.Errorf("net.DialUnix: %w", err)
	}

	return &Client{
		CC:             cc,
		EC:             ec,
		Iface:          iface,
		LocalPathCmd:   localPathCmd,
		LocalPathEvent: localPathEvent,
	}, nil
}

func (c *Client) Close() error {
	errCloseC := c.CC.Close()
	if errCloseC != nil {
		return fmt.Errorf("c.CC.Close: %w", errCloseC)
	}
	errCloseE := c.EC.Close()
	if errCloseE != nil {
		return fmt.Errorf("c.EC.Close: %w", errCloseE)
	}
	errRemoveC := os.Remove(c.LocalPathCmd)
	if errRemoveC != nil {
		return fmt.Errorf("os.Remove %v: %w", c.LocalPathCmd, errRemoveC)
	}
	errRemoveE := os.Remove(c.LocalPathEvent)
	if errRemoveE != nil {
		return fmt.Errorf("os.Remove %v: %w", c.LocalPathEvent, errRemoveE)
	}
	return nil
}
