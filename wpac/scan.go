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
		//fmt.Printf("wpasBSS: %+v\n", wpasBSS)
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
		fmt.Printf("BSSID:%s Freq:%d Band:%s BeaconInt:%d Noise:%d RSSI:%d SNR:%d Age:%d Flags:%s EstThruput:%d SSID:%s Rates:%v CW:%s QBSSUtil:%d QBSSStaCt:%d PHYType:%d\n",
			r.BSSID,
			r.Freq,
			r.Band,
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
	return RichBSS{
		WpasBSS: wpaBSS,
		IEBSS:   ieBSS,
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
			b.ProbeIE = line[3:]
		case strings.HasPrefix(line, "beacon_ie="):
			b.BeaconIE = line[10:]
		}
	}
	return b, nil
}

func parseIETLV(ieString string) ([]TLV, error) {
	ie, err := hex.DecodeString(ieString)
	if err != nil {
		return nil, fmt.Errorf("hex.DecodeString: %w", err)
	}
	var tlvs []TLV

	i := 0
	for i < len(ie) {
		t := ie[i]
		i += 1
		l := ie[i]
		i += 1
		if i+int(l) > len(ie) {
			return nil, fmt.Errorf("length parsed in tlv exceeds total ie length")
		}
		v := ie[i : i+int(l)]
		i += len(v)
		tlv := TLV{
			Type:   t,
			Length: l,
			Value:  v,
		}
		tlvs = append(tlvs, tlv)
	}
	return tlvs, nil
}

func parseTLVs(tlvs []TLV) (IEBSS, error) {
	var ie IEBSS
	var htcw = ChannelWidthUnknown
	var vhtcw = ChannelWidthUnknown
	for _, tlv := range tlvs {
		switch tlv.Type {
		case 0: //SSID
			ie.SSID = string(tlv.Value)
		case 1: //Supported Rates
			sr, err := parseSupportedRates(tlv.Value)
			if err != nil {
				return IEBSS{}, fmt.Errorf("parseSupportedRates: %w", err)
			}
			ie.SupportedRates = sr
		case 3: //DS Parameter Set
		case 11: //QBSS Load
			q := parseQBSSLoad(tlv.Value)
			ie.QBSSStaCt = q.stationCount
			ie.QBSSUtil = q.channelUtilization
		case 48: //RSN Information
		case 50: //Extended Supported Rates
		case 45: //HT Capabilities
		case 61: //HT Operation
			htcw = parseHTOperation(tlv.Value)
		case 127: //Extended Capabilities
		case 191: //VHT Capabilities
		case 192: //VHT Operation
			vhtcw = parseVHTOperation(tlv.Value)
		case 221: //Vendor Specific
		case 255: //Element ID Extension
		}
		//reconcile maximum channel width
		if vhtcw > htcw {
			ie.ChannelWidth = vhtcw
		} else {
			ie.ChannelWidth = htcw
		}
	}
	return ie, nil
}

func parseSupportedRates(value []byte) ([]string, error) {
	var rates []string
	for _, b := range value {
		rate, ok := supportedRates[b]
		if !ok {
			return nil, fmt.Errorf("unknown rate detected at byte: %v", b)
		}
		rates = append(rates, rate)
	}
	return rates, nil
}

func parseQBSSLoad(value []byte) qbssLoad {
	var q qbssLoad
	q.stationCount = uint16(value[0] | value[1])
	q.channelUtilization = value[2]
	q.availableAdmissionCapacity = uint16(value[3] | value[4])
	return q
}

func parseHTOperation(value []byte) ChannelWidth {
	b := value[1]
	bits0and1 := b & 0x3
	bit2 := (b >> 2) & 0x1
	switch {
	case bits0and1 == 0 && bit2 == 0:
		return ChannelWidth20
	case bits0and1 != 0 && bit2 == 1:
		return ChannelWidth40
	}
	return ChannelWidthUnknown
}

func parseVHTOperation(value []byte) ChannelWidth {
	switch {
	case value[0] == 1 && value[1] != 0 && value[2] != 0:
		return ChannelWidth160
	case value[0] == 1 && value[1] != 0 && value[2] == 0:
		return ChannelWidth80
	case value[0] == 0:
		return ChannelWidthUnknown
	}
	return ChannelWidthUnknown
}
