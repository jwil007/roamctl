package roam

import (
	"fmt"
	"hash/fnv"
	"log/slog"
	"slices"
	"sort"

	"github.com/jwil007/roamctl/internal/wpac"
)

func getFreqs(aps []wpac.RichBSS) []int {
	var freqsRaw []int
	for _, ap := range aps {
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

func (rc *roamContext) prepScanResults(c *wpac.Client) error {
	aps, err := c.ScanResults(rc.cfg.SSID)
	if err != nil {
		return fmt.Errorf("c.ScanResults: %w", err)
	}
	rc.scoredAPs = scoreAll(aps, rc.cfg)
	logScoredAPs(rc)
	for _, ap := range rc.scoredAPs {
		if ap.bssid == rc.lastKnown.BSSID {
			rc.currentAP = ap
		}
	}
	hash := hashBSSIDs(aps[0:min(len(aps), 15)])
	rc.scanState.mu.Lock()
	rc.scanState.bssListStable = hash == rc.scanState.bssidHash
	rc.scanState.bssidHash = hash
	rc.scanState.mu.Unlock()
	if rc.scoredAPs == nil {
		return fmt.Errorf("c.ScanResults: scoredAPs is nil")
	}
	rc.candidateAP = rc.scoredAPs[0]
	return nil
}
