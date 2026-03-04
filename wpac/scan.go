package wpac

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jwil007/roamctl/wpas"
)

func RunScan(c *wpas.Client) error {
	out, err := c.Cmd("SCAN")
	if err != nil {
		return fmt.Errorf("c.Cmd(\"SCAN\"): %w", err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf("c.Cmd(\"SCAN\"): %s", string(out))
	}
	return nil
}

func GetScanResults(c *wpas.Client, ssid string) ([]string, error) {
	var bssids []string
	out, err := c.Cmd("SCAN_RESULTS")
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

func BuildBSS(c *wpas.Client, bssid string) (BSS, error) {
	out, err := c.Cmd("BSS " + bssid)
	var b BSS
	if err != nil {
		return nil, fmt.Errorf("c.Cmd(\"BSS\"): %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "bssid="):
			b.BSSID = line[6:]
		case strings.HasPrefix(line, "ssid="):
			b.SSID = line[5:]
		case strings.HasPrefix(line, "freq="):
			b.Freq, err = strconv.Atoi(line[5:])
			if err != nil {
				return BSS{}, fmt.Errorf("strconv.Atoi: %w", err)
		case strings.HasPrefix(line, "beacon_int="):
			b.Freq, err = strconv.Atoi(line[11:])
			if err != nil {
				return BSS{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
		}
	}
	return b, nil
}




	//func ieParser (ie []byte) BSS