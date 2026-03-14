package wpac

import (
	"net"
	"time"
)

type Client struct {
	CC             *net.UnixConn //CommandConnection
	EC             *net.UnixConn //EventConnection
	Iface          string
	LocalPathCmd   string
	LocalPathEvent string
}
type WPAConfig struct {
	SSID      string
	NetworkID string
	BGScan    string
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

type ConnectionStatus struct {
	Signal
	SSID  string
	BSSID string
}

type RoamStats struct {
	Success     bool
	TargetBSSID string
	FinalBSSID  string
	Duration    time.Duration
	Message     string
}

type tlv struct {
	t byte
	l byte
	v []byte
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

var reasonCodes = map[int]string{
	1:  "Unspecified reason",
	2:  "Previous authentication no longer valid",
	3:  "Deauthenticated, STA is leaving the BSS",
	4:  "Disassociated due to inactivity",
	5:  "Disassociated, AP unable to handle all associated STAs",
	6:  "Class 2 frame from unauthenticated STA",
	7:  "Class 3 frame from unassociated STA",
	8:  "Disassociated, STA is leaving the BSS",
	9:  "STA requested association without prior authentication",
	10: "Power capability element is unacceptable",
	11: "Supported channels element is unacceptable",
	12: "Disassociated due to BSS transition",
	13: "Invalid information element",
	14: "MIC failure (Michael)",
	15: "4-Way handshake timed out",
	16: "Group key update timed out",
	17: "IE in 4-Way handshake differs from (re)association request",
	18: "Invalid group cipher",
	19: "Invalid pairwise cipher",
	20: "Invalid AKMP",
	21: "Unsupported RSN IE version",
	22: "Invalid RSN IE capabilities",
	23: "IEEE 802.1X authentication failed",
	24: "Cipher suite rejected by security policy",
	25: "TDLS teardown, peer unreachable",
	26: "TDLS teardown, unspecified reason",
	27: "SSP requested disassociation",
	28: "No SSP roaming agreement",
	29: "Bad cipher or AKM",
	30: "Not authorized at this location",
	31: "Service change precludes traffic stream",
	32: "Unspecified QoS reason",
	33: "Insufficient bandwidth for QoS traffic stream",
	34: "Excessive frames unacknowledged",
	35: "TXOP limit exceeded",
	36: "STA leaving, end traffic stream or BA",
	37: "Requested from peer STA, end traffic stream or BA or DLS",
	38: "Unknown traffic stream or BA",
	39: "Request timed out",
	45: "Peer key mismatch",
	46: "Authorized access limit reached",
	47: "External service requirements",
	48: "Invalid FT action frame count",
	49: "Invalid PMKID",
	50: "Invalid MDE",
	51: "Invalid FTE",
	52: "Mesh peering cancelled",
	53: "Mesh maximum peer links reached",
	54: "Mesh configuration policy violation",
	55: "Mesh close received",
	56: "Mesh maximum retries reached",
	57: "Mesh confirm timed out",
	58: "Mesh invalid GTK",
	59: "Mesh inconsistent parameters",
	60: "Mesh invalid security capabilities",
	61: "Mesh path error, no proxy information",
	62: "Mesh path error, no forwarding information",
	63: "Mesh path error, destination unreachable",
	64: "MAC address already exists in MBSS",
	65: "Mesh channel switch required by regulation",
	66: "Mesh channel switch, unspecified reason",
}

var statusCodes = map[int]string{
	0:   "Success",
	1:   "Unspecified failure",
	2:   "TDLS wakeup request rejected, use alternative",
	3:   "TDLS wakeup request rejected",
	5:   "Security disabled",
	6:   "Unacceptable lifetime",
	7:   "Not in same BSS",
	10:  "Capabilities unsupported",
	11:  "Reassociation denied, no existing association",
	12:  "Association denied, unspecified reason",
	13:  "Authentication algorithm unsupported",
	14:  "Unknown authentication transaction sequence",
	15:  "Authentication challenge failure",
	16:  "Authentication timed out",
	17:  "AP unable to handle additional STAs",
	18:  "Association denied, supported rates mismatch",
	19:  "Association denied, short preamble unsupported",
	22:  "Spectrum management required",
	23:  "Power capability unacceptable",
	24:  "Supported channels unacceptable",
	25:  "Association denied, no short slot time",
	27:  "Association denied, HT not supported",
	28:  "R0KH unreachable",
	29:  "Association denied, no PCO",
	30:  "Association rejected temporarily",
	31:  "Robust management frame policy violation",
	32:  "Unspecified QoS failure",
	33:  "Association denied, insufficient bandwidth",
	34:  "Association denied, poor channel conditions",
	35:  "Association denied, QoS not supported",
	37:  "Request declined",
	38:  "Invalid parameters",
	39:  "Rejected with suggested changes",
	40:  "Invalid information element",
	41:  "Invalid group cipher",
	42:  "Invalid pairwise cipher",
	43:  "Invalid AKMP",
	44:  "Unsupported RSN IE version",
	45:  "Invalid RSN IE capabilities",
	46:  "Cipher rejected per policy",
	47:  "Traffic stream not created",
	48:  "Direct link not allowed",
	49:  "Destination STA not present",
	50:  "Destination STA not a QoS STA",
	51:  "Association denied, listen interval too large",
	52:  "Invalid FT action frame count",
	53:  "Invalid PMKID",
	54:  "Invalid MDIE",
	55:  "Invalid FTIE",
	56:  "Requested TCLAS not supported",
	57:  "Insufficient TCLAS processing resources",
	58:  "Try another BSS",
	59:  "GAS advertisement protocol not supported",
	60:  "No outstanding GAS request",
	61:  "GAS response not received",
	62:  "STA timed out waiting for GAS response",
	63:  "GAS response larger than limit",
	64:  "Request refused, home network",
	65:  "Advertisement service unreachable",
	67:  "Request refused by SSPN",
	68:  "Request refused, unauthorized access",
	72:  "Invalid RSN IE",
	73:  "U-APSD coexistence not supported",
	74:  "U-APSD coexistence mode not supported",
	75:  "Bad interval with U-APSD coexistence",
	76:  "Anti-clogging token required",
	77:  "Finite cyclic group not supported",
	78:  "Cannot find alternative TBTT",
	79:  "Transmission failure",
	80:  "Requested TCLAS not supported",
	81:  "TCLAS resources exhausted",
	82:  "Rejected with suggested BSS transition",
	83:  "Rejected with schedule",
	84:  "Rejected, no wakeup specified",
	85:  "Success, power save mode",
	86:  "Pending admitting FST session",
	87:  "Performing FST now",
	88:  "Pending gap in BA window",
	89:  "Rejected, U-PID setting",
	92:  "Refused, external reason",
	93:  "Refused, AP out of memory",
	94:  "Rejected, emergency services not supported",
	95:  "Query response outstanding",
	96:  "Rejected, DSE band",
	97:  "TCLAS processing terminated",
	98:  "Traffic stream schedule conflict",
	99:  "Denied with suggested band and channel",
	100: "MCCAOP reservation conflict",
	101: "MAF limit exceeded",
	102: "MCCA track limit exceeded",
	103: "Denied due to spectrum management",
	104: "Association denied, VHT not supported",
	105: "Enablement denied",
	106: "Restriction from authorized GDB",
	107: "Authorization de-enabled",
	112: "FILS authentication failure",
	113: "Unknown authentication server",
	123: "Unknown password identifier",
	124: "Denied, HE (802.11ax) not supported",
	126: "SAE hash-to-element required",
	127: "SAE-PK required",
	130: "Denied, STA affiliated with MLD has existing association",
	131: "EPCS denied, unauthorized",
	132: "EPCS denied",
	133: "Denied, TID-to-link mapping",
	134: "Preferred TID-to-link mapping suggested",
	135: "Denied, EHT (802.11be) not supported",
	136: "Invalid public key",
	137: "PASN base AKMP failed",
	138: "OCI mismatch",
	139: "Denied, TX link not accepted",
	140: "EPCS denied, verification failure",
	141: "Denied, operation parameter update",
	152: "802.1X authentication failed",
	153: "802.1X authentication succeeded",
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
