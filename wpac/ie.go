package wpac

import "fmt"

// This file contains functions to parse raw ie hex dump from beacon/probe response frames
func parseIETLV(ie []byte) ([]tlv, error) {
	var tlvs []tlv
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
		tlv := tlv{
			t: t,
			l: l,
			v: v,
		}
		tlvs = append(tlvs, tlv)
	}
	return tlvs, nil
}

func parseTLVs(tlvs []tlv) (IEBSS, error) {
	var ie IEBSS
	var htcw = ChannelWidthUnknown
	var vhtcw = ChannelWidthUnknown
	var hecw = ChannelWidthUnknown
	var ehtcw = ChannelWidthUnknown
	var phy = PHYLegacy
	for _, tlv := range tlvs {
		switch tlv.t {
		case 0: //SSID
			ie.SSID = string(tlv.v)
		case 1: //Supported Rates
			sr, err := parseSupportedRates(tlv.v)
			if err != nil {
				return IEBSS{}, fmt.Errorf("parseSupportedRates: %w", err)
			}
			ie.SupportedRates = sr
		//case 3: //DS Parameter Set
		case 11: //QBSS Load
			q := parseQBSSLoad(tlv.v)
			ie.QBSSStaCt = q.stationCount
			ie.QBSSUtil = q.channelUtilization
		//case 48: //RSN Information
		//case 50: //Extended Supported Rates
		//case 45: //HT Capabilities
		case 61: //HT Operation
			phy = PHY80211n
			htcw = parseHTOperation(tlv.v)
		//case 127: //Extended Capabilities
		//case 191: //VHT Capabilities
		case 192: //VHT Operation
			phy = PHY80211ac
			vhtcw = parseVHTOperation(tlv.v)
		//case 221: //Vendor Specific
		case 255: //Element ID Extension. Nested, with tlv.Value[0] indicating type
			switch tlv.v[0] {
			//case 35: //HE Capabilities
			case 36: //HE Operation
				phy = PHY80211ax
				ehtcw = parseHEOperation(tlv.v[1:])
			case 108: //EHT Capabilities
				phy = PHY80211be
				ehtcw = parseEHTCapabilities(tlv.v[1:])
			}
		}
		//reconcile channel width
		cw := max(ehtcw, hecw, vhtcw, htcw)
		//Default to 20MHz channel width to support 802.11g or older
		if cw == ChannelWidthUnknown {
			cw = ChannelWidth20
		}
		ie.ChannelWidth = cw
		//set PHYType to last written phy value, should show highest PHY in use
		ie.PHYType = phy
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

func parseHEOperation(value []byte) ChannelWidth {
	//check for 6Ghz operation being present by bit masking on first byte
	b := value[2]
	bit1 := (b >> 1) & 0x1
	if bit1 == 1 {
		cwb := value[7]
		bits0and1 := cwb & 0x3
		switch bits0and1 {
		case 0:
			return ChannelWidth20
		case 1:
			return ChannelWidth40
		case 2:
			return ChannelWidth80
		case 3:
			return ChannelWidth160
		}
	}
	return ChannelWidthUnknown
}

func parseEHTCapabilities(value []byte) ChannelWidth {
	b := value[2]
	bit1 := (b >> 1) & 0x1
	if bit1 == 1 {
		return ChannelWidth320
	}
	return ChannelWidthUnknown
}
