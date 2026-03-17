package roam

import (
	"fmt"
	"time"

	"github.com/jwil007/roamctl/wpac"
)

type Config struct {
	Preferences     `toml:"preferences"`
	Thresholds      `toml:"thresholds"`
	ScoreWeights    `toml:"score_weights"`
	Timing          `toml:"timing"`
	BandScores      `toml:"band_scores"`
	ChanWidthScores `toml:"chan_width_scores"`
	PhyScores       `toml:"phy_scores"`
	SSID            string `toml:"-"`
}

type Preferences struct {
	Interface string `toml:"interface"`
}

type BandScores struct {
	Band2point4 int `toml:"2point4ghz"`
	Band5       int `toml:"5ghz"`
	Band6       int `toml:"6ghz"`
}

type ChanWidthScores struct {
	ChannelWidth20  int `toml:"20mhz"`
	ChannelWidth40  int `toml:"40mhz"`
	ChannelWidth80  int `toml:"80mhz"`
	ChannelWidth160 int `toml:"160mhz"`
	ChannelWidth320 int `toml:"320mhz"`
}

type PhyScores struct {
	PHYLegacy  int `toml:"legacy"`
	PHY80211n  int `toml:"80211n"`
	PHY80211ac int `toml:"80211ac"`
	PHY80211ax int `toml:"80211ax"`
	PHY80211be int `toml:"80211be"`
}

type Timing struct {
	SuccessBackoffTime      time.Duration `toml:"success_backoff_time"`
	FailureBackoffTime      time.Duration `toml:"failure_backoff_time"`
	NoCandidatesBackoffTime time.Duration `toml:"no_candidates_backoff_time"`
	SigPollInterval         time.Duration `toml:"sig_poll_interval"`
	BGScanInterval          time.Duration `toml:"bg_scan_interval"`
	MaxScanAge              time.Duration `toml:"max_scan_age"`
}
type Thresholds struct {
	RSSI       int `toml:"rssi"`
	DataRate   int `toml:"data_rate"`
	ScoreDelta int `toml:"score_delta"`
}

type ScoreWeights struct {
	RSSI         int `toml:"rssi"`
	MinRSSI      int `toml:"min_rssi"`
	MaxRSSI      int `toml:"max_rssi"`
	SNR          int `toml:"snr"`
	MinSNR       int `toml:"min_snr"`
	MaxSNR       int `toml:"max_snr"`
	Band         int `toml:"band"`
	ChannelWidth int `toml:"channel_width"`
	EstThruput   int `toml:"-"`
	QBSSUtil     int `toml:"qbss_util"`
	QBSSStaCt    int `toml:"-"`
	PHYType      int `toml:"phy_type"`
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
