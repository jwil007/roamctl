package wpac

// This file contains the API surface
import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
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

func (c *Client) Scan(ctx context.Context, ssid string) ([]RichBSS, error) {
	//run scan and collect scan results to build bssid list
	errScan := c.runScan()
	if errScan != nil {
		return nil, fmt.Errorf("wpac.runScan: %w", errScan)
	}
	errWait := c.WaitForEvent(ctx, "CTRL-EVENT-SCAN-RESULTS", 10*time.Second)
	if errWait != nil {
		return nil, fmt.Errorf("c.WaitForEvent: %w", errWait)
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
		fmt.Printf("BSSID:%s Freq:%d Band:%s Channel:%d BeaconInt:%d Noise:%d RSSI:%d SNR:%d Age:%d"+
			" Flags:%s EstThruput:%d SSID:%s Rates:%v CW:%s QBSSUtil:%d QBSSStaCt:%d PHYType:%s\n",
			r.BSSID,
			r.Freq,
			r.Band,
			r.ChannelNum,
			r.BeaconInt,
			r.Noise,
			r.RSSI,
			r.SNR,
			r.Age,
			r.Flags,
			r.EstThruput,
			r.SSID,
			r.SupportedRates,
			r.ChannelWidth,
			r.QBSSUtil,
			r.QBSSStaCt,
			r.PHYType,
		)
	}
	return richBSSList, nil
}

func (c *Client) PollSignal(ctx context.Context, interval time.Duration) (<-chan Signal, <-chan error) {
	signal := make(chan Signal)
	errc := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s, err := c.getSignal()
				if err != nil {
					errc <- err
					return
				}
				signal <- s
			}
		}
	}()
	return signal, errc
}

func (c *Client) ListenEvents(ctx context.Context) (<-chan string, <-chan error) {
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
