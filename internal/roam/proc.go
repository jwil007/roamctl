package roam

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jwil007/roamctl/internal/config"
	"github.com/jwil007/roamctl/internal/ipc"
	"github.com/jwil007/roamctl/internal/wpac"
)

func Proc(
	c *wpac.Client, ctx context.Context,
	cfg *config.Config,
	ipcChan chan ipc.ProcessState) error {

	//init roamContext - struct that holds all state for roamctl
	slog.Info("Starting roamctl... exit with ctrl+c")
	rc := &roamContext{}
	rc.iface = c.Iface
	rc.ipcChan = ipcChan
	rc.richByBSSID = make(map[string]wpac.RichBSS)
	rc.roamingTier = noRoam
	rc.lastRoamAttempt = time.Now()
	rc.cfg = cfg
	rc.scanState.cond = sync.NewCond(&rc.scanState.mu)
	cs, err := constructConnStatus(c)
	if err != nil {
		return fmt.Errorf("wpac.GetConnectionStatus:%w", err)
	}
	rc.lastKnown = &cs

	//handle wpa_supplicant configuration. Disables bgscan and btm
	slog.Info("Setting wpa_supplicant configuration")
	cleanup, err := rc.handleWpaSuppConfig(c)
	if err != nil {
		return fmt.Errorf("handleWpaSuppConfig: %w", err)
	}
	defer cleanup() //sets wpa_supplicant back to original state

	//initialize SSID name and iface, needed for all scanning/roaming operations.
	//Exit if SSID name is blank (not connected)
	slog.Info("Current SSID",
		"ssid", rc.ssid)
	if rc.ssid == "" {
		return fmt.Errorf("SSID not connected, exiting")
	}
	slog.Info("Selected interface",
		"iface", c.Iface)

	//roamctl runs a full channel scan at startup
	slog.Info("Running full channel scan...")
	err = rc.runFullScan(c, ctx)
	if err != nil && !errors.Is(err, ErrScanRetryLimit) {
		return fmt.Errorf("rc.runFullScan: %w", err)
	}
	err = rc.prepScanResults(c)
	if err != nil {
		return fmt.Errorf("prepScanResults: %w", err)
	}
	rc.lastEvalTime = time.Now()
	rc.updateSnapshot()

	//Start exporter for IPC. Runs as goroutine
	rc.ipcShipper(ctx)

	//Start polling signal stats
	slog.Info("Starting signal polling...")
	//Start external roam monitor
	go rc.monitorExternalRoams(c, ctx)
	sigCh, sigErrCh := pollSignal(c, ctx, cfg.Timing.SigPollInterval)
	var scErrCh <-chan error
	cadenceTicker := time.NewTicker(cfg.BGScanInterval)
	defer cadenceTicker.Stop()
	for {
		select {
		//background scan on timer (cadenceTicker)
		case <-cadenceTicker.C:
			if time.Since(rc.lastConnChange) <= cfg.ConnectionCooldown {
				slog.Debug(
					"Background scan skipped, connection cooldown active")
				continue
			}
			rc.scanState.mu.RLock()
			inProgress := rc.scanState.scanInProgress
			mode := rc.scanState.scanMode
			rc.scanState.mu.RUnlock()
			if !inProgress {
				if mode != noScan {
					slog.Info("Running background scan",
						"scan_mode", mode)
					scErrCh = rc.runScanConcurrent(c, ctx)
				} else {
					slog.Debug("Skipping background scan",
						"scan_mode", mode)
				}
			} else {
				slog.Info("Backgound scan skipped" +
					" - scan already in progress")
			}

		//handle updated signal reading. This dispatches roam logic
		//many guards are in place so roam is only dispatched when needed
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
				slog.Debug("last polled signal stats nil," +
					" check again next cycle")
				rc.updateSnapshot()
				continue
			}
			if rc.wpaDisconnect {
				if con.WPAState == "COMPLETED" {
					slog.Info("Reconnected, handling wpa_supp config")
					_, err = rc.handleWpaSuppConfig(c)
					if err != nil {
						if strings.Contains(err.Error(), "no connected ssid") {
							continue
						}
						return fmt.Errorf("handleWpaSuppConfig: %w", err)
					}
					rc.wpaDisconnect = false
				}
			}
			if rc.lastKnown.SSID != prevSSID && prevSSID != "" {
				rc.updateSnapshot()
				_, err = rc.handleWpaSuppConfig(c)
				if err != nil {
					return fmt.Errorf("handleWpaSuppConfig: %w", err)
				}
				slog.Info("SSID change detected",
					"prev_ssid", "new_ssid", prevSSID, rc.lastKnown.SSID)
				//os.Exit(1)
			}
			if rc.lastKnown.BSSID != prevBSSID && prevBSSID != "" {
				slog.Info("Connection change detected",
					"prev_bssid", prevBSSID,
					"new_bssid", rc.lastKnown.BSSID,
					"cooldown", rc.cfg.ConnectionCooldown)
				rc.onConnectionChange()
			}
			if con.WPAState != "COMPLETED" {
				if con.WPAState == "DISCONNECTED" {
					rc.updateSnapshot()
					slog.Warn("wpa_state is DISCONNECTED")
					rc.wpaDisconnect = true
				}
				slog.Debug("wpa_state not COMPLETED, skipping poll",
					"wpa_state", con.WPAState)
				rc.updateSnapshot()
				continue
			}
			if rc.lastKnown.Freq != prevFreq && prevFreq != 0 {
				slog.Info("Frequency change detected",
					"prev_freq", prevFreq, "new_freq", rc.lastKnown.Freq)
				rc.scanState.mu.Lock()
				if !slices.Contains(rc.scanState.channels, rc.lastKnown.Freq) {
					rc.scanState.channels = append(
						rc.scanState.channels, rc.lastKnown.Freq)
					slices.Sort(rc.scanState.channels)
				}
				rc.scanState.mu.Unlock()
			}
			if con.RSSI >= -1 {
				slog.Debug(
					"Invalid RSSI, skipping poll", "rssi", con.RSSI)
				rc.updateSnapshot()
				continue
			}
			rssi := con.AvgRSSIBeacon
			if con.AvgRSSIBeacon >= -1 {
				rssi = con.RSSI
			}
			rc.lastKnown.RSSI = rc.smoothRSSI(rssi)
			slog.Debug(
				"Last polled connection status",
				"stats", rc.lastKnown)
			if time.Since(rc.lastConnChange) <= cfg.ConnectionCooldown {
				slog.Debug("Connection cooldown in effect",
					"remaining",
					cfg.ConnectionCooldown-time.Since(rc.lastConnChange))
				rc.updateSnapshot()
				continue
			}
			if time.Since(rc.lastRoamAttempt) >= 2*time.Second {
				rc.checkConnectionHealth()
			} else {
				slog.Debug("checkConnectionHealth skipped, backoff timer",
					"time remaining",
					2*time.Second-time.Since(rc.lastRoamAttempt))
				rc.updateSnapshot()
				continue
			}
			//rssi hysteresis to prevent ping-pong roams at borderline
			rc.checkRSSIHysteresis()
			if rc.hysteresisActive {
				rc.updateSnapshot()
				continue
			}
			//set roaming tier
			rc.evalTier()
			//tier based hysteresis to prevent thrash at borderline readings
			if rc.lastKnown.RSSI >= rc.cfg.FairRSSI+rc.cfg.TierHysteresis &&
				(rc.entryScanned || rc.entryScannedCrit) &&
				!rc.unhealthyConn {
				slog.Info("Signal recovered - resetting entry scan flags",
					"rssi", rc.lastKnown.RSSI)
				rc.entryScanned = false
				rc.entryScannedCrit = false
				rc.fullScannedCrit = false
			}
			//dispatch roam code path based on roaming tier
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
			rc.updateSnapshot() //update IPC on every poll

		//error handling from scan and sig poll channels
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

func (rc *roamContext) handleWpaSuppConfig(c *wpac.Client) (func(), error) {
	//Get Current wpa_supplicant status
	storedConf, err := c.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("c.GetConfig: %v", err)
	}

	//toggle btm (802.11v) based on user config
	btmSetting := "0"
	if !rc.cfg.EnableBTM {
		btmSetting = "1"
	}
	//Disable bgscan to prevent autonomous roaming
	noRoamConfig := wpac.WPAConfig{
		SSID:       storedConf.SSID,
		NetworkID:  storedConf.NetworkID,
		BGScan:     "",
		DisableBTM: btmSetting,
	}
	//Only change config if needed
	if storedConf.BGScan != noRoamConfig.BGScan ||
		storedConf.DisableBTM != noRoamConfig.DisableBTM {
		err = c.SetConfig(noRoamConfig)
		if err != nil {
			return nil, fmt.Errorf("c.SetConfig: %w", err)
		}
	}
	rc.ssid = storedConf.SSID
	cleanup := func() {
		err = c.SetConfig(storedConf)
		if err != nil {
			slog.Error(
				"error restoring wpa_supplicant config",
				"value", err)
		}
	}
	return cleanup, nil
}

func (rc *roamContext) onConnectionChange() {
	rc.lastConnChange = time.Now()
	rc.rssiRingBuffer = nil
	rc.unhealthyConn = false
	rc.unhealthyLogged = false
}
