package roam

import (
	"fmt"
	"time"

	"github.com/jwil007/roamctl/wpac"
)

type Config struct {
	Thresholds   `toml:"Thresholds"`
	ScoreWeights `toml:"ScoreWeights"`
	Timing       `toml:"Timing"`
	Preferences  `toml:"Preferences"`
	SSID         string `toml:"-"`
}

type Preferences struct {
	Interface string
}

type Timing struct {
	SuccessBackoffTime      time.Duration
	FailureBackoffTime      time.Duration
	NoCandidatesBackoffTime time.Duration
	SigPollInterval         time.Duration
	BGScanInterval          time.Duration
	MaxScanAge              time.Duration
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
	age        time.Duration
}

func (s scoredBSS) String() string {
	return fmt.Sprintf(
		"bssid:%s score:%d rssi:%d(scr:%d) snr:%d(scr:%d) band:%s(scr:%d) cw:%s(scr:%d) util:%d(scr:%d) phy:%s(scr:%d) age:%s",
		s.bssid,
		s.finalScore,
		s.rssi, s.rssiScore,
		s.snr, s.snrScore,
		s.band, s.bandScore,
		s.cw, s.cwScore,
		s.util, s.utilScore,
		s.phy, s.phyScore,
		s.age,
	)
}

type roamResultFlag int

const (
	success roamResultFlag = iota
	failure
	noCandidates
	unknown
)
