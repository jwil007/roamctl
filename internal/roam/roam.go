package roam

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jwil007/roamctl/internal/wpac"
)

func (rc *roamContext) handleOppRoam(c *wpac.Client, ctx context.Context) error {
	err := rc.evalAndAttemptRoam(c, ctx)
	if err != nil {
		return fmt.Errorf("evalAndAttemptRoam: %w", err)
	}
	rc.fullScanIfBSSIDsChanged()
	return nil
}

func (rc *roamContext) handleActiveRoam(c *wpac.Client, ctx context.Context) error {
	if !rc.entryScanned {
		slog.Info("Active roaming entered, running fast scan")
		err := rc.runFastScan(c, ctx)
		if err != nil && !errors.Is(err, ErrScanRetryLimit) {
			return fmt.Errorf("runFastScan: %w", err)
		}
		rc.entryScanned = true
	}
	err := rc.evalAndAttemptRoam(c, ctx)
	if err != nil {
		return fmt.Errorf("evalAndAttemptRoam: %w", err)
	}
	rc.fullScanIfBSSIDsChanged()
	return nil
}

func (rc *roamContext) handleCriticalRoam(c *wpac.Client, ctx context.Context) error {
	if !rc.entryScannedCrit {
		slog.Info("Critical roaming entered, running fast scan")
		err := rc.runFastScan(c, ctx)
		if err != nil && !errors.Is(err, ErrScanRetryLimit) {
			return fmt.Errorf("runFastScan: %w", err)
		}
		rc.entryScannedCrit = true
	}
	err := rc.evalAndAttemptRoam(c, ctx)
	if err != nil {
		return fmt.Errorf("evalAndAttemptRoam: %w", err)
	}
	rc.scanState.mu.RLock()
	stable := rc.scanState.bssListStable
	rc.scanState.mu.RUnlock()
	if rc.roamResultFlag == noCandidates {
		if !rc.fullScannedCrit {
			slog.Info("No candidates found, running break-glass full scan...")
			err = rc.runFullScan(c, ctx)
			if err != nil && !errors.Is(err, ErrScanRetryLimit) {
				return fmt.Errorf("runFullScan: %w", err)
			}
			err = rc.evalAndAttemptRoam(c, ctx)
			if err != nil {
				return fmt.Errorf("evalAndAttemptRoam: %w", err)
			}
			rc.fullScannedCrit = true
		} else {
			if !stable {
				slog.Info("BSS list has changed, running full scan...",
					"last_roam_result", rc.roamResultFlag,
					"roam_tier", rc.roamingTier)
				err = rc.runFullScan(c, ctx)
				if err != nil && !errors.Is(err, ErrScanRetryLimit) {
					return fmt.Errorf("runFullScan: %w", err)
				}
				err = rc.evalAndAttemptRoam(c, ctx)
				if err != nil {
					return fmt.Errorf("evalAndAttemptRoam: %w", err)
				}
			}
		}
	}
	return nil
}

func (rc *roamContext) evalAndAttemptRoam(
	c *wpac.Client,
	ctx context.Context) error {
	if rc.checkIfNewScan() {
		slog.Info("Evaluating candidates", "roaming_tier", rc.roamingTier)
		err := rc.prepScanResults(c)
		if err != nil {
			return fmt.Errorf("prepScanResults: %w", err)
		}
		logScoredAPs(rc)
		err = rc.attemptRoam(c, ctx)
		if err != nil {
			return fmt.Errorf("attemptRoam: %w", err)
		}
	}
	return nil
}

func (rc *roamContext) attemptRoam(c *wpac.Client, ctx context.Context) error {
	if rc.checkRoam() {
		err := rc.roamToCandidate(c, ctx)
		if err != nil {
			return fmt.Errorf("c.roamToCandidate: %w", err)
		}
	} else {
		//no candidate APs
		rc.roamResultFlag = noCandidates
		//slog.Debug("No candidate AP above roaming threshold, returning")
		return nil
	}
	return nil
}

func (rc *roamContext) checkRoam() bool {
	if rc.hysteresisActive {
		recovered := rc.lastKnown.RSSI >=
			rc.lastTriggerRSSI+rc.cfg.RSSIHysteresisUp
		degraded := rc.lastKnown.RSSI <=
			rc.lastTriggerRSSI-rc.cfg.RSSIHysteresisDown
		if recovered || degraded {
			slog.Info("RSSI Hysteresis cleared. Roam attempt allowed",
				"rssi", rc.lastKnown.RSSI)
			rc.hysteresisActive = false
		} else {
			slog.Debug("RSSI Hysteresis active. Roam not allowed",
				"rssi", rc.lastKnown.RSSI,
				"upper_bound", rc.lastTriggerRSSI+rc.cfg.RSSIHysteresisUp,
				"lower_bound", rc.lastTriggerRSSI-rc.cfg.RSSIHysteresisDown)
			return false
		}
	}
	if rc.currentAP.bssid == rc.candidateAP.bssid {
		slog.Info("Current AP is best AP in scan data, skipping roam")
		return false
	}
	switch rc.roamingTier {
	case unknownTier:
	case noRoam:
	case opportunistic:
		if rc.candidateAP.finalScore-rc.cfg.FairDelta >=
			rc.currentAP.finalScore {
			slog.Info("Score delta above threshold, attempting roam...",
				"candidate_bssid", rc.candidateAP.bssid,
				"measured_delta", rc.candidateAP.finalScore-rc.currentAP.finalScore,
				"required_delta", rc.cfg.FairDelta)
			return true
		}
		slog.Info("Score delta below threshold, no roam",
			"candidate_bssid", rc.candidateAP.bssid,
			"measured_delta", rc.candidateAP.finalScore-rc.currentAP.finalScore,
			"required_delta", rc.cfg.FairDelta)
		return false
	case active:
		if rc.candidateAP.finalScore-rc.cfg.DegradedDelta >=
			rc.currentAP.finalScore {
			slog.Info("Score delta above threshold, attempting roam...",
				"candidate_bssid", rc.candidateAP.bssid,
				"measured_delta", rc.candidateAP.finalScore-rc.currentAP.finalScore,
				"required_delta", rc.cfg.DegradedDelta)
			return true
		}
		slog.Info("Score delta below threshold, no roam",
			"candidate_bssid", rc.candidateAP.bssid,
			"measured_delta", rc.candidateAP.finalScore-rc.currentAP.finalScore,
			"required_delta", rc.cfg.DegradedDelta)
		return false
	case critical:
		if rc.candidateAP.finalScore-rc.cfg.CriticalDelta >=
			rc.currentAP.finalScore {
			slog.Info("Score delta above threshold, attempting roam...",
				"candidate_bssid", rc.candidateAP.bssid,
				"measured_delta", rc.candidateAP.finalScore-rc.currentAP.finalScore,
				"required_delta", rc.cfg.CriticalDelta)
			return true
		}
		slog.Info("Score delta below threshold, no roam",
			"candidate_bssid", rc.candidateAP.bssid,
			"measured_delta", rc.candidateAP.finalScore-rc.currentAP.finalScore,
			"required_delta", rc.cfg.CriticalDelta)
		return false
	default:
		panic(fmt.Sprintf("unknown roaming tier: %d", rc.roamingTier))
	}
	return false
}

func (rc *roamContext) roamToCandidate(
	c *wpac.Client,
	ctx context.Context,
) error {
	result, err := c.Roam(ctx, rc.candidateAP.bssid)
	if err != nil {
		if strings.Contains(err.Error(), "timed out waiting for event") {
			slog.Error("Roam attempt timed out")
			rc.roamResultFlag = failure
			return nil
		}
		rc.roamResultFlag = unknown
		return fmt.Errorf("c.Roam(%v): %w", rc.candidateAP.bssid, err)
	}
	switch result.Success {
	case true:
		slog.Info(green.Render("ROAM SUCCESS"),
			"bssid", rc.candidateAP.bssid,
			"rssi", rc.candidateAP.rssi,
			"band", rc.candidateAP.band,
			"score", rc.candidateAP.finalScore,
			"duration", result.Duration,
			"message", result.Message)
		rc.roamResultFlag = success
		rc.hysteresisActive = true
		rc.lastTriggerRSSI = rc.lastKnown.RSSI
		slog.Debug("RSSI Hysteresis active. Signal change needed next roam attempt",
			"current", rc.lastKnown.RSSI,
			"upper", rc.lastTriggerRSSI+rc.cfg.RSSIHysteresisUp,
			"lower", rc.lastTriggerRSSI-rc.cfg.RSSIHysteresisDown)
		return nil
	case false:
		slog.Warn(red.Render("ROAM FAILURE"),
			"bssid", rc.candidateAP.bssid,
			"rssi", rc.candidateAP.rssi,
			"band", rc.candidateAP.band,
			"score", rc.candidateAP.finalScore,
			"duration", result.Duration,
			"message", result.Message)
		rc.roamResultFlag = failure
		return nil
	default:
		panic(fmt.Sprintf("unknown roam result status: %v", result.Success))
	}
}

func (rc *roamContext) checkIfNewScan() bool {
	rc.scanState.mu.RLock()
	lastScan := rc.scanState.lastScanTime
	rc.scanState.mu.RUnlock()
	if lastScan.After(rc.lastEvalTime) {
		slog.Debug("Fresh scan data received")
		rc.lastEvalTime = time.Now()
		return true
	}
	slog.Debug("Scan not yet fresh",
		"last_scan_time", lastScan,
		"last_eval_time", rc.lastEvalTime,
		"delta", rc.lastEvalTime.Sub(lastScan).Seconds(),
	)
	return false
}

func (rc *roamContext) fullScanIfBSSIDsChanged() {
	rc.scanState.mu.RLock()
	stable := rc.scanState.bssListStable
	mode := rc.scanState.scanMode
	rc.scanState.mu.RUnlock()
	if rc.roamResultFlag == noCandidates {
		if !stable && mode != fullScan {
			slog.Info("BSS list has changed, requesting full channel scan",
				"roam_tier", rc.roamingTier)
			rc.scanState.mu.Lock()
			rc.scanState.scanMode = fullScan
			rc.scanState.mu.Unlock()
		}
	}
}
