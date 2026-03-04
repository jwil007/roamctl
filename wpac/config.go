// Package wpac: wpa_supplicant commands
package wpac

import (
	"fmt"
	"strings"

	"github.com/jwil007/roamctl/wpas"
)

func GetConfig(c *wpas.Client) (WPAConfig, error) {
	ssid, err := getSSID(c)
	if err != nil {
		return WPAConfig{}, fmt.Errorf("getSSID: %w", err)
	}
	networkID, err := getNetworkID(c)
	if err != nil {
		return WPAConfig{}, fmt.Errorf("getNetworkID: %w", err)
	}
	bgscan, err := getBGScan(c, networkID)
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

func getSSID(c *wpas.Client) (string, error) {
	out, err := c.Cmd("STATUS")
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

func getNetworkID(c *wpas.Client) (string, error) {
	out, err := c.Cmd("LIST_NETWORKS")
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

func getBGScan(c *wpas.Client, networkID string) (string, error) {
	out, err := c.Cmd("GET_NETWORK " + networkID + " bgscan")
	if err != nil {
		return "", fmt.Errorf("c.Cmd(\"GET_NETWORK\""+networkID+"\" bgscan\"): %w", err)
	}
	return string(out), nil
}

func SetConfig(c *wpas.Client, config WPAConfig) error {
	err := setBGScan(c, config)
	if err != nil {
		return fmt.Errorf("setBGScan: %w", err)
	}
	return nil
}

func setBGScan(c *wpas.Client, config WPAConfig) error {
	s := "SET_NETWORK " + config.NetworkID + " bgscan " + config.BGScan
	out, err := c.Cmd(s)
	if err != nil {
		return fmt.Errorf("c.Cmd(%s): %w", s, err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf("c.Cmd(%s): %s", s, string(out))
	}
	return nil
}
