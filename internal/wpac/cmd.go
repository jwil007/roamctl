package wpac

//This file contains functions to run and process output from wpa_supplicant control interface commands
import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/jwil007/roamctl/internal/netlink"
)

func (c *Client) cmd(command string) ([]byte, error) {
	c.cmdMu.Lock()
	defer c.cmdMu.Unlock()
	err := c.CC.SetDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("could not set read deadline: %w", err)
	}
	defer func() {
		_ = c.CC.SetDeadline(time.Time{})
	}()
	buf := make([]byte, 65536)
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

func (c *Client) cmdP(command string) ([]byte, error) {
	err := c.PC.SetDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		return nil, fmt.Errorf("could not set read deadline: %w", err)
	}
	defer func() {
		_ = c.PC.SetDeadline(time.Time{})
	}()
	buf := make([]byte, 65536)
	_, wErr := c.PC.Write([]byte(command))
	if wErr != nil {
		return nil, fmt.Errorf("n.Write: %v", wErr)
	}
	out, err := c.PC.Read(buf)
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
		return fmt.Errorf(
			"c.cmd(ROAM %v): output not \"OK\": %v", bssid, string(out))
	}
	return nil
}

func (c *Client) getStatus() (Status, error) {
	var status Status
	out, err := c.cmdP("STATUS")
	if err != nil {
		return Status{}, fmt.Errorf("c.cmd(\"STATUS\"): %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		//fmt.Printf("DEBUG: status output: %v\n", line)
		if strings.HasPrefix(line, "ssid=") {
			status.SSID = line[5:]
		}
		if strings.HasPrefix(line, "wpa_state=") {
			status.WPAState = line[10:]
		}
	}
	return status, nil
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
		return "", fmt.Errorf(
			"c.Cmd(\"GET_NETWORK\""+networkID+"\" bgscan\"): %w", err)
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

func (c *Client) getBTM() (string, error) {
	out, err := c.cmd("GET " + "disable_btm")
	if err != nil {
		return "", fmt.Errorf("c.Cmd(\"GET disable_btm\"): %w", err)
	}
	return string(out), nil
}

func (c *Client) setBTM(config WPAConfig) error {
	out, err := c.cmd("SET " + "disable_btm " + config.DisableBTM)
	if err != nil {
		return fmt.Errorf(
			"c.Cmd(\"SET disable_btm %v\"): %w", config.DisableBTM, err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf(
			"c.Cmd(\"SET disable_btm %v\"): %v", config.DisableBTM, string(out))
	}
	return nil
}

func (c *Client) runScan(s ScanParams) error {
	var freqStr string
	var ssidStr string
	if s.Freqs != nil {
		slices.Sort(s.Freqs)
		var freqStrs []string
		for _, freq := range s.Freqs {
			freqStrs = append(freqStrs, strconv.Itoa(freq))
		}
		freqStr = " freq=" + strings.Join(freqStrs, ",")
	}
	if s.SSID != "" {
		ssidStr = " ssid=" + hex.EncodeToString([]byte(s.SSID))
	}
	out, err := c.cmd("SCAN TYPE=ONLY" + freqStr + ssidStr)
	if err != nil {
		return fmt.Errorf(
			"c.Cmd(SCAN TYPE=ONLY %v %v): %w", freqStr, ssidStr, err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf(
			"c.Cmd(SCAN TYPE=ONLY %v %v): %s", freqStr, ssidStr, string(out))
	}
	return nil
}

func (c *Client) runScanWithRetry(s ScanParams) error {
	maxRetries := s.RetryCount
	for range maxRetries {
		err := c.runScan(s)
		if err != nil {
			if strings.Contains(err.Error(), "FAIL-BUSY") {
				log.Println("interface busy, retrying scan in 2 seconds")
				time.Sleep(2 * time.Second)
				continue
			}
			return fmt.Errorf("wpac.runScan: %w", err)
		}
		return nil
	}
	return fmt.Errorf("wpac.runScanWithRetry: max retries exceeded")
}

func (c *Client) getScanResults(ssid string) ([]string, error) {
	var bssids []string
	out, err := c.cmd("SCAN_RESULTS")
	if err != nil {
		return nil, fmt.Errorf("c.Cmd(\"SCAN_RESULTS\"): %w", err)
	}
	//fmt.Printf("DEBUG - raw out of SCAN_RESULTS %v", string(out)) //debug
	for _, line := range strings.Split(string(out), "\n")[1:] {
		//fmt.Printf("DEBUG - line split of SCAN_RESULTS out: %v", line)
		parts := strings.SplitN(line, "\t", 5)
		//fmt.Printf("DEBUG - fields of SCAN_RESULTS line: %v", parts)
		if len(parts) == 5 && parts[4] == ssid {
			//fmt.Printf("DEBUG - BSSID from SCAN_RESULTS: %v", parts[0])
			bssids = append(bssids, parts[0])
		}
	}
	//fmt.Printf("DEBUG - BSSID list from SCAN_RESULTS: %v", bssids)
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
			ageInt, errA := strconv.Atoi(line[4:])
			if errA != nil {
				return WpasBSS{}, fmt.Errorf("strconv.Atoi: %w", errA)
			}
			b.Age = time.Duration(ageInt) * time.Second
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
	out, err := c.cmdP("SIGNAL_POLL")
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
		//case strings.HasPrefix(line, "WIDTH="):
		//	width := line[6:]
		//	s.ChannelWidth = width
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
	status, err := c.getStatus()
	if err != nil {
		return ConnectionStatus{}, fmt.Errorf("c.getStatus: %w", err)
	}
	signal, err := c.getSignal()
	if err != nil {
		return ConnectionStatus{}, fmt.Errorf("c.getSignal(): %w", err)
	}
	type staResult struct {
		info netlink.STAInfo
		err  error
	}
	ch := make(chan staResult, 1)
	go func() {
		info, err := netlink.GetStationInfo(c.Iface)
		ch <- staResult{info, err}
	}()

	var staInfo netlink.STAInfo
	select {
	case res := <-ch:
		if res.err != nil {
			// non-fatal, continue with zero STAInfo
			slog.Debug("GetStationInfo failed", "err", res.err)
		} else {
			staInfo = res.info
		}
	case <-time.After(5 * time.Second):
		slog.Warn("GetStationInfo timed out")
	}

	return ConnectionStatus{
		Status:  status,
		Signal:  signal,
		STAInfo: staInfo,
	}, nil
}

func (c *Client) listenEvents(
	ctx context.Context) (<-chan string, <-chan error) {
	events := make(chan string)
	errc := make(chan error, 1)
	go func() {
		_, err := c.EC.Write([]byte("ATTACH"))
		if err != nil {
			errc <- err
			return
		}
		buf := make([]byte, 65536)
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
			case events <- string(buf[:n]):
			}
		}
	}()
	return events, errc
}

func (c *Client) waitForEvent(
	ctx context.Context,
	match []string,
	timeout time.Duration) (string, error) {
	listenCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	events, errc := c.listenEvents(listenCtx)
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
