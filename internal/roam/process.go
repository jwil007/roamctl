// Package roam: handles the main roaming loop
package roam

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jwil007/roamctl/internal/wpac"
)

func (cfg *Config) ProcessLoop(c *wpac.Client, ctx context.Context) error {
	rc := &roamContext{}
	slog.Info("Starting roamctl... exit with ctrl+c")
	cleanup, err := cfg.handleWpaSuppConfig(c, rc)
	if err != nil {
		return fmt.Errorf("handleWpaSuppConfig: %w", err)
	}
	defer cleanup() //sets wpa_supplicant back to original state
	//Start polling signal stats
	slog.Info("Waiting for trigger to enter roam decision loop...")
	sigCh, sigErrCh := c.PollSignal(ctx, cfg.Timing.SigPollInterval)
	bgScanTicker := time.NewTicker(cfg.Timing.BGScanInterval)
	for {
		select {
		case <-bgScanTicker.C:
			slog.Debug("bgScanTicker reached 0, running scan...")
			rc.bgScanReady = false
			if err = c.Scan(ctx); err != nil {
				return fmt.Errorf("c.Scan: %w", err)
			}
			slog.Debug("bgScan complete")
			rc.lastBGScan = time.Now()
			rc.bgScanReady = true
			rc.bgScanChecked = false
		case <-ctx.Done():
			return ctx.Err()
		case con := <-sigCh:
			if con.BSSID != "" {
				rc.lastKnown = &con
			}
			if rc.lastKnown == nil {
				continue
			}
			slog.Debug("Last polled connection status", "stats", rc.lastKnown)
			if cfg.thresholdCheck(rc) {
				if cfg.backoffCheck(rc) {
					cfg.logRoamEntry(rc)
					err := cfg.roamProcessWrapper(c, ctx, rc)
					if err != nil {
						return fmt.Errorf("roamProcessWrapper: %w", err)
					}
				} else {
					cfg.logBackoff(rc)
					continue
				}
			} else {
				cfg.logThreshold(rc)
				continue
			}
		case err = <-sigErrCh:
			return fmt.Errorf("c.PollSignal: %w", err)
		}
	}
}

func (cfg *Config) thresholdCheck(rc *roamContext) bool {
	rssi := rc.lastKnown.AvgRSSIBeacon
	if rc.lastKnown.AvgRSSIBeacon == 0 {
		// slog.Info("No RSSI BEACON available, falling back to basic RSSI")
		rssi = rc.lastKnown.RSSI
	}
	// slog.Info("threshold RSSI recorded as: %d", rssi)
	if rc.hysteresisActive {
		if rc.lastKnown.RSSI > rc.lastTriggerRSSI-cfg.RSSIHysteresisDown &&
			rc.lastKnown.RSSI < rc.lastTriggerRSSI+cfg.RSSIHysteresisUp {
			rc.thresholdFlag = inHysteresis
			return false
		}
		slog.Info("Hysteresis cleared", "rssi", rssi)
		rc.hysteresisActive = false
	}
	switch {
	case rc.noCandCounter > cfg.MaxNoCandidates && rssi < cfg.Thresholds.RSSI:
		rc.thresholdFlag = noCandidateLimit
		rc.waitForBGScan = true
		rc.lastTriggerRSSI = rssi
		return true
	case rssi < cfg.Thresholds.RSSI:
		rc.thresholdFlag = lowRSSI
		rc.lastTriggerRSSI = rssi
		return true
	case rc.lastKnown.LinkSpeed < cfg.Thresholds.DataRate:
		rc.thresholdFlag = lowDataRate
		return true
	}
	rc.thresholdFlag = noValue
	return false
}

func (cfg *Config) backoffCheck(rc *roamContext) bool {
	if rc.waitForBGScan {
		if rc.bgScanReady && !rc.bgScanChecked {
			rc.bgScanChecked = true
			return true
		}
		rc.backoffTriggerCt++
		return false
	}
	if time.Since(rc.lastRoamFailure) < cfg.FailureBackoffTime {
		rc.backoffTrigger = failureBackoff
		rc.backoffTriggerCt++
		return false
	}
	if time.Since(rc.lastRoamSuccess) < cfg.SuccessBackoffTime {
		rc.backoffTrigger = successBackoff
		rc.backoffTriggerCt++
		return false
	}
	if time.Since(rc.lastNoCandidates) < cfg.NoCandidatesBackoffTime {
		rc.backoffTrigger = noCandidatesBackoff
		rc.backoffTriggerCt++
		return false
	}
	rc.backoffTriggerCt = 0
	return true
}

func (cfg *Config) logThreshold(rc *roamContext) {
	if rc.backoffTriggerCt < 2 { //only log on first trigger
		switch rc.thresholdFlag {
		// Thresholds
		case noValue:
			return
		case noCandidateLimit:
			slog.Info("No candidate attempts exceed threshold, falling back to bgscan",
				"attempts", rc.noCandCounter,
				"threshold", cfg.MaxNoCandidates)
		case lowRSSI:
			slog.Info("Last polled RSSI below threshold. Entering roam decision loop...",
				"rssi", rc.lastTriggerRSSI,
				"threshold", cfg.Thresholds.RSSI)
		case lowDataRate:
			slog.Info("Last polled data rate below threshold. Entering roam decision loop...",
				"datarate", rc.lastKnown.LinkSpeed,
				"threshold", cfg.Thresholds.DataRate)
		case inHysteresis:
			slog.Debug("Hysteresis active, waiting for signal to change by configured bounds...",
				"rssi", rc.lastTriggerRSSI,
				"exit_threshold_up", rc.lastTriggerRSSI+cfg.Thresholds.RSSIHysteresisUp,
				"exit_threshold_down", rc.lastTriggerRSSI-cfg.Thresholds.RSSIHysteresisDown)
		}
	}
}

func (cfg *Config) logBackoff(rc *roamContext) {
	slog.Debug("roamEnterCounter", "count", rc.roamEnterCounter)
	if rc.backoffTriggerCt < 2 { //only log on first trigger
		switch {
		case rc.waitForBGScan:
			slog.Info("Waiting for next bgscan...",
				"remaining", cfg.BGScanInterval-time.Since(rc.lastBGScan))
		case rc.backoffTrigger == noBackoff:
			return
		case rc.backoffTrigger == successBackoff:
			slog.Info("Roam success backoff in effect",
				"remaining", cfg.SuccessBackoffTime-time.Since(rc.lastRoamSuccess))
		case rc.backoffTrigger == failureBackoff:
			slog.Info("Roam failure backoff in effect",
				"remaining", cfg.FailureBackoffTime-time.Since(rc.lastRoamFailure))
		case rc.backoffTrigger == noCandidatesBackoff:
			slog.Info("No candidates backoff in effect.",
				"remaining", cfg.NoCandidatesBackoffTime-time.Since(rc.lastNoCandidates))
		}
	}
}

func (cfg *Config) logRoamEntry(rc *roamContext) {
	slog.Debug("roamEnterCounter", "count", rc.roamEnterCounter)
	if rc.backoffTriggerCt < 2 { //only log on first trigger
		switch {
		// Thresholds
		case rc.thresholdFlag == noValue:
			return
		case rc.thresholdFlag == noCandidateLimit:
			slog.Info("No candidate attempts exceed threshold, falling back to bgscan",
				"attempts", rc.noCandCounter,
				"threshold", cfg.MaxNoCandidates)
		case rc.thresholdFlag == lowRSSI:
			slog.Info("Last polled RSSI below threshold. Entering roam decision loop...",
				"rssi", rc.lastTriggerRSSI,
				"threshold", cfg.Thresholds.RSSI)
		case rc.thresholdFlag == lowDataRate:
			slog.Info("Last polled data rate below threshold. Entering roam decision loop...",
				"datarate", rc.lastKnown.LinkSpeed,
				"threshold", cfg.Thresholds.DataRate)
		// Backoff timers
		case rc.waitForBGScan:
			slog.Info("Waiting for next bgscan...",
				"remaining", cfg.BGScanInterval-time.Since(rc.lastBGScan))
		case rc.backoffTrigger == noBackoff:
			return
		case rc.backoffTrigger == successBackoff:
			slog.Info("Roam success backoff in effect",
				"remaining", cfg.SuccessBackoffTime-time.Since(rc.lastRoamSuccess))
		case rc.backoffTrigger == failureBackoff:
			slog.Info("Roam failure backoff in effect",
				"remaining", cfg.FailureBackoffTime-time.Since(rc.lastRoamFailure))
		case rc.backoffTrigger == noCandidatesBackoff:
			slog.Info("No candidates backoff in effect.",
				"remaining", cfg.NoCandidatesBackoffTime-time.Since(rc.lastNoCandidates))
		}
	}
}

func (cfg *Config) roamProcessWrapper(
	c *wpac.Client,
	ctx context.Context,
	rc *roamContext,
) error {
	resultFlag, errR := cfg.roamDecisionLoop(c, ctx, rc)
	if errR != nil {
		return fmt.Errorf("makeRoamDecision %w", errR)
	}
	switch resultFlag {
	case success:
		rc.lastRoamSuccess = time.Now()
		rc.waitForBGScan = false
		rc.lastKnown = nil //clear lastKnown stats
		rc.noCandCounter = 0
		rc.roamEnterCounter = 0
	case failure:
		rc.lastRoamFailure = time.Now()
		rc.waitForBGScan = false
		rc.lastKnown = nil
		rc.noCandCounter = 0
		rc.roamEnterCounter = 0
	case noCandidates:
		rc.lastNoCandidates = time.Now()
		rc.noCandCounter++
		rc.roamEnterCounter = 0
		rc.hysteresisActive = true
		slog.Info(yellow.Render("NO CANDIDATE") + " returning to signal monitoring...")
		slog.Debug("No candidates counter",
			"count", rc.noCandCounter,
			"max", cfg.MaxNoCandidates)
	case unknown:
		rc.roamEnterCounter = 0
		return fmt.Errorf("unexpected roam result")
	}
	return nil
}

func (cfg *Config) roamDecisionLoop(
	c *wpac.Client,
	ctx context.Context,
	rc *roamContext,
) (roamResultFlag, error) {
	var scoredAPs []scoredBSS
	var currAP scoredBSS
	if rc.thresholdFlag == noCandidateLimit {
		slog.Info("Roaming using background scan...")
		aps, err := c.ScanResults(rc.ssid)
		if err != nil {
			return unknown, fmt.Errorf("c.ScanResults: %w", err)
		}
		saps := cfg.scoreAll(aps)
		scoredAPs = saps
		logScoredAPs(scoredAPs, rc.lastKnown.BSSID)
		for _, ap := range scoredAPs {
			if ap.bssid == rc.lastKnown.BSSID {
				currAP = ap
			}
		}
	} else {
		prepAPs, err := cfg.prepareScoredAPs(c, ctx, rc)
		if err != nil {
			return unknown, fmt.Errorf("prepareScoredAPs: %w", err)
		}
		scoredAPs = prepAPs
		for _, ap := range scoredAPs {
			if ap.bssid == rc.lastKnown.BSSID {
				currAP = ap
			}
		}
	}
	if len(scoredAPs) == 0 {
		return unknown, fmt.Errorf("scored APs array is nil")
	}
	candAP := scoredAPs[0] //scoredAP is sorted with highest score first
	switch {
	case currAP.bssid == "":
		slog.Warn("current AP not in scan data, selecting AP with highest score")
		flag, err := cfg.roamToCandidate(c, ctx, candAP)
		if err != nil {
			return flag, fmt.Errorf("roamToCandidate: %w", err)
		}
		return flag, nil
	case cfg.roamReadyCheck(candAP, currAP, rc) == true:
		slog.Info("Better AP found",
			"bssid", candAP.bssid,
			"score", candAP.finalScore)
		flag, err := cfg.roamToCandidate(c, ctx, candAP)
		if err != nil {
			return flag, fmt.Errorf("roamToCandidate: %w", err)
		}
		return flag, nil
	default:
		return noCandidates, nil
	}
}

func (cfg *Config) prepareScoredAPs(
	c *wpac.Client,
	ctx context.Context,
	rc *roamContext,
) ([]scoredBSS, error) {
	aps, err := c.ScanResults(rc.ssid)
	if err != nil {
		return nil, fmt.Errorf("c.ScanResults: %w", err)
	}
	scoredAPs := cfg.scoreAll(aps)
	logScoredAPs(scoredAPs, rc.lastKnown.BSSID)
	hasFreshCandidate := false

	for _, candAP := range scoredAPs {
		if candAP.age < cfg.MaxScanAge {
			hasFreshCandidate = true
			break
		}
	}
	if !hasFreshCandidate {
		slog.Info("Stale scan data, rerunning scan...")
		out, err := cfg.rescan(c, ctx, rc.ssid)
		if err != nil {
			return nil, fmt.Errorf("cfg.rescan: %w", err)
		}
		scoredAPs = out
		logScoredAPs(scoredAPs, rc.lastKnown.BSSID)
	}
	return scoredAPs, nil
}

func (cfg *Config) roamToCandidate(
	c *wpac.Client,
	ctx context.Context,
	candAP scoredBSS,
) (roamResultFlag, error) {
	result, err := c.Roam(ctx, candAP.bssid)
	if err != nil {
		return failure, fmt.Errorf("c.Roam(%v): %w", candAP.bssid, err)
	}
	slog.Info("Roam complete", "stats", result)
	switch result.Success {
	case true:
		slog.Info(green.Render("ROAM SUCCESS"),
			"bssid", candAP.bssid,
			"rssi", candAP.rssi,
			"band", candAP.band,
			"score", candAP.finalScore)
		slog.Info("Waiting for next trigger...")
		return success, nil
	case false:
		slog.Warn(red.Render("ROAM FAILURE"),
			"bssid", candAP.bssid,
			"rssi", candAP.rssi,
			"band", candAP.band,
			"score", candAP.finalScore,
			"reason", result.Message)
		slog.Info("Waiting for next trigger...")
		return failure, nil
	}
	return unknown, fmt.Errorf("missing result.Success data from c.Roam")
}

func (cfg *Config) roamReadyCheck(candidate scoredBSS, current scoredBSS, rc *roamContext) bool {
	switch rc.thresholdFlag {
	case noValue:
	case noCandidateLimit:
		slog.Debug("Ignoring scan age limit, selecting best AP available.")
		if candidate.finalScore-current.finalScore > cfg.ScoreDelta &&
			candidate.bssid != current.bssid {
			return true
		}
		return false
	case lowRSSI:
		if candidate.finalScore-current.finalScore > cfg.ScoreDelta &&
			candidate.bssid != current.bssid &&
			candidate.age < cfg.MaxScanAge {
			return true
		}
		return false
	case lowDataRate:
		if candidate.finalScore-current.finalScore > cfg.ScoreDelta &&
			candidate.bssid != current.bssid &&
			candidate.age < cfg.MaxScanAge {
			return true
		}
	default:
		return false
	}
	return false
}

func (cfg *Config) rescan(c *wpac.Client, ctx context.Context, ssid string) ([]scoredBSS, error) {
	if err := c.Scan(ctx); err != nil {
		return nil, fmt.Errorf("c.Scan: %w", err)
	}
	aps, err := c.ScanResults(ssid)
	if err != nil {
		return nil, fmt.Errorf("c.ScanResults: %w", err)
	}
	scoredAPs := cfg.scoreAll(aps)
	return scoredAPs, nil
}

func logScoredAPs(scoredAPs []scoredBSS, bssid string) {
	slog.Info("Most recent scan data")
	for _, a := range scoredAPs {
		if a.bssid == bssid {
			slog.Info(blue.Render("curr ap"), "bss", a)
		} else {
			slog.Info("cand ap", "bss", a)
		}
	}
}

func (cfg *Config) handleWpaSuppConfig(c *wpac.Client, rc *roamContext) (func(), error) {
	//Get Current wpa_supplicant status
	storedConf, err := c.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("c.GetConfig: %v", err)
	}
	//Disable bgscan to prevent autonomous roaming
	bgscanOffConfig := wpac.WPAConfig{
		SSID:      storedConf.SSID,
		NetworkID: storedConf.NetworkID,
		BGScan:    "",
	}
	err = c.SetConfig(bgscanOffConfig)
	if err != nil {
		return nil, fmt.Errorf("c.SetConfig: %w", err)
	}
	rc.ssid = storedConf.SSID

	cleanup := func() {
		err = c.SetConfig(storedConf)
		if err != nil {
			slog.Error("error restoring wpa_supplicant config", "value", err)
		}
	}
	return cleanup, nil
}
