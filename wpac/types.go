package wpac

type WPAConfig struct {
	SSID      string
	NetworkID string
	BGScan    string
	Iface     string
}

type RichBSS struct {
	WpasBSS
	IEBSS
}

type WpasBSS struct {
	BSSID      string // IE?
	Freq       int    //IE?
	Band       string //derived from freq
	BeaconInt  int    //IE?
	Noise      int
	RSSI       int
	SNR        int
	Age        int
	Flags      string
	EstThruput int
	ProbeIE    string
	BeaconIE   string
}

type IEBSS struct {
	SSID           string
	SupportedRates []string
	ChannelWidth   ChannelWidth
	QBSSUtil       uint8
	QBSSStaCt      uint16
	PHYType        int
}

var ieNames = map[byte]string{
	0:   "SSID",
	1:   "Supported Rates",
	3:   "DS Parameter Set",
	11:  "QBSS Load",
	48:  "RSN Information",
	50:  "Extended Supported Rates",
	45:  "HT Capabilities",
	61:  "HT Operation",
	127: "Extended Capabilities",
	191: "VHT Capabilities",
	192: "VHT Operation",
	221: "Vendor Specific",
	255: "Element ID Extension", // HE and EHT live behind this
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
		return "20Mhz"
	case ChannelWidth40:
		return "40Mhz"
	case ChannelWidth80:
		return "80Mhz"
	case ChannelWidth160:
		return "160Mhz"
	case ChannelWidth80Plus80:
		return "80+80Mhz"
	case ChannelWidth320:
		return "320Mhz"
	case ChannelWidthUnknown:
		return "Unknown"
	}
	return ""
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
