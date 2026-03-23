package roam

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jwil007/roamctl/internal/config"
	"github.com/jwil007/roamctl/internal/wpac"
)

func Proc(c *wpac.Client, ctx context.Context, cfg *config.Config) error {
	rc := &roamContext{}
	rc.cfg = cfg
	rc.scanState.cond = sync.NewCond(&rc.scanState.mu)
	rc.scanState.scanMode = fullScan
	err := rc.smartScan(c, ctx)
	if err != nil {
		if strings.Contains(err.Error(), "max retries exceeded") {
			slog.Warn("Background scan retry limit exceeded")
		} else {
			return fmt.Errorf("smartScan: %w", err)
		}
	}
	slog.Info("Starting roamctl... exit with ctrl+c")
	cleanup, err := rc.handleWpaSuppConfig(c)
	if err != nil {
		return fmt.Errorf("handleWpaSuppConfig: %w", err)
	}
	defer cleanup() //sets wpa_supplicant back to original state
	//Start polling signal stats
	slog.Info("Starting signal polling...")
	sigCh, sigErrCh := c.PollSignal(ctx, cfg.Timing.SigPollInterval)
	scErrCh := rc.runScanConcurrent(c, ctx)
	cadenceTicker := time.NewTicker(cfg.BGScanInterval)
	defer cadenceTicker.Stop()
	for {
		select {
		case <-cadenceTicker.C:
			rc.scanState.mu.RLock()
			if !rc.scanState.scanInProgress {
				rc.scanState.mu.RUnlock()
				scErrCh = rc.runScanConcurrent(c, ctx)
			} else {
				rc.scanState.mu.RUnlock()
			}
			if rc.roamingTier == opportunistic {
				err = rc.handleOppRoam(c, ctx)
				if err != nil {
					return fmt.Errorf("handleOppRoam: %w", err)
				}
			}
		case con := <-sigCh:
			if con.BSSID != "" {
				rc.lastKnown = &con
			}
			if rc.lastKnown == nil {
				continue
			}
			rc.evalTier()
		case err = <-sigErrCh:
			if err != nil {
				return fmt.Errorf("c.PollSignal: %w", err)
			}
		case err = <-scErrCh:
			if err != nil {
				if strings.Contains(err.Error(), "max retries exceeded") {
					slog.Warn("Background scan retry limit exceeded")
				}
				return fmt.Errorf("rc.runScanLoop: %w", err)
			}
		}
	}
}

func (rc *roamContext) runScanConcurrent(
	c *wpac.Client,
	ctx context.Context) <-chan error {
	errc := make(chan error)
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
		//scan only specified channels
		sp := wpac.ScanParams{
			Freqs:      rc.scanState.channels,
			SSID:       rc.ssid,
			Timeout:    15 * time.Second,
			RetryCount: 3,
		}
		err := rc.executeScan(c, ctx, sp)
		if err != nil {
			return fmt.Errorf("executeScan: %w", err)
		}
		return nil
	case fullScan:
		//Scan all channels by not specifying freqs
		sp := wpac.ScanParams{
			Freqs:      nil,
			SSID:       rc.ssid,
			Timeout:    15 * time.Second,
			RetryCount: 3,
		}
		err := rc.executeScan(c, ctx, sp)
		if err != nil {
			return fmt.Errorf("executeScan: %w", err)
		}
		// check scan results to build fast scan channel list
		aps, err := c.ScanResults(rc.ssid)
		if err != nil {
			return fmt.Errorf("c.ScanResults: %w", err)
		}
		freqs := getFreqs(aps[0:min(len(aps), 15)])
		rc.scanState.mu.Lock()
		rc.scanState.channels = freqs
		rc.scanState.mu.Unlock()
		hash := hashBSSIDs(aps[0:min(len(aps), 15)])
		rc.scanState.mu.Lock()
		rc.scanState.bssListStable = hash == rc.scanState.bssidHash
		rc.scanState.bssidHash = hash
		rc.scanState.mu.Unlock()
	default:
		panic(fmt.Sprintf("unknown scan mode: %d", rc.scanState.scanMode))
	}
	return nil
}

func (rc *roamContext) executeScan(c *wpac.Client, ctx context.Context, sp wpac.ScanParams) error {
	rc.scanState.mu.Lock()
	rc.scanState.scanInProgress = true
	rc.scanState.mu.Unlock()
	start := time.Now()
	err := c.Scan(ctx, sp)
	if err != nil {
		rc.scanState.mu.Lock()
		rc.scanState.scanInProgress = false
		rc.scanState.cond.Broadcast()
		rc.scanState.mu.Unlock()
		return fmt.Errorf("c.Scan: %w", err)
	}
	rc.scanState.mu.Lock()
	rc.scanState.scanInProgress = false
	rc.scanState.scanDuration = time.Since(start)
	rc.scanState.cond.Broadcast()
	rc.scanState.mu.Unlock()
	return nil
}

func (rc *roamContext) reScan(
	c *wpac.Client,
	ctx context.Context,
	sp wpac.ScanParams) error {
	rc.scanState.mu.Lock()
	for rc.scanState.scanInProgress {
		rc.scanState.cond.Wait()
	}
	rc.scanState.mu.Unlock()
	err := rc.executeScan(c, ctx, sp)
	if err != nil {
		if strings.Contains(err.Error(), "max retries exceeded") {
			slog.Warn("Background scan retry limit exceeded")
		} else {
			return fmt.Errorf("c.ExecuteScan: %w", err)
		}
	}
	aps, err := c.ScanResults(rc.cfg.SSID)
	if err != nil {
		return fmt.Errorf("c.ScanResults: %w", err)
	}
	logScoredAPs(rc)
	for _, ap := range rc.scoredAPs {
		if ap.bssid == rc.lastKnown.BSSID {
			rc.currentAP = ap
		}
	}
	rc.scoredAPs = scoreAll(aps, rc.cfg)
	return nil
}

func (rc *roamContext) evalTier() {
	switch {
	case rc.lastKnown.RSSI >= rc.cfg.ExcellentRSSI:
		rc.roamingTier = noRoam
	case rc.lastKnown.RSSI >= rc.cfg.OpportunisticRSSI &&
		rc.lastKnown.RSSI < rc.cfg.ExcellentRSSI:
		rc.roamingTier = opportunistic
	case rc.lastKnown.RSSI >= rc.cfg.ActiveRSSI &&
		rc.lastKnown.RSSI < rc.cfg.OpportunisticRSSI:
		rc.roamingTier = active
	case rc.lastKnown.RSSI <= rc.cfg.CriticalRSSI:
		rc.roamingTier = critical
	}
}

func (rc *roamContext) handleOppRoam(c *wpac.Client, ctx context.Context) error {
	err := rc.prepScanResults(c)
	if err != nil {
		return fmt.Errorf("prepScanResults: %w", err)
	}
	if rc.checkRoam() {
		sp := wpac.ScanParams{
			Freqs:      []int{rc.candidateAP.freq, rc.currentAP.freq},
			SSID:       rc.ssid,
			Timeout:    15 * time.Second,
			RetryCount: 3,
		}
		err := rc.reScan(c, ctx, sp) //rescan to ensure candidate AP is still best
		if err != nil {
			if strings.Contains(err.Error(), "max retries exceeded") {
				slog.Warn("Scan retry limit exceeded")
				return nil
			}
			return fmt.Errorf("rc.reScan: %w", err)
		}
		if rc.checkRoam() { //check one more time after rescan
			err = rc.roamToCandidate(c, ctx)
			if err != nil {
				return fmt.Errorf("c.roamToCandidate: %w", err)
			}
		}
	} else {
		//no candidate APs
		slog.Warn(yellow.Render("NO CANDIDATES") + " returning to signal monitoring")
		return nil
	}
	return nil
}

func (rc *roamContext) handleActiveRoam(c *wpac.Client, ctx context.Context) error {
	err := rc.prepScanResults(c)
	if rc.checkRoam() {
		err = rc.roamToCandidate(c, ctx)
		if err != nil {
			return fmt.Errorf("c.roamToCandidate: %w", err)
		}
	} else {
		//no candidate APs
		slog.Warn(yellow.Render("NO CANDIDATES") + " returning to signal monitoring")
		rc.scanState.mu.Lock()
		rc.scanState.scanMode = fullScan
		rc.scanState.mu.Unlock()
		return nil
	}
	return nil
}

func (rc *roamContext) checkRoam() bool {
	switch rc.roamingTier {
	case unknownTier:
	case noRoam:
	case opportunistic:
		if rc.candidateAP.finalScore-rc.cfg.OpportunisticDelta >=
			rc.currentAP.finalScore {
			return true
		}
		return false
	case active:
		if rc.candidateAP.finalScore-rc.cfg.ActiveDelta >=
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

func (rc *roamContext) handleWpaSuppConfig(c *wpac.Client) (func(), error) {
	//Get Current wpa_supplicant status
	storedConf, err := c.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("c.GetConfig: %v", err)
	}
	//Disable bgscan to prevent autonomous roaming
	noRoamConfig := wpac.WPAConfig{
		SSID:       storedConf.SSID,
		NetworkID:  storedConf.NetworkID,
		BGScan:     "",
		DisableBTM: "1",
	}
	err = c.SetConfig(noRoamConfig)
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
