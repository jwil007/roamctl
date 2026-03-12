package wpac

//This file contains functions to run and process output from wpa_supplicant control interface commands
import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

func (c *Client) cmd(command string) ([]byte, error) {

	buf := make([]byte, 4096)

	_, wErr := c.CC.Write([]byte(command))
	if wErr != nil {
		return nil, fmt.Errorf("n.Write: %v", wErr)
	}

	out, err := c.CC.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("conn.Read: %v", err)
	}
	return buf[:out], nil
}

func (c *Client) getSSID() (string, error) {
	out, err := c.cmd("STATUS")
	if err != nil {
		return "", fmt.Errorf("c.Cmd(\"STATUS\"): %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "ssid=") {
			return line[5:], nil
		}
	}
	return "", fmt.Errorf("ssid field not found - check if wifi iface connected")
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

func (c *Client) runScan() error {
	out, err := c.cmd("SCAN")
	if err != nil {
		return fmt.Errorf("c.Cmd(\"SCAN\"): %w", err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf("c.Cmd(\"SCAN\"): %s", string(out))
	}
	return nil
}

func (c *Client) getScanResults(ssid string) ([]string, error) {
	var bssids []string
	out, err := c.cmd("SCAN_RESULTS")
	if err != nil {
		return nil, fmt.Errorf("c.Cmd(\"SCAN_RESULTS\"): %w", err)
	}
	for _, line := range strings.Split(string(out), "\n")[1:] {
		if strings.Contains(line, ssid) {
			bssid := strings.Fields(line)[0]
			bssids = append(bssids, bssid)
		}
	}
	return bssids, nil
}

func (c *Client) parseWpasBSS(bssid string) (WpasBSS, error) {
	out, err := c.cmd("BSS " + bssid)
	var b WpasBSS
	if err != nil {
		return WpasBSS{}, fmt.Errorf("c.Cmd(\"BSS\"): %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "bssid="):
			b.BSSID = line[6:]
		case strings.HasPrefix(line, "freq="):
			b.Freq, err = strconv.Atoi(line[5:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
		case strings.HasPrefix(line, "beacon_int="):
			b.BeaconInt, err = strconv.Atoi(line[11:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
		case strings.HasPrefix(line, "noise="):
			b.Noise, err = strconv.Atoi(line[6:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
		case strings.HasPrefix(line, "level="):
			b.RSSI, err = strconv.Atoi(line[6:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
		case strings.HasPrefix(line, "snr="):
			b.SNR, err = strconv.Atoi(line[4:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
		case strings.HasPrefix(line, "age="):
			b.Age, err = strconv.Atoi(line[4:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
		case strings.HasPrefix(line, "flags="):
			b.Flags = line[6:]
		case strings.HasPrefix(line, "est_throughput="):
			b.EstThruput, err = strconv.Atoi(line[15:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
		case strings.HasPrefix(line, "ie="):
			probeIE, err := hex.DecodeString(line[3:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("hex.DecodeString: %w", err)
			}
			b.ProbeIE = probeIE
		case strings.HasPrefix(line, "beacon_ie="):
			beaconIE, err := hex.DecodeString(line[10:])
			if err != nil {
				return WpasBSS{}, fmt.Errorf("hex.DecodeString: %w", err)
			}
			b.BeaconIE = beaconIE

		}
	}
	return b, nil
}

func (c *Client) getSignal() (Signal, error) {
	out, err := c.cmd("SIGNAL_POLL")
	if err != nil {
		return Signal{}, fmt.Errorf("c.Cmd(\"SIGNAL_POLL\") %w", err)
	}
	var s Signal
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "RSSI="):
			rssi, err := strconv.Atoi(line[5:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.RSSI = rssi
		case strings.HasPrefix(line, "LINKSPEED="):
			linkspeed, err := strconv.Atoi(line[10:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.LinkSpeed = linkspeed
		case strings.HasPrefix(line, "NOISE="):
			noise, err := strconv.Atoi(line[6:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.Noise = noise
		case strings.HasPrefix(line, "FREQUENCY="):
			freq, err := strconv.Atoi(line[10:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.Freq = freq
		case strings.HasPrefix(line, "WIDTH="):
			width := line[6:]
			s.ChannelWidth = width
		case strings.HasPrefix(line, "AVG_RSSI="):
			avgRSSI, err := strconv.Atoi(line[9:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.AvgRSSI = avgRSSI
		case strings.HasPrefix(line, "AVG_BEACON_RSSI="):
			avgRSSIbeacon, err := strconv.Atoi(line[16:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.AvgRSSIBeacon = avgRSSIbeacon
		}
	}
	return s, nil
}
