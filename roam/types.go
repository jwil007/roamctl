package roam

import (
	"time"

	"github.com/jwil007/roamctl/wpac"
)

type Config struct {
	Thresholds
	ScoreWeights
	Timing
}

type Timing struct {
	RoamBackoffTime time.Duration
	SigPollInterval time.Duration
	BGScanInterval  time.Duration
}
type Thresholds struct {
	RSSI       int
	DataRate   int
	ScoreDelta int
}

type ScoreWeights struct {
	RSSI         int
	MinRSSI      int
	MaxRSSI      int
	SNR          int
	MinSNR       int
	MaxSNR       int
	Band         int
	ChannelWidth int
	EstThruput   int
	QBSSUtil     int
	QBSSStaCt    int
	PHYType      int
}

type scoredBSS struct {
	bssid      string
	finalScore int
	rssiScore  int
	rssi       int
	snrScore   int
	snr        int
	bandScore  int
	band       wpac.Band
	cwScore    int
	cw         wpac.ChannelWidth
	utilScore  int
	util       uint8
	phyScore   int
	phy        wpac.PHYType
	age        int
}
