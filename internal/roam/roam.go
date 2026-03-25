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
	if !rc.checkIfNewScan() {
		slog.Debug("No new scan data, skipping roam attempt")
		return nil
	}
	err := rc.prepScanResults(c)
	if err != nil {
		return fmt.Errorf("prepScanResults: %w", err)
	}
	err = rc.attemptRoam(c, ctx)
	if err != nil {
		return fmt.Errorf("attemptRoam: %w", err)
	}
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
	if !rc.checkIfNewScan() {
		slog.Debug("No new scan data, skipping roam attempt")
		return nil
	}
	err := rc.prepScanResults(c)
	if err != nil {
		return fmt.Errorf("prepScanResults: %w", err)
	}
	err = rc.attemptRoam(c, ctx)
	if err != nil {
		return fmt.Errorf("attemptRoam: %w", err)
	}
	rc.scanState.mu.RLock()
	stable := rc.scanState.bssListStable
	rc.scanState.mu.RUnlock()
	if rc.roamResultFlag == noCandidates {
		if !stable {
			slog.Info("BSS list has changed, requesting full channel scan")
			rc.scanState.mu.Lock()
			rc.scanState.scanMode = fullScan
			rc.scanState.mu.Unlock()
		}
	}
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
	if !rc.checkIfNewScan() {
		slog.Debug("No new scan data, skipping roam attempt")
		return nil
	}
	err := rc.prepScanResults(c)
	if err != nil {
		return fmt.Errorf("prepScanResults: %w", err)
	}
	err = rc.attemptRoam(c, ctx)
	if err != nil {
		return fmt.Errorf("attemptRoam: %w", err)
	}
	rc.scanState.mu.RLock()
	stable := rc.scanState.bssListStable
	rc.scanState.mu.RUnlock()
	if rc.roamResultFlag == noCandidates {
		if !stable {
			slog.Info("Running full scan immediately...")
			// break glass full scan
			err = rc.runFullScan(c, ctx)
			if err != nil && !errors.Is(err, ErrScanRetryLimit) {
				return fmt.Errorf("runFullScan: %w", err)
			}
			err = rc.prepScanResults(c)
			if err != nil {
				return fmt.Errorf("prepScanResults: %w", err)
			}
			err = rc.attemptRoam(c, ctx)
			if err != nil {
				return fmt.Errorf("attemptRoam: %w", err)
			}
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
		slog.Debug("No candidate AP above roaming threshold, returning")
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
			return false
		}
	}
	switch rc.roamingTier {
	case unknownTier:
	case noRoam:
	case opportunistic:
		if rc.candidateAP.finalScore-rc.cfg.FairDelta >=
			rc.currentAP.finalScore {
			return true
		}
		return false
	case active:
		if rc.candidateAP.finalScore-rc.cfg.DegradedDelta >=
			rc.currentAP.finalScore {
			return true
		}
		return false
	case critical:
		if rc.candidateAP.finalScore-rc.cfg.CriticalDelta >=
			rc.currentAP.finalScore {
			return true
		}
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
	slog.Info("Roam complete", "stats", result)
	switch result.Success {
	case true:
		slog.Info(green.Render("ROAM SUCCESS"),
			"bssid", rc.candidateAP.bssid,
			"rssi", rc.candidateAP.rssi,
			"band", rc.candidateAP.band,
			"score", rc.candidateAP.finalScore)
		slog.Info("Waiting for next trigger...")
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
			"reason", result.Message)
		slog.Info("Waiting for next trigger...")
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
		rc.lastEvalTime = time.Now()
		return true
	}
	rc.lastEvalTime = time.Now()
	return false
}
