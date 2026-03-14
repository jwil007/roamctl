package wpac

// This file contains the API surface
import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (c *Client) Roam(ctx context.Context, bssid string) (RoamStats, error) {
	var r RoamStats
	r.TargetBSSID = bssid
	eventMatch := []string{
		"CTRL-EVENT-AUTH-REJECT",
		"CTRL-EVENT-ASSOC-REJECT",
		"CTRL-EVENT-DISCONNECTED",
		"CTRL-EVENT-CONNECTED",
	}
	start := time.Now()
	errRoam := c.runRoam(bssid)
	if errRoam != nil {
		return RoamStats{}, fmt.Errorf("c.runRoam(%v): %w", bssid, errRoam)
	}
	ev, err := c.waitForEvent(ctx, eventMatch, 15*time.Second)
	if err != nil {
		return RoamStats{}, fmt.Errorf("c.waitForEvent: %w", err)
	}
	r.Duration = time.Since(start)
	switch {
	case strings.Contains(ev, "CTRL-EVENT-AUTH-REJECT"):
		sc, errSt := extractStatusCode(ev)
		if errSt != nil {
			return RoamStats{}, fmt.Errorf("extractStatusCode(%v): %w", ev, errSt)
		}
		r.Success = false
		r.Message = "Auth rejected - " + sc
	case strings.Contains(ev, "CTRL-EVENT-ASSOC-REJECT"):
		sc, errSt := extractStatusCode(ev)
		if errSt != nil {
			return RoamStats{}, fmt.Errorf("extractStatusCode(%v): %w", ev, errSt)
		}
		r.Success = false
		r.Message = "Assoc rejected - " + sc
	case strings.Contains(ev, "CTRL-EVENT-DISCONNECTED"):
		rc, errEx := extractReasonCode(ev)
		if errEx != nil {
			return RoamStats{}, fmt.Errorf("extractReasonCode(%v): %w", ev, errEx)
		}
		r.Success = false
		r.Message = "Disconnected - " + rc
	case strings.Contains(ev, "CTRL-EVENT-CONNECTED"):
		f := strings.Fields(ev)
		for _, e := range f {
			if isMACAddress(e) {
				r.FinalBSSID = e
			}
		}
		if r.FinalBSSID == bssid {
			r.Success = true
		} else {
			r.Success = false
			r.Message = "Target and final BSSID do not match"
		}
	}
	return r, nil
}

func (c *Client) GetConfig() (WPAConfig, error) {
	ssid, _, err := c.getSSID()
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
	}, nil
}

func (c *Client) SetConfig(config WPAConfig) error {
	err := c.setBGScan(config)
	if err != nil {
		return fmt.Errorf("setBGScan: %w", err)
	}
	return nil
}

func (c *Client) Scan(ctx context.Context, ssid string) ([]RichBSS, error) {
	//run scan and collect scan results to build bssid list
	errScan := c.runScan()
	if errScan != nil {
		return nil, fmt.Errorf("wpac.runScan: %w", errScan)
	}
	_, errWait := c.waitForEvent(ctx, []string{"CTRL-EVENT-SCAN-RESULTS"}, 10*time.Second)
	if errWait != nil {
		return nil, fmt.Errorf("c.waitForEvent: %w", errWait)
	}
	bssids, err := c.getScanResults(ssid)
	if err != nil {
		return nil, fmt.Errorf("c.getScanResults: %w", err)
	}
	//process bssid list
	var wpasBSSList []WpasBSS
	var richBSSList []RichBSS
	for _, bss := range bssids {
		b, err := c.parseWpasBSS(bss)
		if err != nil {
			return nil, fmt.Errorf("parseWpasBSS: %w", err)
		}
		wpasBSSList = append(wpasBSSList, b)
	}
	for _, wpasBSS := range wpasBSSList {
		tlvList, err := parseIETLV(wpasBSS.ProbeIE)
		if err != nil {
			return nil, fmt.Errorf("parseIETLV: %w", err)
		}
		ieBSS, err := parseTLVs(tlvList)
		if err != nil {
			return nil, fmt.Errorf("parseTLVs: %w", err)
		}
		//fmt.Printf("Data parsed from beacon TLVs:\n %+v\n", ieBSS)
		richBSSList = append(richBSSList, constructRichBSS(wpasBSS, ieBSS))
	}
	for _, r := range richBSSList {
		fmt.Printf("SSID:%s BSSID:%s Freq:%d Band:%s Channel:%d BeaconInt:%d Noise:%d RSSI:%d SNR:%d Age:%d"+
			" EstThruput:%d CW:%s QBSSUtil:%d QBSSStaCt:%d PHYType:%s Flags:%s Rates:%v\n",
			r.SSID,
			r.BSSID,
			r.Freq,
			r.Band,
			r.ChannelNum,
			r.BeaconInt,
			r.Noise,
			r.RSSI,
			r.SNR,
			r.Age,
			r.EstThruput,
			r.ChannelWidth,
			r.QBSSUtil,
			r.QBSSStaCt,
			r.PHYType,
			r.Flags,
			r.SupportedRates,
		)
	}
	return richBSSList, nil
}

func (c *Client) PollSignal(ctx context.Context, interval time.Duration) (<-chan ConnectionStatus, <-chan error) {
	connStatus := make(chan ConnectionStatus)
	errc := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			case <-ticker.C:
				s, err := c.constructConnStatus()
				if err != nil && !strings.Contains(err.Error(), "ssid or bssid field not found") {
					errc <- err
					return
				}
				connStatus <- s
			}
		}
	}()
	return connStatus, errc
}
