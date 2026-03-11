// Package wpac: APIs to establish unix socket to wpa_supplicant control interface and issue commands
package wpac

import (
	"fmt"
	"strings"
)

func (c *Client) GetConfig() (WPAConfig, error) {
	ssid, err := c.getSSID()
	if err != nil {
		return WPAConfig{}, fmt.Errorf("getSSID: %w", err)
	}
	networkID, err := c.getNetworkID()
	if err != nil {
		return WPAConfig{}, fmt.Errorf("getNetworkID: %w", err)
	}
	bgscan, err := c.getBGScan(networkID)
	if err != nil {
		return WPAConfig{}, fmt.Errorf("getBGScan: %w", err)
	}
	return WPAConfig{
		SSID:      ssid,
		NetworkID: networkID,
		BGScan:    bgscan,
		Iface:     c.Iface,
	}, nil
}

func (c *Client) SetConfig(config WPAConfig) error {
	err := c.setBGScan(config)
	if err != nil {
		return fmt.Errorf("setBGScan: %w", err)
	}
	return nil
}

func (c *Client) getSSID() (string, error) {
	out, err := c.cmd("STATUS")
	if err != nil {
		return "", fmt.Errorf("c.Cmd(\"STATUS\"): %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if line[:5] == "ssid=" {
			return line[5:], nil
		}
		//return "", fmt.Errorf("error parsing ssid, not connected maybe")
	}
	return "", fmt.Errorf("ssid field not found")
}

func (c *Client) getNetworkID() (string, error) {
	out, err := c.cmd("LIST_NETWORKS")
	if err != nil {
		return "", fmt.Errorf("c.Cmd(\"LIST_NETWORKS\"): %w", err)
	}
	for _, line := range strings.Split(string(out), "\n")[1:] {
		if strings.Contains(line, "[CURRENT]") {
			return strings.Fields(line)[0], nil
		}
	}
	return "", fmt.Errorf("no connected ssid")
}

func (c *Client) getBGScan(networkID string) (string, error) {
	out, err := c.cmd("GET_NETWORK " + networkID + " bgscan")
	if err != nil {
		return "", fmt.Errorf("c.Cmd(\"GET_NETWORK\""+networkID+"\" bgscan\"): %w", err)
	}
	return string(out), nil
}

func (c *Client) setBGScan(config WPAConfig) error {
	s := "SET_NETWORK " + config.NetworkID + " bgscan " + config.BGScan
	out, err := c.cmd(s)
	if err != nil {
		return fmt.Errorf("c.Cmd(%s): %w", s, err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf("c.Cmd(%s): %s", s, string(out))
	}
	return nil
}
