package wpac

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

func (c *Client) RunScan() error {
	out, err := c.cmd("SCAN")
	if err != nil {
		return fmt.Errorf("c.Cmd(\"SCAN\"): %w", err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf("c.Cmd(\"SCAN\"): %s", string(out))
	}
	return nil
}

func (c *Client) GetScanResults(ssid string) ([]string, error) {
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

func (c *Client) ParseWpasBSS(bssid string) (WpasBSS, error) {
	out, err := c.cmd("BSS " + bssid)
	var b WpasBSS
	if err != nil {
		return WpasBSS{}, fmt.Errorf("c.Cmd(\"BSS\"): %w", err)
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

func ParseIETLV(ieString string) ([]TLV, error) {
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

func ParseTLVs(tlvs []TLV) (IEBSS, error) {
	var ie IEBSS
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
		case 127: //Extended Capabilities
		case 191: //VHT Capabilities
		case 192: //VHT Operation
		case 221: //Vendor Specific
		case 255: //Element ID Extension
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
