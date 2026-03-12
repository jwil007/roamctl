package wpac

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

func Scan(iface string, ssid string) ([]RichBSS, error) {
	//make unix socket connections for cmd and event streaming
	cc, err := Connect(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to open ctrl_iface unix connection: %w", err)
	}
	defer func() {
		err := cc.Close()
		if err != nil {
			log.Fatalf("failed to close unix connection: %v", err)
		}
	}()
	ce, err := Connect(iface)
	if err != nil {
		return nil, fmt.Errorf("failed to open ctrl_iface unix connection: %w", err)
	}
	defer func() {
		err := ce.Close()
		if err != nil {
			log.Fatalf("failed to close unix connection: %v", err)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//run scan and collect scan results to build bssid list
	errScan := cc.runScan()
	if errScan != nil {
		return nil, fmt.Errorf("wpac.runScan: %w", errScan)
	}
	errWait := ce.WaitForEvent(ctx, "CTRL-EVENT-SCAN-RESULTS", 10*time.Second)
	if errWait != nil {
		return nil, fmt.Errorf("ce.WaitForEvent: %w", errWait)
	}
	bssids, err := cc.getScanResults(ssid)

	//process bssid list
	var wpasBSSList []WpasBSS
	var richBSSList []RichBSS
	for _, bss := range bssids {
		b, err := cc.parseWpasBSS(bss)
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

func constructRichBSS(wpaBSS WpasBSS, ieBSS IEBSS) RichBSS {
	band, channel, err := getBandandChanfromFreq(wpaBSS.Freq)
	if err != nil {
		log.Printf("getBandandChanfromFreq: %v", err)
	}
	return RichBSS{
		WpasBSS:    wpaBSS,
		IEBSS:      ieBSS,
		Band:       band,
		ChannelNum: channel,
	}
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

func getBandandChanfromFreq(freq int) (Band, int, error) {
	var channel int
	switch {
	case freq == 2484:
		channel = 14
		return Band2point4, channel, nil
	case freq >= 2412 && freq <= 2472:
		channel = (freq - 2407) / 5
		return Band2point4, channel, nil
	case freq >= 5180 && freq <= 5825:
		channel = (freq - 5000) / 5
		return Band5, channel, nil
	case freq >= 5955 && freq <= 7115:
		channel = (freq - 5950) / 5
		return Band6, channel, nil
	}
	return BandUnknown, channel, fmt.Errorf("failed to determine channel/band from freq: %v", freq)
}
