package ipc

import (
	"time"
)

type ProcessState struct {
	SSID    string
	BSSList []BSS
	ConnState
	RoamStats
	RoamingTier     string
	RoamResultFlag  string
	LastTriggerRSSI int
	Flags
	ScanState
}

type ConnState struct {
	//wpac.status
	SSID     string
	WPAState string
	//netlink
	RxBitrate     int
	RxMCS         int
	RxPHY         string
	TxBitrate     int
	TxMCS         int
	TxPHY         string
	TxRetries     int
	RetryRate     int
	TxFails       int
	BeaconLoss    int
	RSSI          int
	AvgRSSI       int
	AvgRSSIBeacon int
	ConnDuration  time.Duration
	BSSID         string
	Freq          int
	ChannelWidth  string
}

type RoamStats struct {
	Success     bool
	TargetBSSID string
	FinalBSSID  string
	Duration    time.Duration
	Message     string
	CompletedAt time.Time
}

type BSS struct {
	BSSID        string
	Freq         int
	ChannelNum   int
	Band         string
	ChannelWidth string
	PHYType      string
	BeaconInt    int
	Noise        int
	RSSI         int
	SNR          int
	Age          time.Duration
	Flags        string
	EstThruput   int
	QBSSUtil     uint8
	QBSSStaCt    uint16
	FinalScore   int
	RssiScore    int
	SnrScore     int
	BandScore    int
	CwScore      int
	UtilScore    int
	PhyScore     int
	IsCurrentAP  bool
}
type Flags struct {
	HysteresisActive bool
	EntryScanned     bool
	EntryScannedCrit bool
	FullScannedCrit  bool
	UnhealthyConn    bool
	RoamInProgress   bool
}

type ScanState struct {
	ScanInProgress bool
	ScanMode       string
	ScanDuration   time.Duration
	BSSListStable  bool
	LastScanTime   time.Time
}
