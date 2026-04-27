package roam

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jwil007/roamctl/internal/wpac"
)

func (rc *roamContext) monitorExternalEvents(
	c *wpac.Client,
	ctx context.Context) {
	evCh, errCh := c.WatchForEvents(ctx)
	// init local vars
	timeout := time.NewTimer(20 * time.Second)
	timeout.Stop()
	extScanRunning := false
	var scanStart time.Time
	var btmUsed bool
	var beaconLoss bool
	var watchingRoam bool
	var roamStart time.Time
	var targetBSSID string
	var finalBSSID string
	var message string
	// function log roam state and reset local vars
	wrapUp := func() {
		rc.lastRoamStats.TargetBSSID = targetBSSID
		rc.lastRoamStats.FinalBSSID = finalBSSID
		rc.lastRoamStats.Duration = time.Since(roamStart)
		rc.lastRoamStats.Message = message
		rc.lastRoamStats.CompletedAt = time.Now()
		targetBSSID = ""
		finalBSSID = ""
		message = "Spontaneous roam (not triggered by roamctl)"
		btmUsed = false
		beaconLoss = false
		watchingRoam = false
		rc.roamInProgress = false
		rc.updateSnapshot()
	}
	for {
		select {
		case <-ctx.Done():
			rc.scanState.cond.Broadcast()
			return
		case <-timeout.C:
			slog.Warn("External scan timed out")
			extScanRunning = false
			rc.scanState.mu.Lock()
			rc.scanState.scanInProgress = false
			rc.scanState.cond.Broadcast()
			rc.scanState.mu.Unlock()
		case ev := <-evCh:
			// logic for external scans
			if strings.Contains(ev, "CTRL-EVENT-SCAN-STARTED") {
				rc.scanState.mu.RLock()
				inProg := rc.scanState.scanInProgress
				rc.scanState.mu.RUnlock()
				if !inProg {
					scanStart = time.Now()
					timeout.Reset(20 * time.Second)
					extScanRunning = true
					rc.scanState.mu.Lock()
					rc.scanState.scanInProgress = true
					rc.scanState.scanMode = external
					rc.scanState.mu.Unlock()
					slog.Info("Scan started externally")
					rc.updateSnapshot()
				}
			}
			if extScanRunning {
				if strings.Contains(ev, "CTRL-EVENT-SCAN-RESULTS") {
					slog.Info("External scan finished, processing results...")
					timeout.Stop()
					rc.scanState.mu.Lock()
					rc.scanState.scanInProgress = false
					rc.scanState.lastScanTime = time.Now()
					rc.scanState.scanDuration = time.Since(scanStart)
					rc.scanState.cond.Broadcast()
					rc.scanState.mu.Unlock()
					extScanRunning = false
					rc.updateSnapshot()
					err := rc.prepScanResults(c)
					if err != nil {
						slog.Error(err.Error())
					}
				}
			}
			// logic for external roams
			if strings.Contains(ev, "CTRL-EVENT-BEACON-LOSS") {
				slog.Warn("Beacon loss detected")
				beaconLoss = true
			}
			if strings.Contains(ev, "WNM: ") {
				slog.Info("BTM message received")
				btmUsed = true
			}
			if strings.Contains(ev, "SME: Trying to authenticate with ") {
				//don't log roamctl initiated roams
				if rc.roamInProgress {
					continue
				}
				slog.Info("Detected roam started externally")
				rc.roamInProgress = true
				rc.updateSnapshot()
				roamStart = time.Now()
				flds := strings.Fields(ev)
				for _, fld := range flds {
					if wpac.IsMACAddress(fld) {
						targetBSSID = fld
					}
				}
				watchingRoam = true
			}
			if watchingRoam {
				if beaconLoss {
					message = "Roam triggered by beacon loss"
				}
				if btmUsed {
					message = "Roam triggered by BSS transition mgmt (802.11v)"
				}
				if strings.Contains(ev, "CTRL-EVENT-CONNECTED") {
					slog.Info("External roam completed")
					flds := strings.Fields(ev)
					for _, fld := range flds {
						if wpac.IsMACAddress(fld) {
							finalBSSID = fld
						}
					}
					var suc bool
					if targetBSSID == finalBSSID {
						rc.lastRoamStats.Success = true
						rc.roamResultFlag = success
						rc.onConnectionChange()
					} else {
						rc.lastRoamStats.Success = false
						rc.roamResultFlag = failure
					}
					rc.lastRoamStats.Success = suc
					wrapUp()
					continue
				}
				if strings.Contains(ev, "CTRL-EVENT-DISCONNECTED") {
					slog.Warn("External roam failed")
					finalBSSID = ""
					rc.roamResultFlag = failure
					rc.lastRoamStats.Success = false
					wrapUp()
					continue
				}
			}
		case err := <-errCh:
			slog.Error(err.Error())
		}
	}
}
