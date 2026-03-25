package config

import (
	"fmt"
	"strings"
)

func (cfg *Config) Validate() error {
	var errs []string

	if cfg.Interface == "" {
		errs = append(errs, "preferences.interface cannot be empty")
	}

	// RoamingTiers
	if !validRSSI(cfg.RoamingTiers.ExcellentRSSI) {
		errs = append(errs, fmt.Sprintf(
			"roaming_tiers.excellent_rssi %v invalid. Must be in range -128 to 0",
			cfg.RoamingTiers.ExcellentRSSI))
	}
	if !validRSSI(cfg.RoamingTiers.OpportunisticRSSI) {
		errs = append(errs, fmt.Sprintf(
			"roaming_tiers.opportunistic_rssi %v invalid. Must be in range -128 to 0",
			cfg.RoamingTiers.OpportunisticRSSI))
	}
	if !validScore(cfg.RoamingTiers.OpportunisticDelta) {
		errs = append(errs, fmt.Sprintf(
			"roaming_tiers.opportunistic_score_delta %v invalid. Must be in range 0 to 100",
			cfg.RoamingTiers.OpportunisticDelta))
	}
	if !validRSSI(cfg.RoamingTiers.ActiveRSSI) {
		errs = append(errs, fmt.Sprintf(
			"roaming_tiers.active_rssi %v invalid. Must be in range -128 to 0",
			cfg.RoamingTiers.ActiveRSSI))
	}
	if !validScore(cfg.RoamingTiers.ActiveDelta) {
		errs = append(errs, fmt.Sprintf(
			"roaming_tiers.active_score_delta %v invalid. Must be in range 0 to 100",
			cfg.RoamingTiers.ActiveDelta))
	}
	if !validRSSI(cfg.RoamingTiers.CriticalRSSI) {
		errs = append(errs, fmt.Sprintf(
			"roaming_tiers.critical_rssi %v invalid. Must be in range -128 to 0",
			cfg.RoamingTiers.CriticalRSSI))
	}
	if !validScore(cfg.RoamingTiers.CriticalDelta) {
		errs = append(errs, fmt.Sprintf(
			"roaming_tiers.critical_score_delta %v invalid. Must be in range 0 to 100",
			cfg.RoamingTiers.CriticalDelta))
	}
	// tier ordering
	if cfg.RoamingTiers.ExcellentRSSI <= cfg.RoamingTiers.OpportunisticRSSI {
		errs = append(errs, "roaming_tiers.excellent_rssi must be greater than opportunistic_rssi")
	}
	if cfg.RoamingTiers.OpportunisticRSSI <= cfg.RoamingTiers.ActiveRSSI {
		errs = append(errs, "roaming_tiers.opportunistic_rssi must be greater than active_rssi")
	}
	if cfg.RoamingTiers.ActiveRSSI <= cfg.RoamingTiers.CriticalRSSI {
		errs = append(errs, "roaming_tiers.active_rssi must be greater than critical_rssi")
	}

	// Stability
	if cfg.Stability.RSSIHysteresisUp > 15 || cfg.Stability.RSSIHysteresisUp < 0 {
		errs = append(errs, fmt.Sprintf(
			"stability.rssi_hysteresis_up %v invalid. Must be in range 0 to 15",
			cfg.Stability.RSSIHysteresisUp))
	}
	if cfg.Stability.RSSIHysteresisDown > 15 || cfg.Stability.RSSIHysteresisDown < 0 {
		errs = append(errs, fmt.Sprintf(
			"stability.rssi_hysteresis_down %v invalid. Must be in range 0 to 15",
			cfg.Stability.RSSIHysteresisDown))
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
	if cfg.ScoreClamps.MinRSSI >= cfg.ScoreClamps.MaxRSSI {
		errs = append(errs, "score_clamps.min_rssi must be less than max_rssi")
	}
	if !validScore(cfg.ScoreClamps.MinSNR) {
		errs = append(errs, fmt.Sprintf(
			"score_clamps.min_snr %v invalid. Must be in range 0 to 100",
			cfg.ScoreClamps.MinSNR))
	}
	if !validScore(cfg.ScoreClamps.MaxSNR) {
		errs = append(errs, fmt.Sprintf(
			"score_clamps.max_snr %v invalid. Must be in range 0 to 100",
			cfg.ScoreClamps.MaxSNR))
	}
	if cfg.ScoreClamps.MinSNR >= cfg.ScoreClamps.MaxSNR {
		errs = append(errs, "score_clamps.min_snr must be less than max_snr")
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
