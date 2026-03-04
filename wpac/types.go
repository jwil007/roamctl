package wpac

type WPAConfig struct {
	SSID      string
	NetworkID string
	BGScan    string
	Iface     string
}

type BSS struct {
	BSSID        string
	SSID         string
	Freq         int
	Band         string
	Channel      int
	BeaconInt    int
	Noise        int
	RSSI         int
	Age          int
	Flags        string
	EstThruput   int
	Load         QBSSLoad
	ChannelWidth string
}

type QBSSLoad struct {
	StationCount               int
	ChannelUtilization         int
	AvailableAdmissionCapacity int
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
