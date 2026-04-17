package roam

import (
	"context"
	"time"

	"github.com/jwil007/roamctl/internal/ipc"
)

func (rc *roamContext) updateSnapshot() {
	bssList := rc.buildBSSForIPC()
	connState := rc.buildConnStateForIPC()
	roamStats := rc.buildRoamStatsForIPC()
	rc.scanState.mu.RLock()
	scS := ipc.ScanState{
		ScanInProgress: rc.scanState.scanInProgress,
		ScanMode:       rc.scanState.scanMode.String(),
		ScanDuration:   rc.scanState.scanDuration,
		BSSListStable:  rc.scanState.bssListStable,
	}
	rc.scanState.mu.RUnlock()

	newState := ipc.ProcessState{
		SSID:            rc.ssid,
		BSSList:         bssList,
		ConnState:       connState,
		RoamStats:       roamStats,
		RoamingTier:     rc.roamingTier.String(),
		RoamResultFlag:  rc.roamResultFlag.String(),
		LastTriggerRSSI: rc.lastTriggerRSSI,
		Flags: ipc.Flags{
			HysteresisActive: rc.hysteresisActive,
			EntryScanned:     rc.entryScanned,
			EntryScannedCrit: rc.entryScannedCrit,
			FullScannedCrit:  rc.fullScannedCrit,
			UnhealthyConn:    rc.unhealthyConn,
			RoamInProgress:   rc.roamInProgress,
		},
		ScanState: scS,
	}
	rc.snapshot.Store(&newState)
}

func (rc *roamContext) ipcShipper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(rc.cfg.SigPollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s := rc.snapshot.Load(); s != nil {
					select {
					case rc.ipcChan <- *s:
					default:
						// drop if channel full, stale data OK
					}
				}
			}
		}
	}()
}

func (rc *roamContext) buildConnStateForIPC() ipc.ConnState {
	return ipc.ConnState{
		SSID:          rc.lastKnown.SSID,
		WPAState:      rc.lastKnown.WPAState,
		RxBitrate:     rc.lastKnown.RxBitrate,
		RxMCS:         rc.lastKnown.RxMCS,
		RxPHY:         rc.lastKnown.RxPHY,
		TxBitrate:     rc.lastKnown.TxBitrate,
		TxMCS:         rc.lastKnown.TxMCS,
		TxPHY:         rc.lastKnown.TxPHY,
		TxRetries:     rc.lastKnown.TxRetries,
		RetryRate:     rc.lastKnown.RetryRate,
		TxFails:       rc.lastKnown.TxFails,
		BeaconLoss:    rc.lastKnown.BeaconLoss,
		RSSI:          rc.lastKnown.RSSI,
		AvgRSSI:       rc.lastKnown.AvgRSSI,
		AvgRSSIBeacon: rc.lastKnown.AvgRSSIBeacon,
		ConnDuration:  rc.lastKnown.ConnDuration,
		BSSID:         rc.lastKnown.BSSID,
		Freq:          rc.lastKnown.Freq,
		ChannelWidth:  rc.lastKnown.ChannelWidth,
	}
}

func (rc *roamContext) buildRoamStatsForIPC() ipc.RoamStats {
	return ipc.RoamStats{
		Success:     rc.lastRoamStats.Success,
		TargetBSSID: rc.lastRoamStats.TargetBSSID,
		FinalBSSID:  rc.lastRoamStats.FinalBSSID,
		Duration:    rc.lastRoamStats.Duration,
		Message:     rc.lastRoamStats.Message,
		CompletedAt: rc.lastRoamStats.CompletedAt,
	}
}

func (rc *roamContext) buildBSSForIPC() []ipc.BSS {
	clear(rc.richByBSSID)
	for _, r := range rc.richBSSList {
		rc.richByBSSID[r.BSSID] = r
	}
	var bssList []ipc.BSS
	for _, scored := range rc.scoredAPs {
		rich := rc.richByBSSID[scored.bssid]
		var isCurrentAP bool
		if rc.lastKnown.BSSID == scored.bssid {
			isCurrentAP = true
		} else {
			isCurrentAP = false
		}
		bssList = append(bssList, ipc.BSS{
			BSSID:        rich.BSSID,
			Freq:         rich.Freq,
			ChannelNum:   rich.ChannelNum,
			Band:         rich.Band.String(),
			ChannelWidth: rich.ChannelWidth.String(),
			PHYType:      rich.PHYType.String(),
			BeaconInt:    rich.BeaconInt,
			Noise:        rich.Noise,
			RSSI:         rich.RSSI,
			SNR:          rich.SNR,
			Age:          rich.Age,
			Flags:        rich.Flags,
			EstThruput:   rich.EstThruput,
			QBSSUtil:     rich.QBSSUtil,
			QBSSStaCt:    rich.QBSSStaCt,
			FinalScore:   scored.finalScore,
			RssiScore:    scored.rssiScore,
			SnrScore:     scored.snrScore,
			BandScore:    scored.bandScore,
			CwScore:      scored.cwScore,
			UtilScore:    scored.utilScore,
			PhyScore:     scored.phyScore,
			IsCurrentAP:  isCurrentAP,
		})
	}
	return bssList
}
