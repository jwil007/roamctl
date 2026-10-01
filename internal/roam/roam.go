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

func (rc *roamContext) handleOppRoam(
	c *wpac.Client,
	ctx context.Context) error {
	err := rc.evalAndAttemptRoam(c, ctx)
	if err != nil {
		return fmt.Errorf("evalAndAttemptRoam: %w", err)
	}
	//request full scan for next cycle if bssid list changed
	rc.fullScanIfBSSIDsChanged()
	return nil
}

func (rc *roamContext) handleActiveRoam(
	c *wpac.Client,
	ctx context.Context) error {
	//scan on entry if flag unset
	if !rc.entryScanned {
		slog.Info("Active roaming entered, running scan...")
		err := rc.smartScan(c, ctx)
		if err != nil && !errors.Is(err, ErrScanRetryLimit) {
			return fmt.Errorf("runFastScan: %w", err)
		}
		rc.entryScanned = true
	}
	err := rc.evalAndAttemptRoam(c, ctx)
	if err != nil {
		return fmt.Errorf("evalAndAttemptRoam: %w", err)
	}
	//request full scan for next cycle if bssid list changed
	rc.fullScanIfBSSIDsChanged()
	return nil
}

func (rc *roamContext) handleCriticalRoam(
	c *wpac.Client,
	ctx context.Context) error {
	//scan on entry if flag unset
	if !rc.entryScannedCrit {
		slog.Info("Critical roaming entered, running scan...")
		err := rc.smartScan(c, ctx)
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
		//do inline full scan if prior attempt didn't find a better AP
		if !rc.fullScannedCrit {
			slog.Info(
				"No candidates found, running break-glass full scan...")
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
		err := rc.attemptRoam(c, ctx)
		if err != nil {
			return fmt.Errorf("attemptRoam: %w", err)
		}
	}
	return nil
}

func (rc *roamContext) attemptRoam(c *wpac.Client, ctx context.Context) error {
	// Take one consistent view of the scored APs. A scan goroutine may rescore
	// (and replace candidateAP) at any time, including during the roam below.
	rc.apMu.Lock()
	current, candidate := rc.currentAP, rc.candidateAP
	rc.apMu.Unlock()
	if rc.checkRoam(current, candidate) {
		if !rc.roamInProgress {
			err := rc.roamToCandidate(c, ctx, candidate)
			if err != nil {
				return fmt.Errorf("c.roamToCandidate: %w", err)
			}
		} else {
			//no candidate APs
			rc.roamResultFlag = noCandidates
			//slog.Debug("No candidate AP above roaming threshold, returning")
			return nil
		}
	}
	return nil
}

func (rc *roamContext) checkRoam(current, candidate scoredBSS) bool {
	//various checks to gate roam attempt
	rc.checkRSSIHysteresis()
	if rc.hysteresisActive {
		return false
	}
	if candidate.finalScore-current.finalScore >= rc.cfg.FairDelta*2 {
		slog.Info("Score delta overrides hysteresis, roam attempt allowed",
			"measured_delta",
			candidate.finalScore-current.finalScore,
			"override_threshold", rc.cfg.FairDelta*2)
		rc.hysteresisActive = false
	} else if rc.hysteresisActive {
		slog.Debug("RSSI Hysteresis active. Roam not allowed",
			"rssi", rc.lastKnown.RSSI,
			"upper_bound", rc.lastTriggerRSSI+rc.cfg.RSSIHysteresisUp,
			"lower_bound", rc.lastTriggerRSSI-rc.cfg.RSSIHysteresisDown)
		return false
	}
	if current.bssid == candidate.bssid {
		slog.Info("Current AP is best AP in scan data, skipping roam")
		rc.roamResultFlag = noCandidates
		return false
	}
	switch rc.roamingTier {
	case unknownTier:
	case noRoam:
	case opportunistic:
		if candidate.finalScore-rc.cfg.FairDelta >=
			current.finalScore {
			slog.Info("Score delta above threshold, attempting roam...",
				"candidate_bssid", candidate.bssid,
				"measured_delta",
				candidate.finalScore-current.finalScore,
				"required_delta", rc.cfg.FairDelta)
			return true
		}
		slog.Info("Score delta below threshold, no roam",
			"candidate_bssid", candidate.bssid,
			"measured_delta",
			candidate.finalScore-current.finalScore,
			"required_delta", rc.cfg.FairDelta)
		return false
	case active:
		if candidate.finalScore-rc.cfg.DegradedDelta >=
			current.finalScore {
			slog.Info("Score delta above threshold, attempting roam...",
				"candidate_bssid", candidate.bssid,
				"measured_delta",
				candidate.finalScore-current.finalScore,
				"required_delta", rc.cfg.DegradedDelta)
			return true
		}
		slog.Info("Score delta below threshold, no roam",
			"candidate_bssid", candidate.bssid,
			"measured_delta",
			candidate.finalScore-current.finalScore,
			"required_delta", rc.cfg.DegradedDelta)
		return false
	case critical:
		if candidate.finalScore-rc.cfg.CriticalDelta >=
			current.finalScore {
			slog.Info("Score delta above threshold, attempting roam...",
				"candidate_bssid", candidate.bssid,
				"measured_delta",
				candidate.finalScore-current.finalScore,
				"required_delta",
				rc.cfg.CriticalDelta)
			return true
		}
		slog.Info("Score delta below threshold, no roam",
			"candidate_bssid", candidate.bssid,
			"measured_delta", candidate.finalScore-current.finalScore,
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
	candidate scoredBSS,
) error {
	if candidate.bssid == "" {
		return fmt.Errorf("roamToCandidate: roam aborted, empty BSSID")
	}
	rc.roamInProgress = true
	rc.updateSnapshot()
	result, err := c.Roam(ctx, candidate.bssid)
	if err != nil {
		if strings.Contains(err.Error(), "timed out waiting for event") {
			slog.Error("Roam attempt timed out")
			result.Message = "Roam attempt timed out"
			rc.lastRoamStats.CompletedAt = time.Now()
			rc.onRoamFailure(result, candidate)
			return nil
		}
		rc.roamResultFlag = unknown
		rc.unhealthyConn = false
		return fmt.Errorf("c.Roam(%v): %w", candidate.bssid, err)
	}
	switch result.Success {
	case true:
		rc.onRoamSuccess(result, candidate)
		return nil
	case false:
		rc.onRoamFailure(result, candidate)
		return nil
	default:
		panic(fmt.Sprintf(
			"unknown roam result status: %v", result.Success))
	}
}

func (rc *roamContext) checkRSSIHysteresis() {
	if rc.hysteresisActive {
		recovered := rc.lastKnown.RSSI >=
			rc.lastTriggerRSSI+rc.cfg.RSSIHysteresisUp
		degraded := rc.lastKnown.RSSI <=
			rc.lastTriggerRSSI-rc.cfg.RSSIHysteresisDown
		if recovered || degraded {
			slog.Info("RSSI Hysteresis cleared. Roam attempt allowed",
				"rssi", rc.lastKnown.RSSI)
			rc.hysteresisActive = false
		}
	}
	return
}

func (rc *roamContext) onRoamSuccess(result wpac.RoamStats, candidate scoredBSS) {
	slog.Info(green.Render("ROAM SUCCESS"),
		"bssid", candidate.bssid,
		"rssi", candidate.rssi,
		"band", candidate.band,
		"score", candidate.finalScore,
		"duration", result.Duration,
		"message", result.Message)
	rc.lastRoamStats = result
	rc.roamResultFlag = success
	rc.onConnectionChange()
	rc.lastRoamAttempt = time.Now()
	rc.hysteresisActive = true
	rc.lastTriggerRSSI = rc.lastKnown.RSSI
	slog.Info(
		"RSSI Hysteresis active. Signal change needed next roam attempt",
		"current", rc.lastKnown.RSSI,
		"upper", rc.lastTriggerRSSI+rc.cfg.RSSIHysteresisUp,
		"lower", rc.lastTriggerRSSI-rc.cfg.RSSIHysteresisDown)
	err := rc.recordBSSPenalty(false, candidate)
	if err != nil {
		slog.Error("Error updating BSS Penalty file",
			"err", err)
	}
	rc.roamInProgress = false
	rc.updateSnapshot()
}

func (rc *roamContext) onRoamFailure(result wpac.RoamStats, candidate scoredBSS) {
	slog.Warn(red.Render("ROAM FAILURE"),
		"bssid", candidate.bssid,
		"rssi", candidate.rssi,
		"band", candidate.band,
		"score", candidate.finalScore,
		"duration", result.Duration,
		"message", result.Message)
	rc.lastRoamStats = result
	rc.roamResultFlag = failure
	rc.lastRoamAttempt = time.Now()
	rc.unhealthyConn = false
	slog.Info("Logging failed roam attempt",
		"bssid", candidate.bssid)
	err := rc.recordBSSPenalty(true, candidate)
	if err != nil {
		slog.Error("Error updating BSS Penalty file",
			"err", err)
	}
	rc.roamInProgress = false
	rc.updateSnapshot()
}
