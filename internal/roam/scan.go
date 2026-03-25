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
	rc.scanState.mu.RLock()
	freqs := rc.scanState.channels
	rc.scanState.mu.RUnlock()
	sp := wpac.ScanParams{
		Freqs:      freqs,
		SSID:       rc.ssid,
		Timeout:    15 * time.Second,
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
	freqs := getFreqsByRSSI(aps[0:min(len(aps), rc.cfg.MaxBSSCt)])
	slog.Info("Channels identified for fast scan", "channels", freqs)
	hash := hashBSSIDs(aps[0:min(len(aps), rc.cfg.MaxBSSCt)])
	rc.scanState.mu.Lock()
	rc.scanState.channels = freqs
	rc.scanState.bssListStable = hash == rc.scanState.bssidHash
	rc.scanState.bssidHash = hash
	rc.scanState.mu.Unlock()
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
		if strings.Contains(err.Error(), "max retries exceeded") {
			slog.Warn("Scan retry limit exceeded")
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
	slog.Info("Scan completed", "duration", duration)
	return nil
}

func (rc *roamContext) reScan(
	c *wpac.Client,
	ctx context.Context,
	sp wpac.ScanParams) error {
	rc.scanState.mu.RLock()
	inProgress := rc.scanState.scanInProgress
	rc.scanState.mu.RUnlock()
	if !inProgress {
		err := rc.executeScan(c, ctx, sp)
		if err != nil {
			return fmt.Errorf("c.ExecuteScan: %w", err)
		}
	} else {
		rc.scanState.mu.Lock()
		for rc.scanState.scanInProgress {
			rc.scanState.cond.Wait()
		}
		rc.scanState.mu.Unlock()
	}
	err := rc.prepScanResults(c)
	if err != nil {
		return fmt.Errorf("c.prepScanResults: %w", err)
	}
	return nil
}

func (rc *roamContext) prepScanResults(c *wpac.Client) error {
	aps, err := c.ScanResults(rc.ssid)
	if err != nil {
		return fmt.Errorf("c.ScanResults: %w", err)
	}
	rc.scoredAPs = scoreAll(aps, rc.cfg)
	for _, ap := range rc.scoredAPs {
		if ap.bssid == rc.lastKnown.BSSID {
			rc.currentAP = ap
		}
	}
	logScoredAPs(rc)
	hash := hashBSSIDs(aps[0:min(len(aps), rc.cfg.MaxBSSCt)])
	rc.scanState.mu.Lock()
	rc.scanState.bssListStable = hash == rc.scanState.bssidHash
	stable := rc.scanState.bssListStable
	rc.scanState.mu.Unlock()
	slog.Debug("bssListStable checked", "bool", stable)
	if len(rc.scoredAPs) == 0 {
		slog.Warn("scoredAPs is empty")
		return nil
	}
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

func getFreqsByScore(aps []scoredBSS) []int {
	var freqsRaw []int
	for _, ap := range aps {
		freqsRaw = append(freqsRaw, ap.freq)
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
	slog.Info("Most recent scan data")
	for _, a := range rc.scoredAPs {
		if a.bssid == rc.lastKnown.BSSID {
			slog.Info(blue.Render("current ap"), "bss", a)
		} else {
			slog.Info("candidate ap", "bss", a)
		}
	}
}
