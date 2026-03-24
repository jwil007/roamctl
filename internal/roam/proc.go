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
	rc := &roamContext{}
	rc.cfg = cfg
	rc.scanState.cond = sync.NewCond(&rc.scanState.mu)
	err := rc.runFullScan(c, ctx)
	if err != nil && !errors.Is(err, ErrScanRetryLimit) {
		return fmt.Errorf("rc.runFullScan: %w", err)
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
			slog.Debug("Last polled connection status", "stats", rc.lastKnown)
			rc.evalTier()
			if rc.roamingTier == opportunistic || rc.roamingTier == noRoam {
				slog.Debug("Clearing entryScanned flag")
				rc.entryScanned = false
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
				return fmt.Errorf("rc.runScanLoop: %w", err)
			}
		}
	}
}

func (rc *roamContext) evalTier() {
	switch {
	case rc.lastKnown.RSSI >= rc.cfg.ExcellentRSSI:
		rc.roamingTier = noRoam
	case rc.lastKnown.RSSI >= rc.cfg.OpportunisticRSSI:
		rc.roamingTier = opportunistic
	case rc.lastKnown.RSSI >= rc.cfg.ActiveRSSI:
		rc.roamingTier = active
	default:
		rc.roamingTier = critical
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
