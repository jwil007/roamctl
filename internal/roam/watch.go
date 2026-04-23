package roam

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/jwil007/roamctl/internal/wpac"
)

func (rc *roamContext) monitorExternalRoams(
	c *wpac.Client,
	ctx context.Context) {
	evCh, errCh := c.WatchForEvents(ctx)
	// init local vars
	var btmUsed bool
	var beaconLoss bool
	var watchingRoam bool
	var start time.Time
	var targetBSSID string
	var finalBSSID string
	var message string
	// function log roam state and reset local vars
	wrapUp := func() {
		rc.lastRoamStats.TargetBSSID = targetBSSID
		rc.lastRoamStats.FinalBSSID = finalBSSID
		rc.lastRoamStats.Duration = time.Since(start)
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
			return
		case ev := <-evCh:
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
				start = time.Now()
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
