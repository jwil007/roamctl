package config

import "time"

type Config struct {
	Preferences     `toml:"preferences"`
	Thresholds      `toml:"thresholds"`
	RoamingTiers    `toml:"roaming_tiers"`
	ScoreWeights    `toml:"score_weights"`
	ScoreClamps     `toml:"score_clamps"`
	BandScores      `toml:"band_scores"`
	ChanWidthScores `toml:"chan_width_scores"`
	PhyScores       `toml:"phy_scores"`
	Timing          `toml:"timing"`
	SSID            string `toml:"-"`
}

type Preferences struct {
	Interface string `toml:"interface"`
}

type RoamingTiers struct {
	ExcellentRSSI      int `toml:"excellent_rssi"`
	OpportunisticRSSI  int `toml:"opportunistic_rssi"`
	OpportunisticDelta int `toml:"opportunistic_delta"`
	ActiveRSSI         int `toml:"active_rssi"`
	ActiveDelta        int `toml:"active_delta"`
	CriticalRSSI       int `toml:"critical_rssi"`
	CriticalDelta      int `toml:"critical_delta"`
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
	RSSI               int `toml:"rssi"`
	RSSIHysteresisUp   int `toml:"rssi_hysteresis_up"`
	RSSIHysteresisDown int `toml:"rssi_hysteresis_down"`
	RetryRate          int `toml:"retry_rate"`
	DataRate           int `toml:"data_rate"`
	ScoreDelta         int `toml:"score_delta"`
	MaxNoCandidates    int `toml:"max_no_candidate_attempts"`
}

type ScoreWeights struct {
	RSSI int `toml:"rssi"`
	SNR  int `toml:"snr"`

	Band         int `toml:"band"`
	ChannelWidth int `toml:"channel_width"`
	EstThruput   int `toml:"-"`
	QBSSUtil     int `toml:"qbss_util"`
	QBSSStaCt    int `toml:"-"`
	PHYType      int `toml:"phy_type"`
}

type ScoreClamps struct {
	MinRSSI int `toml:"min_rssi"`
	MaxRSSI int `toml:"max_rssi"`
	MinSNR  int `toml:"min_snr"`
	MaxSNR  int `toml:"max_snr"`
}
