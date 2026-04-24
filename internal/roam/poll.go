package roam

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/jwil007/roamctl/internal/netlink"
	"github.com/jwil007/roamctl/internal/wpac"
)

func pollSignal(
	c *wpac.Client,
	ctx context.Context,
	interval time.Duration) (<-chan ConnectionStatus, <-chan error) {
	connStatus := make(chan ConnectionStatus)
	errc := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			case <-ticker.C:
				s, err := constructConnStatus(c)
				if err != nil && !strings.Contains(
					err.Error(), "bssid field not found") {
					errc <- err
					return
				}
				connStatus <- s
			}
		}
	}()
	return connStatus, errc
}

func constructConnStatus(c *wpac.Client) (ConnectionStatus, error) {
	status, err := c.GetStatus()
	if err != nil {
		return ConnectionStatus{}, fmt.Errorf("c.getStatus: %w", err)
	}
	type staResult struct {
		info netlink.STAInfo
		err  error
	}
	ch := make(chan staResult, 1)
	go func() {
		info, err := netlink.GetStationInfo(c.Iface)
		ch <- staResult{info, err}
	}()

	var staInfo netlink.STAInfo
	select {
	case res := <-ch:
		if res.err != nil {
			// non-fatal, continue with zero STAInfo
			slog.Debug("GetStationInfo failed", "err", res.err)
		} else {
			staInfo = res.info
		}
	case <-time.After(5 * time.Second):
		slog.Warn("GetStationInfo timed out")
	}

	return ConnectionStatus{
		Status:  status,
		STAInfo: staInfo,
	}, nil
}

func (rc *roamContext) evalTier() {
	prevTier := rc.roamingTier
	if rc.unhealthyConn {
		if !rc.unhealthyLogged {
			slog.Info("Tier degraded to critical, unhealthy connection",
				"retry_rate", rc.lastKnown.RetryRate,
				"retry_limit", rc.cfg.RetryRate,
				"data_bitrate", max(
					rc.lastKnown.TxBitrate, rc.lastKnown.RxBitrate),
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
			if rc.scanState.scanMode == external &&
				!rc.scanState.scanInProgress {
				rc.scanState.scanMode = noScan
			}
			if rc.scanState.scanMode == fastScan &&
				!rc.scanState.scanInProgress {
				rc.scanState.scanMode = noScan
			}
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
	if slices.Contains(legacyRates, max(
		rc.lastKnown.TxBitrate, rc.lastKnown.RxBitrate)) {
		slog.Debug(
			"Device using legacy rates, skipping connection health check",
			"tx_bitrate", rc.lastKnown.TxBitrate,
			"rx_bitrate", rc.lastKnown.RxBitrate)
		rc.unhealthyConn = false
		return
	}
	if rc.lastKnown.RetryRate >= rc.cfg.RetryRate ||
		max(rc.lastKnown.TxBitrate, rc.lastKnown.RxBitrate) <=
			rc.cfg.DataRate*1000000 ||
		max(rc.lastKnown.TxMCS, rc.lastKnown.RxMCS) <= rc.cfg.MCSIndex {
		slog.Debug("Current connection unhealthy",
			"retry_rate", rc.lastKnown.RetryRate,
			"retry_limit", rc.cfg.RetryRate,
			"mcs_index", max(rc.lastKnown.TxMCS, rc.lastKnown.RxMCS),
			"mcs_limit", rc.cfg.MCSIndex,
			"data_bitrate", max(rc.lastKnown.TxBitrate, rc.lastKnown.RxBitrate),
			"dr_limit", rc.cfg.DataRate*1000000)
		rc.unhealthyConn = true
		return
	}
	rc.unhealthyConn = false
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
