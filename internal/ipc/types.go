package ipc

import (
	"time"

	"github.com/jwil007/roamctl/internal/wpac"
)

type ProcessState struct {
	SSID            string
	BSSList         []BSS
	ConnState       wpac.ConnectionStatus
	RoamStats       wpac.RoamStats
	RoamingTier     string
	RoamResultFlag  string
	LastTriggerRSSI int
	Flags
	ScanState
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
}

type ScanState struct {
	ScanInProgress bool
	ScanMode       string
	ScanDuration   time.Duration
	BSSListStable  bool
}
