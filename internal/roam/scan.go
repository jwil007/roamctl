package roam

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/jwil007/roamctl/internal/wpac"
)

func (rc *roamContext) runScanConcurrent(
	c *wpac.Client,
	ctx context.Context) <-chan error {
	errc := make(chan error, 1)
	go func() {
		err := rc.smartScan(c, ctx)
		if err != nil {
			errc <- fmt.Errorf("c.smartScan: %w", err)
			return
		}
	}()
	return errc
}

func (rc *roamContext) smartScan(c *wpac.Client, ctx context.Context) error {
	switch rc.scanState.scanMode {
	case noScan:
		return nil
	case fastScan:
		err := rc.runFastScan(c, ctx)
		if err != nil {
			return fmt.Errorf("rc.runFastScan: %w", err)
		}
	case fullScan:
		err := rc.runFullScan(c, ctx)
		if err != nil {
			return fmt.Errorf("rc.runFullScan: %w", err)
		}
	default:
		panic(fmt.Sprintf("unknown scan mode: %d", rc.scanState.scanMode))
	}
	return nil
}

func (rc *roamContext) runFastScan(c *wpac.Client, ctx context.Context) error {
	//scan only specified channels
	var ssid string
	if rc.lastKnown == nil {
		ssid = rc.ssid
	} else {
		ssid = rc.lastKnown.SSID
	}
	rc.scanState.mu.RLock()
	freqs := rc.scanState.channels
	rc.scanState.mu.RUnlock()
	sp := wpac.ScanParams{
		Freqs:      freqs,
		SSID:       ssid,
		Timeout:    20 * time.Second,
		RetryCount: 3,
	}
	err := rc.executeScan(c, ctx, sp)
	if err != nil {
		return fmt.Errorf("executeScan: %w", err)
	}
	return nil
}

func (rc *roamContext) runFullScan(c *wpac.Client, ctx context.Context) error {
	//Scan all channels by not specifying freqs
	var ssid string
	if rc.lastKnown == nil {
		ssid = rc.ssid
	} else {
		ssid = rc.lastKnown.SSID
	}
	sp := wpac.ScanParams{
		Freqs:      nil,
		SSID:       ssid,
		Timeout:    20 * time.Second,
		RetryCount: 3,
	}
	rc.scanState.mu.Lock()
	rc.scanState.scanMode = fullScan
	rc.scanState.mu.Unlock()
	err := rc.executeScan(c, ctx, sp)
	if err != nil {
		return fmt.Errorf("executeScan: %w", err)
	}
	// check scan results to build fast scan channel list
	aps, err := c.ScanResults(rc.ssid)
	if err != nil {
		return fmt.Errorf("c.ScanResults: %w", err)
	}
	freqs := getFreqsByRSSI(aps[0:min(len(aps), rc.cfg.MaxBSSCt)])
	hash := hashBSSIDs(aps[0:min(len(aps), rc.cfg.MaxBSSCt)])
	rc.scanState.mu.Lock()
	rc.scanState.channels = freqs
	rc.scanState.bssidHash = hash
	rc.scanState.bssListStable = true
	rc.scanState.scanMode = fastScan
	rc.scanState.mu.Unlock()
	slog.Info("Full scan complete",
		"channels", freqs)
	return nil
}

func (rc *roamContext) executeScan(
	c *wpac.Client,
	ctx context.Context,
	sp wpac.ScanParams) error {
	rc.scanState.mu.Lock()
	for rc.scanState.scanInProgress {
		slog.Info("Execute Scan: Scan in progress, waiting for completion")
		rc.scanState.cond.Wait()
		rc.scanState.mu.Unlock()
		slog.Info("Execute Scan: in-progress scan completed")
		return nil
	}
	rc.scanState.scanInProgress = true
	mode := rc.scanState.scanMode
	stable := rc.scanState.bssListStable
	rc.scanState.mu.Unlock()
	slog.Info("Scan dispatched",
		"trigger_tier", rc.roamingTier,
		"last_result", rc.roamResultFlag,
		"scan_mode", mode,
		"entry_scanned", rc.entryScanned,
		"entry_scanned_crit", rc.entryScannedCrit,
		"bss_stable", stable)
	start := time.Now()
	err := c.Scan(ctx, sp)
	if err != nil {
		rc.scanState.mu.Lock()
		rc.scanState.scanInProgress = false
		rc.scanState.cond.Broadcast()
		rc.scanState.mu.Unlock()
		if strings.Contains(err.Error(), "max retries exceeded") {
			slog.Warn("Scan retry limit exceeded")
			return ErrScanRetryLimit
		}
		if strings.Contains(err.Error(), "timed out waiting for event") {
			slog.Warn("Scan timed out")
			return ErrScanRetryLimit
		}
		return fmt.Errorf("c.Scan: %w", err)
	}
	duration := time.Since(start)
	completeTime := time.Now()
	rc.scanState.mu.Lock()
	rc.scanState.scanInProgress = false
	rc.scanState.scanDuration = duration
	rc.scanState.lastScanTime = completeTime
	rc.scanState.cond.Broadcast()
	rc.scanState.mu.Unlock()
	slog.Info(
		"Scan completed", "scan_mode", mode, "duration", duration)
	return nil
}

func (rc *roamContext) prepScanResults(c *wpac.Client) error {
	rc.currentAP = scoredBSS{}
	err := rc.readBSSPenaltyFile()
	if err != nil {
		slog.Error("Error reading BSS Penalty file", "err", err)
	}
	aps, err := c.ScanResults(rc.ssid)
	if err != nil {
		return fmt.Errorf("c.ScanResults: %w", err)
	}
	if len(aps) == 0 {
		slog.Warn("scanResults empty")
		return nil
	}
	rc.richBSSList = aps
	rc.scoredAPs = scoreAll(aps, rc.cfg)
	for i, ap := range rc.scoredAPs {
		if ap.bssid == rc.lastKnown.BSSID {
			rc.currentAP = ap
		}
		// check if AP is in penalty list
		for j, bp := range rc.bssPenalties {
			if bp.BSSID == ap.bssid &&
				bp.SSID == rc.ssid &&
				bp.Band == ap.band {
				//check if last fail is old enough to reset
				if time.Since(bp.LastFail) > 60*time.Minute {
					slog.Info("BSS penalty timer expired,"+
						" resetting fail count",
						"bssid", bp.BSSID)
					rc.bssPenalties = slices.Delete(rc.bssPenalties, j, j+1)
					err = rc.writeBSSPenaltyFile()
					if err != nil {
						slog.Error("Error updating BSS Penalty file",
							"err", err)
					}
					continue
				}
				slog.Info("AP with failed roams found, modifying score",
					"bssid", bp.BSSID,
					"failcount", bp.FailCount)
				rc.scoredAPs[i].failCount = bp.FailCount
				rc.scoredAPs[i].finalScore = ap.finalScore -
					bp.FailCount*rc.cfg.UnhealthyScoreMod
			}
		}
	}

	if rc.unhealthyConn {
		slog.Info("Current AP connection unhealthy, penalizing its score",
			"original_score", rc.currentAP.finalScore,
			"modified_score", rc.currentAP.finalScore-rc.cfg.UnhealthyScoreMod)
		rc.currentAP.finalScore =
			rc.currentAP.finalScore - rc.cfg.UnhealthyScoreMod
		for i := range rc.scoredAPs {
			if rc.scoredAPs[i].bssid == rc.lastKnown.BSSID {
				rc.scoredAPs[i].finalScore -= rc.cfg.UnhealthyScoreMod
				if rc.scoredAPs[i].finalScore < 0 {
					rc.scoredAPs[i].finalScore = 0
				}
				break
			}
		}
	}
	slices.SortFunc(rc.scoredAPs, func(a, b scoredBSS) int {
		return b.finalScore - a.finalScore
	})
	hash := hashBSSIDs(aps[0:min(len(aps), rc.cfg.MaxBSSCt)])
	rc.scanState.mu.Lock()
	rc.scanState.bssListStable = hash == rc.scanState.bssidHash
	stable := rc.scanState.bssListStable
	rc.scanState.mu.Unlock()
	slog.Debug("bssListStable", "bool", stable)
	rc.candidateAP = rc.scoredAPs[0]
	return nil
}

func getFreqsByRSSI(aps []wpac.RichBSS) []int {
	var freqsRaw []int
	for _, ap := range aps {
		slog.Debug("BSS to build chan list",
			"ssid", ap.SSID,
			"bssid", ap.BSSID,
			"freq", ap.Freq)
		freqsRaw = append(freqsRaw, ap.Freq)
	}
	slices.Sort(freqsRaw)
	return slices.Compact(freqsRaw)
}

func hashBSSIDs(aps []wpac.RichBSS) uint64 {
	var bssids []string
	for _, ap := range aps {
		bssids = append(bssids, ap.BSSID)
	}
	sort.Strings(bssids)
	h := fnv.New64a()
	for _, bssid := range bssids {
		_, _ = h.Write([]byte(bssid))
	}
	slog.Debug("BSSID list hashed", "hash", h.Sum64())
	return h.Sum64()
}

func logScoredAPs(rc *roamContext) {
	slog.Info(blue.Render("current ap  "), "bss", rc.currentAP)
	for _, a := range rc.scoredAPs {
		if a.bssid == rc.lastKnown.BSSID {
			continue
		} else {
			slog.Info("candidate ap", "bss", a)
		}
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
