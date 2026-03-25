package roam

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jwil007/roamctl/internal/config"
	"github.com/jwil007/roamctl/internal/wpac"
)

func Proc(c *wpac.Client, ctx context.Context, cfg *config.Config) error {
	slog.Info("Starting roamctl... exit with ctrl+c")
	rc := &roamContext{}
	rc.cfg = cfg
	rc.scanState.cond = sync.NewCond(&rc.scanState.mu)
	slog.Info("Setting wpa_supplicant configuration")
	cleanup, err := rc.handleWpaSuppConfig(c)
	if err != nil {
		return fmt.Errorf("handleWpaSuppConfig: %w", err)
	}
	defer cleanup() //sets wpa_supplicant back to original state
	slog.Info("Current SSID",
		"ssid", rc.ssid)
	slog.Info("Running full channel scan...")
	err = rc.runFullScan(c, ctx)
	if err != nil && !errors.Is(err, ErrScanRetryLimit) {
		return fmt.Errorf("rc.runFullScan: %w", err)
	}
	//Start polling signal stats
	slog.Info("Starting signal polling...")
	sigCh, sigErrCh := c.PollSignal(ctx, cfg.Timing.SigPollInterval)
	var scErrCh <-chan error
	cadenceTicker := time.NewTicker(cfg.BGScanInterval)
	defer cadenceTicker.Stop()
	for {
		select {
		case <-cadenceTicker.C:
			rc.scanState.mu.RLock()
			inProgress := rc.scanState.scanInProgress
			mode := rc.scanState.scanMode
			rc.scanState.mu.RUnlock()
			if !inProgress {
				if mode != noScan {
					slog.Info("Running background scan", "scan_mode", mode)
					scErrCh = rc.runScanConcurrent(c, ctx)
				} else {
					slog.Debug("Skipping background scan",
						"scan_mode", mode)
				}
			} else {
				slog.Info("Backgound scan skipped - scan already in progress")
			}
		case con := <-sigCh:
			if con.AvgRSSIBeacon != 0 {
				con.RSSI = con.AvgRSSIBeacon
			}
			if con.BSSID != "" && con.RSSI < -1 {
				rc.lastKnown = &con
			}
			if rc.lastKnown == nil {
				slog.Debug("last polled signal stats nil, check again next cycle")
				continue
			}
			//slog.Debug("Last polled connection status", "stats", rc.lastKnown)
			rc.evalTier()
			if rc.lastKnown.RSSI >= rc.cfg.FairRSSI+rc.cfg.RSSIHysteresisUp &&
				(rc.entryScanned || rc.entryScannedCrit) {
				slog.Info("Signal recovered - resetting entry scan flags",
					"rssi", rc.lastKnown.RSSI)
				rc.entryScanned = false
				rc.entryScannedCrit = false
			}
			if rc.roamingTier == opportunistic {
				err = rc.handleOppRoam(c, ctx)
				if err != nil {
					return fmt.Errorf("handleOppRoam: %w", err)
				}
			}
			if rc.roamingTier == active {
				err = rc.handleActiveRoam(c, ctx)
				if err != nil {
					return fmt.Errorf("handleActiveRoam: %w", err)
				}
			}
			if rc.roamingTier == critical {
				err = rc.handleCriticalRoam(c, ctx)
				if err != nil {
					return fmt.Errorf("handleCriticalRoam: %w", err)
				}
			}
		case err = <-sigErrCh:
			if err != nil {
				return fmt.Errorf("c.PollSignal: %w", err)
			}
		case err = <-scErrCh:
			if err != nil && !errors.Is(err, ErrScanRetryLimit) {
				return fmt.Errorf("rc.runScanConcurrent: %w", err)
			}
		}
	}
}

func (rc *roamContext) evalTier() {
	prevTier := rc.roamingTier
	switch {
	case rc.lastKnown.RSSI >= rc.cfg.ExcellentRSSI+rc.tierUpBuffer(noRoam):
		rc.roamingTier = noRoam
		rc.scanState.mu.Lock()
		if rc.scanState.scanMode != fullScan {
			rc.scanState.scanMode = noScan
		}
		rc.scanState.mu.Unlock()
		slog.Debug("roaming tier noRoam",
			"rssi", rc.lastKnown.RSSI)
	case rc.lastKnown.RSSI >= rc.cfg.FairRSSI+rc.tierUpBuffer(opportunistic):
		rc.roamingTier = opportunistic
		rc.scanState.mu.Lock()
		if rc.scanState.scanMode != fullScan {
			rc.scanState.scanMode = fastScan
		}
		rc.scanState.mu.Unlock()
		slog.Debug("roaming tier opportunistic",
			"rssi", rc.lastKnown.RSSI)
	case rc.lastKnown.RSSI >= rc.cfg.DegradedRSSI+rc.tierUpBuffer(active):
		rc.roamingTier = active
		rc.scanState.mu.Lock()
		if rc.scanState.scanMode != fullScan {
			rc.scanState.scanMode = fastScan
		}
		rc.scanState.mu.Unlock()
		slog.Debug("roaming tier active",
			"rssi", rc.lastKnown.RSSI)
	default: //Anything lower than degraded RSSI is critical
		rc.roamingTier = critical
		rc.scanState.mu.Lock()
		if rc.scanState.scanMode != fullScan {
			rc.scanState.scanMode = fastScan
		}
		rc.scanState.mu.Unlock()
		slog.Debug("roaming tier critical",
			"rssi", rc.lastKnown.RSSI)
	}
	if rc.roamingTier != prevTier {
		slog.Info("Roaming tier changed",
			"from", prevTier,
			"to", rc.roamingTier,
			"rssi", rc.lastKnown.RSSI)
	}
}

func (rc *roamContext) tierUpBuffer(evalTier roamingTier) int {
	if rc.roamingTier > evalTier {
		slog.Debug("Tier hysteresis in effect")
		return rc.cfg.TierHysteresis
	}
	return 0
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
