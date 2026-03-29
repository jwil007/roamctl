package config

import "time"

type Config struct {
	Preferences     `toml:"preferences"`
	RoamingTiers    `toml:"roaming_tiers"`
	Stability       `toml:"stability"`
	Timing          `toml:"timing"`
	ScoreWeights    `toml:"score_weights"`
	ScoreClamps     `toml:"score_clamps"`
	BandScores      `toml:"band_scores"`
	ChanWidthScores `toml:"chan_width_scores"`
	PhyScores       `toml:"phy_scores"`
}

type Preferences struct {
	Interface string `toml:"interface"`
}

type RoamingTiers struct {
	ExcellentRSSI int `toml:"excellent_rssi"`
	FairRSSI      int `toml:"fair_rssi"`
	FairDelta     int `toml:"fair_score_delta"`
	DegradedRSSI  int `toml:"degraded_rssi"`
	DegradedDelta int `toml:"degraded_score_delta"`
	CriticalDelta int `toml:"critical_score_delta"`
}

type Stability struct {
	RSSIHysteresisUp   int `toml:"rssi_hysteresis_up"`
	RSSIHysteresisDown int `toml:"rssi_hysteresis_down"`
	TierHysteresis     int `toml:"tier_hysteresis"`
	RetryRate          int `toml:"retry_rate"` // not used, may implement for tier select
	DataRate           int `toml:"data_rate"`  // not used, may implement for tier select
	UnhealthyScoreMod  int `toml:"unhealthy_score_mod"`
}

type Timing struct {
	SigPollInterval time.Duration `toml:"sig_poll_interval"`
	BGScanInterval  time.Duration `toml:"base_scan_interval"`
	MaxBSSCt        int           `toml:"max_bss_ct"`
}

type ScoreWeights struct {
	RSSI         int     `toml:"rssi"`
	SNR          int     `toml:"snr"`
	Band         int     `toml:"band"`
	ChannelWidth int     `toml:"channel_width"`
	EstThruput   int     `toml:"-"`
	QBSSUtil     int     `toml:"qbss_util"`
	QBSSStaCt    int     `toml:"-"`
	PHYType      int     `toml:"phy_type"`
	RSSIKnee     int     `toml:"rssi_knee"`
	RSSIExponent float64 `toml:"rssi_exponent"`
}

type ScoreClamps struct {
	MinRSSI int `toml:"min_rssi"`
	MaxRSSI int `toml:"max_rssi"`
	MinSNR  int `toml:"min_snr"`
	MaxSNR  int `toml:"max_snr"`
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
