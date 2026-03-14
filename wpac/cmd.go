package wpac

//This file contains functions to run and process output from wpa_supplicant control interface commands
import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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

func (c *Client) runRoam(bssid string) error {
	out, err := c.cmd("ROAM " + bssid)
	if err != nil {
		return fmt.Errorf("c.cmd(ROAM %v): %w", bssid, err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf("c.cmd(ROAM %v): output not \"OK\": %v", bssid, string(out))
	}
	return nil
}

func (c *Client) getSSID() (string, string, error) {
	out, err := c.cmd("STATUS")
	if err != nil {
		return "", "", fmt.Errorf("c.cmd(\"STATUS\"): %w", err)
	}
	var bssid string
	var ssid string
	for _, line := range strings.Split(string(out), "\n") {
		fmt.Println(line) //debug print
		switch {
		case strings.HasPrefix(line, "bssid="):
			bssid = line[6:]
		case strings.HasPrefix(line, "ssid="):
			ssid = line[5:]
		}
	}
	if ssid == "" || bssid == "" {
		return "", "", fmt.Errorf("ssid or bssid field not found - check if wifi iface connected")
	}
	return ssid, bssid, nil
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
func (c *Client) constructConnStatus() (ConnectionStatus, error) {
	s, err := c.getSignal()
	if err != nil {
		return ConnectionStatus{}, fmt.Errorf("c.getSignal(): %w", err)
	}
	ssid, bssid, err := c.getSSID()
	if err != nil {
		return ConnectionStatus{}, fmt.Errorf("c.getSSID(): %w", err)
	}
	return ConnectionStatus{
		Signal: s,
		SSID:   ssid,
		BSSID:  bssid,
	}, nil
}

func (c *Client) listenEvents(ctx context.Context) (<-chan string, <-chan error) {
	events := make(chan string)
	errc := make(chan error, 1)
	go func() {
		_, err := c.EC.Write([]byte("ATTACH"))
		if err != nil {
			errc <- err
		}
		buf := make([]byte, 4096)
		for {
			errDeadline := c.EC.SetReadDeadline(time.Now().Add(1 * time.Second))
			if errDeadline != nil {
				errc <- errDeadline
				return
			}
			n, err := c.EC.Read(buf)
			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					continue
				}
				errc <- err
				return
			}
			//fmt.Printf("[event] %q\n", string(buf[:n])) //debug print
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

func (c *Client) waitForEvent(ctx context.Context, match []string, timeout time.Duration) (string, error) {
	events, errc := c.listenEvents(ctx)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timer.C:
			return "", fmt.Errorf("timed out waiting for event")
		case event := <-events:
			for _, s := range match { //return on first event matching
				if strings.Contains(event, s) {
					return event, nil
				}
			}
		case err := <-errc:
			return "", err
		}
	}
}
