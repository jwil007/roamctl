package wpac

type WPAConfig struct {
	SSID      string
	NetworkID string
	BGScan    string
	Iface     string
}

type Signal struct {
	RSSI          int
	LinkSpeed     int
	Noise         int
	Freq          int
	ChannelWidth  string
	AvgRSSI       int
	AvgRSSIBeacon int
}

type TLV struct {
	Type   byte
	Length byte
	Value  []byte
}

type qbssLoad struct {
	stationCount               uint16
	channelUtilization         uint8
	availableAdmissionCapacity uint16
}

type RichBSS struct {
	WpasBSS
	IEBSS
	Band       Band
	ChannelNum int
}

type WpasBSS struct {
	BSSID      string
	Freq       int
	BeaconInt  int
	Noise      int
	RSSI       int
	SNR        int
	Age        int
	Flags      string
	EstThruput int
	ProbeIE    []byte
	BeaconIE   []byte
}

type IEBSS struct {
	SSID           string
	SupportedRates []string
	ChannelWidth   ChannelWidth
	QBSSUtil       uint8
	QBSSStaCt      uint16
	PHYType        PHYType
}

var supportedRates = map[byte]string{
	0x02: "1",
	0x03: "1.5",
	0x04: "2",
	0x05: "2.5",
	0x06: "3",
	0x09: "4.5",
	0x0B: "5.5",
	0x0C: "6",
	0x12: "9",
	0x16: "11",
	0x18: "12",
	0x1B: "13.5",
	0x24: "18",
	0x2C: "22",
	0x30: "24",
	0x36: "27",
	0x42: "33",
	0x48: "36",
	0x60: "48",
	0x6C: "54",
	0x82: "1(B)",
	0x83: "1.5(B)",
	0x84: "2(B)",
	0x85: "2.5(B)",
	0x86: "3(B)",
	0x89: "4.5(B)",
	0x8B: "5.5(B)",
	0x8C: "6(B)",
	0x92: "9(B)",
	0x96: "11(B)",
	0x98: "12(B)",
	0x9B: "13.5(B)",
	0xA4: "18(B)",
	0xAC: "22(B)",
	0xB0: "24(B)",
	0xB6: "27(B)",
	0xC2: "33(B)",
	0xC8: "36(B)",
	0xE0: "48(B)",
	0xEC: "54(B)",
	//bss membership selectors
	0xFA: "HE PHY",
	0xFB: "SAE Hash to Element Only",
	0xFC: "EPD", /* 802.11ak */
	0xFD: "GLK", /* 802.11ak */
	0xFE: "VHT PHY",
	0xFF: "HT PHY",
}

type ChannelWidth int

const (
	ChannelWidthUnknown ChannelWidth = iota
	ChannelWidth20
	ChannelWidth40
	ChannelWidth80
	ChannelWidth160
	ChannelWidth80Plus80
	ChannelWidth320
)

func (cw ChannelWidth) String() string {
	switch cw {
	case ChannelWidth20:
		return "20MHz"
	case ChannelWidth40:
		return "40MHz"
	case ChannelWidth80:
		return "80MHz"
	case ChannelWidth160:
		return "160MHz"
	case ChannelWidth80Plus80:
		return "80+80MHz"
	case ChannelWidth320:
		return "320Mhz"
	case ChannelWidthUnknown:
		return "Unknown"
	}
	return ""
}

type PHYType int

const (
	PHYUnknown PHYType = iota
	PHYLegacy
	PHY80211n
	PHY80211ac
	PHY80211ax
	PHY80211be
)

func (ph PHYType) String() string {
	switch ph {
	case PHYLegacy:
		return "Legacy a/b/g"
	case PHY80211n:
		return "802.11n"
	case PHY80211ac:
		return "802.11ac"
	case PHY80211ax:
		return "802.11ax"
	case PHY80211be:
		return "802.11be"
	case PHYUnknown:
		return "Unknown"
	}
	return ""
}

type Band int

const (
	BandUnknown Band = iota
	Band2point4
	Band5
	Band6
)

func (b Band) String() string {
	switch b {
	case Band2point4:
		return "2.4GHz"
	case Band5:
		return "5GHz"
	case Band6:
		return "6GHz"
	case BandUnknown:
		return "Unknown"
	}
	return ""
}
