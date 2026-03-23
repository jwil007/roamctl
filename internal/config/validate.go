package config

import (
	"fmt"
	"strings"
)

func (cfg *Config) Validate() error {
	var errs []string
	// Preferences
	if cfg.Interface == "" {
		errs = append(errs, "Interface field cannot be empty")
	}
	// Thresholds
	if !validRSSI(cfg.Thresholds.RSSI) {
		errs = append(errs, fmt.Sprintf(
			"thresholds.rssi %v invalid. Must be in range -128 to 0",
			cfg.Thresholds.RSSI))
	}
	if !validScore(cfg.Thresholds.ScoreDelta) {
		errs = append(errs, fmt.Sprintf(
			"thresholds.score_delta %v invalid. Must be in range 0 to 100",
			cfg.Thresholds.ScoreDelta))
	}
	if !validScore(cfg.Thresholds.ScoreDelta) {
		errs = append(errs, fmt.Sprintf(
			"thresholds.retry_rate %v invalid. Must be in range 0 to 100",
			cfg.Thresholds.RetryRate))
	}
	if cfg.MaxNoCandidates > 20 || cfg.MaxNoCandidates < 0 {
		errs = append(errs, fmt.Sprintf(
			"thresholds.score_delta %v invalid. Must be in range 0 to 20",
			cfg.MaxNoCandidates))
	}
	if cfg.RSSIHysteresisUp > 15 || cfg.RSSIHysteresisUp < 0 {
		errs = append(errs, fmt.Sprintf(
			"thresholds.hysteresis_up %v invalid. Must be in range 0 to 15",
			cfg.RSSIHysteresisUp))
	}
	if cfg.RSSIHysteresisDown > 15 || cfg.RSSIHysteresisDown < 0 {
		errs = append(errs, fmt.Sprintf(
			"thresholds.hysteresis_down %v invalid. Must be in range 0 to 15",
			cfg.RSSIHysteresisUp))
	}
	// ScoreWeights
	if !validScore(cfg.ScoreWeights.RSSI) {
		errs = append(errs, fmt.Sprintf(
			"score_weights.rssi %v invalid. Must be in range 0 to 100",
			cfg.ScoreWeights.RSSI))
	}
	if !validScore(cfg.ScoreWeights.SNR) {
		errs = append(errs, fmt.Sprintf(
			"score_weights.snr %v invalid. Must be in range 0 to 100",
			cfg.ScoreWeights.SNR))
	}
	if !validScore(cfg.ScoreWeights.Band) {
		errs = append(errs, fmt.Sprintf(
			"score_weights.band %v invalid. Must be in range 0 to 100",
			cfg.ScoreWeights.Band))
	}
	if !validScore(cfg.ScoreWeights.ChannelWidth) {
		errs = append(errs, fmt.Sprintf(
			"score_weights.channel_width %v invalid. Must be in range 0 to 100",
			cfg.ScoreWeights.ChannelWidth))
	}
	if !validScore(cfg.ScoreWeights.QBSSUtil) {
		errs = append(errs, fmt.Sprintf(
			"score_weights.qbss_util %v invalid. Must be in range 0 to 100",
			cfg.ScoreWeights.QBSSUtil))
	}
	if !validScore(cfg.ScoreWeights.PHYType) {
		errs = append(errs, fmt.Sprintf(
			"score_weights.phy_type %v invalid. Must be in range 0 to 100",
			cfg.ScoreWeights.PHYType))
	}
	// BandScores
	if !validScore(cfg.BandScores.Band2point4) {
		errs = append(errs, fmt.Sprintf(
			"band_scores.2point4ghz %v invalid. Must be in range 0 to 100",
			cfg.BandScores.Band2point4))
	}
	if !validScore(cfg.BandScores.Band5) {
		errs = append(errs, fmt.Sprintf(
			"band_scores.5ghz %v invalid. Must be in range 0 to 100",
			cfg.BandScores.Band5))
	}
	if !validScore(cfg.BandScores.Band6) {
		errs = append(errs, fmt.Sprintf(
			"band_scores.6ghz %v invalid. Must be in range 0 to 100",
			cfg.BandScores.Band6))
	}
	// ScoreClamps
	if !validRSSI(cfg.ScoreClamps.MinRSSI) {
		errs = append(errs, fmt.Sprintf(
			"score_clamps.min_rssi %v invalid. Must be in range -128 to 0",
			cfg.ScoreClamps.MinRSSI))
	}
	if !validRSSI(cfg.ScoreClamps.MaxRSSI) {
		errs = append(errs, fmt.Sprintf(
			"score_clamps.max_rssi %v invalid. Must be in range -128 to 0",
			cfg.ScoreClamps.MaxRSSI))
	}
	if !validScore(cfg.ScoreClamps.MinSNR) {
		errs = append(errs, fmt.Sprintf(
			"score_clamps.min_snr %v invalid. Must be in range 0 to 100",
			cfg.ScoreClamps.MinSNR))
	}
	if !validScore(cfg.ScoreClamps.MaxSNR) {
		errs = append(errs, fmt.Sprintf(
			"score_clamps.min_snr %v invalid. Must be in range 0 to 100",
			cfg.ScoreClamps.MinSNR))
	}
	// ChanWidthScores
	if !validScore(cfg.ChanWidthScores.ChannelWidth20) {
		errs = append(errs, fmt.Sprintf(
			"chan_width_scores.20mhz %v invalid. Must be in range 0 to 100",
			cfg.ChanWidthScores.ChannelWidth20))
	}
	if !validScore(cfg.ChanWidthScores.ChannelWidth40) {
		errs = append(errs, fmt.Sprintf(
			"chan_width_scores.40mhz %v invalid. Must be in range 0 to 100",
			cfg.ChanWidthScores.ChannelWidth40))
	}
	if !validScore(cfg.ChanWidthScores.ChannelWidth80) {
		errs = append(errs, fmt.Sprintf(
			"chan_width_scores.80mhz %v invalid. Must be in range 0 to 100",
			cfg.ChanWidthScores.ChannelWidth80))
	}
	if !validScore(cfg.ChanWidthScores.ChannelWidth160) {
		errs = append(errs, fmt.Sprintf(
			"chan_width_scores.160mhz %v invalid. Must be in range 0 to 100",
			cfg.ChanWidthScores.ChannelWidth160))
	}
	if !validScore(cfg.ChanWidthScores.ChannelWidth320) {
		errs = append(errs, fmt.Sprintf(
			"chan_width_scores.320mhz %v invalid. Must be in range 0 to 100",
			cfg.ChanWidthScores.ChannelWidth320))
	}
	// PhyScores
	if !validScore(cfg.PhyScores.PHYLegacy) {
		errs = append(errs, fmt.Sprintf(
			"phy_scores.legacy %v invalid. Must be in range 0 to 100",
			cfg.PhyScores.PHYLegacy))
	}
	if !validScore(cfg.PhyScores.PHY80211n) {
		errs = append(errs, fmt.Sprintf(
			"phy_scores.80211n %v invalid. Must be in range 0 to 100",
			cfg.PhyScores.PHY80211n))
	}
	if !validScore(cfg.PhyScores.PHY80211ac) {
		errs = append(errs, fmt.Sprintf(
			"phy_scores.80211ac %v invalid. Must be in range 0 to 100",
			cfg.PhyScores.PHY80211ac))
	}
	if !validScore(cfg.PhyScores.PHY80211ax) {
		errs = append(errs, fmt.Sprintf(
			"phy_scores.80211ax %v invalid. Must be in range 0 to 100",
			cfg.PhyScores.PHY80211ax))
	}
	if !validScore(cfg.PhyScores.PHY80211be) {
		errs = append(errs, fmt.Sprintf(
			"phy_scores.80211be %v invalid. Must be in range 0 to 100",
			cfg.PhyScores.PHY80211be))
	}
	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n%v", strings.Join(errs, "\n"))
	}
	return nil
}

func validScore(score int) bool {
	if score >= 0 && score <= 100 {
		return true
	}
	return false
}

func validRSSI(rssi int) bool {
	if rssi >= -128 && rssi <= 0 {
		return true
	}
	return false
}
