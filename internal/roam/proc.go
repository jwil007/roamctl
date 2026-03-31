package roam

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/jwil007/roamctl/internal/config"
	"github.com/jwil007/roamctl/internal/wpac"
)

func Proc(c *wpac.Client, ctx context.Context, cfg *config.Config) error {
	slog.Info("Starting roamctl... exit with ctrl+c")
	rc := &roamContext{}
	rc.roamingTier = noRoam
	rc.lastRoamAttempt = time.Now()
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
	slog.Info("Selected interface",
		"iface", c.Iface)
	slog.Info("Running full channel scan...")
	err = rc.runFullScan(c, ctx)
	if err != nil && !errors.Is(err, ErrScanRetryLimit) {
		return fmt.Errorf("rc.runFullScan: %w", err)
	}
	rc.lastEvalTime = time.Now() //set lastEvalTime - prevents the first roam attempt from using the initial scan
	//Start polling signal stats
	slog.Info("Starting signal polling...")
	sigCh, sigErrCh := c.PollSignal(ctx, cfg.Timing.SigPollInterval)
	var scErrCh <-chan error
	cadenceTicker := time.NewTicker(cfg.BGScanInterval)
	defer cadenceTicker.Stop()
	for {
		select {
		case <-cadenceTicker.C:
			if time.Since(rc.lastConnChange) <= cfg.ConnectionCooldown {
				slog.Debug("Background scan skipped, connection cooldown active")
				continue
			}
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
			var prevBSSID string
			var prevSSID string
			var prevFreq int
			if rc.lastKnown != nil {
				prevBSSID = rc.lastKnown.BSSID
				prevSSID = rc.lastKnown.SSID
				prevFreq = rc.lastKnown.Freq
			}
			if con.BSSID != "" && con.RSSI < -1 {
				rc.lastKnown = &con
			}
			if rc.lastKnown == nil {
				slog.Debug("last polled signal stats nil, check again next cycle")
				continue
			}
			if rc.lastKnown.SSID != prevSSID && prevSSID != "" {
				return fmt.Errorf("SSID change detected, exiting."+
					" prev_ssid: %v, new_ssid: %v", prevSSID, rc.lastKnown.SSID)
			}
			if rc.lastKnown.BSSID != prevBSSID && prevBSSID != "" {
				slog.Info("Connection change detected",
					"prev_bssid", prevBSSID,
					"new_bssid", rc.lastKnown.BSSID,
					"cooldown", rc.cfg.ConnectionCooldown)
				rc.onConnectionChange()
			}
			if con.WPAState != "COMPLETED" {
				slog.Info("wpa_state not COMPLETED, skipping poll",
					"wpa_state", con.WPAState)
				continue
			}
			if rc.lastKnown.Freq != prevFreq && prevFreq != 0 {
				slog.Info("Frequency change detected",
					"prev_freq", prevFreq, "new_freq", rc.lastKnown.Freq)
				rc.scanState.mu.Lock()
				if !slices.Contains(rc.scanState.channels, rc.lastKnown.Freq) {
					rc.scanState.channels = append(rc.scanState.channels, rc.lastKnown.Freq)
					slices.Sort(rc.scanState.channels)
				}
				rc.scanState.mu.Unlock()
			}
			if con.RSSI >= -1 {
				slog.Debug("Invalid RSSI, skipping poll", "rssi", con.RSSI)
				continue
			}
			rc.lastKnown.RSSI = rc.smoothRSSI(con.RSSI)
			slog.Debug("Last polled connection status", "stats", rc.lastKnown)
			if time.Since(rc.lastConnChange) <= cfg.ConnectionCooldown {
				slog.Debug("Connection cooldown in effect",
					"remaining", cfg.ConnectionCooldown-time.Since(rc.lastConnChange))
				continue
			}
			if time.Since(rc.lastRoamAttempt) >= 2*time.Second {
				rc.checkConnectionHealth()
			} else {
				slog.Debug("checkConnectionHealth skipped, backoff timer",
					"time remaining", 2*time.Second-time.Since(rc.lastRoamAttempt))
				continue
			}
			rc.evalTier()
			if rc.lastKnown.RSSI >= rc.cfg.FairRSSI+rc.cfg.TierHysteresis &&
				(rc.entryScanned || rc.entryScannedCrit) &&
				!rc.unhealthyConn {
				slog.Info("Signal recovered - resetting entry scan flags",
					"rssi", rc.lastKnown.RSSI)
				rc.entryScanned = false
				rc.entryScannedCrit = false
				rc.fullScannedCrit = false
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
				if errors.Is(err, os.ErrDeadlineExceeded) {
					slog.Warn("Signal poll read deadline exceeded")
					continue
				} else {
					return fmt.Errorf("c.PollSignal: %w", err)
				}
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
	if rc.unhealthyConn {
		if !rc.unhealthyLogged {
			slog.Info("Tier degraded to critical, unhealthy connection",
				"retry_rate", rc.lastKnown.RetryRate,
				"retry_limit", rc.cfg.RetryRate,
				"data_bitrate", max(rc.lastKnown.TxBitrate, rc.lastKnown.RxBitrate),
				"dr_limit", rc.cfg.DataRate*1000000)
			rc.unhealthyLogged = true
		}
		rc.roamingTier = critical
		rc.scanState.mu.Lock()
		if rc.scanState.scanMode != fullScan {
			rc.scanState.scanMode = fastScan
		}
		rc.scanState.mu.Unlock()
		return
	}
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
		if rc.roamingTier < prevTier {
			slog.Info("Tier improved — hysteresis threshold cleared",
				"from", prevTier,
				"to", rc.roamingTier,
				"rssi", rc.lastKnown.RSSI)
		} else {
			slog.Info("Tier degraded",
				"from", prevTier,
				"to", rc.roamingTier,
				"rssi", rc.lastKnown.RSSI)
		}
	}
}

func (rc *roamContext) tierUpBuffer(evalTier roamingTier) int {
	if rc.roamingTier > evalTier {
		slog.Debug("Tier hysteresis in effect")
		return rc.cfg.TierHysteresis
	}
	return 0
}

func (rc *roamContext) checkConnectionHealth() {
	legacyRates := []int{1000000, 2000000, 5500000, 6000000, 9000000, 11000000,
		12000000, 18000000, 24000000, 36000000, 48000000, 54000000}
	if rc.lastKnown.TxBitrate < 1000000 || rc.lastKnown.RxBitrate < 1000000 {
		slog.Debug("Invalid bitrate, skipping connection health check",
			"tx_bitrate", rc.lastKnown.TxBitrate,
			"rx_bitrate", rc.lastKnown.RxBitrate)
		rc.unhealthyConn = false
		return
	}
	if slices.Contains(legacyRates, max(rc.lastKnown.TxBitrate, rc.lastKnown.RxBitrate)) {
		slog.Debug("Device using legacy rates, skipping connection health check",
			"tx_bitrate", rc.lastKnown.TxBitrate,
			"rx_bitrate", rc.lastKnown.RxBitrate)
		rc.unhealthyConn = false
		return
	}
	if rc.lastKnown.RetryRate >= rc.cfg.RetryRate ||
		max(rc.lastKnown.TxBitrate, rc.lastKnown.RxBitrate) <= rc.cfg.DataRate*1000000 {
		slog.Debug("Current connection unhealthy",
			"retry_rate", rc.lastKnown.RetryRate,
			"retry_limit", rc.cfg.RetryRate,
			"data_bitrate", max(rc.lastKnown.TxBitrate, rc.lastKnown.RxBitrate),
			"dr_limit", rc.cfg.DataRate*1000000)
		rc.unhealthyConn = true
		return
	}
	rc.unhealthyConn = false
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

func (rc *roamContext) smoothRSSI(rssi int) int {
	if len(rc.rssiRingBuffer) < rc.cfg.RSSISmoothWindow {
		rc.rssiRingBuffer = append(rc.rssiRingBuffer, rssi)
	} else {
		rc.rssiRingBuffer[rc.rssiWriteIdx] = rssi
	}
	rc.rssiWriteIdx = (rc.rssiWriteIdx + 1) % rc.cfg.RSSISmoothWindow
	total := 0
	for _, r := range rc.rssiRingBuffer {
		total += r
	}
	smoothed := total / len(rc.rssiRingBuffer)
	slog.Debug("rssi smoothing stats:",
		"buffer", rc.rssiRingBuffer,
		"avg_rssi", total/len(rc.rssiRingBuffer),
	)
	return smoothed
}

func (rc *roamContext) onConnectionChange() {
	rc.lastConnChange = time.Now()
	rc.rssiRingBuffer = nil
	rc.unhealthyConn = false
	rc.unhealthyLogged = false
}
