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
	localPathPoll := "/tmp/wpa_ctrl_" + strconv.Itoa(os.Getpid()) + "poll"

	laddrC := &net.UnixAddr{
		Name: localPathCmd,
		Net:  "unixgram",
	}
	laddrE := &net.UnixAddr{
		Name: localPathEvent,
		Net:  "unixgram",
	}
	laddrP := &net.UnixAddr{
		Name: localPathPoll,
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
		if err = cc.Close(); err != nil {
			return nil, fmt.Errorf("cc.Close: %w", err)
		}
		return nil, fmt.Errorf("net.DialUnix: %w", err)
	}
	pc, err := net.DialUnix("unixgram", laddrP, raddr)
	if err != nil {
		if err = ec.Close(); err != nil {
			if err = cc.Close(); err != nil {
				return nil, fmt.Errorf("cc.Close: %w", err)
			}
			return nil, fmt.Errorf("cc.Close: %w", err)
		}
		if err = cc.Close(); err != nil {
			return nil, fmt.Errorf("cc.Close: %w", err)
		}
		return nil, fmt.Errorf("net.DialUnix: %w", err)
	}
	return &Client{
		CC:             cc,
		EC:             ec,
		PC:             pc,
		Iface:          iface,
		LocalPathCmd:   localPathCmd,
		LocalPathEvent: localPathEvent,
		LocalPathPoll:  localPathPoll,
	}, nil
}

func (c *Client) Close() error {
	err := c.CC.Close()
	if err != nil {
		return fmt.Errorf("c.CC.Close: %w", err)
	}
	err = c.EC.Close()
	if err != nil {
		return fmt.Errorf("c.EC.Close: %w", err)
	}
	err = c.PC.Close()
	if err != nil {
		return fmt.Errorf("c.PC.Close: %w", err)
	}
	err = os.Remove(c.LocalPathCmd)
	if err != nil {
		return fmt.Errorf("os.Remove %v: %w", c.LocalPathCmd, err)
	}
	err = os.Remove(c.LocalPathEvent)
	if err != nil {
		return fmt.Errorf("os.Remove %v: %w", c.LocalPathEvent, err)
	}
	err = os.Remove(c.LocalPathPoll)
	if err != nil {
		return fmt.Errorf("os.Remove %v: %w", c.LocalPathPoll, err)
	}
	return nil
}
