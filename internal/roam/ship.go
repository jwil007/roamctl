package roam

import (
	"github.com/jwil007/roamctl/internal/ipc"
)

func (rc *roamContext) shipProcessState(ch chan<- ipc.ProcessState) {
	bssList := rc.buildBSSForIPC()
	rc.scanState.mu.RLock()
	scS := ipc.ScanState{
		ScanInProgress: rc.scanState.scanInProgress,
		ScanMode:       rc.scanState.scanMode.String(),
		ScanDuration:   rc.scanState.scanDuration,
		BSSListStable:  rc.scanState.bssListStable,
	}
	rc.scanState.mu.RUnlock()
	ch <- ipc.ProcessState{
		SSID:           rc.ssid,
		BSSList:        bssList,
		ConnState:      *rc.lastKnown,
		RoamStats:      rc.lastRoamStats,
		RoamingTier:    rc.roamingTier.String(),
		RoamResultFlag: rc.roamResultFlag.String(),
		Flags: ipc.Flags{
			HysteresisActive: rc.hysteresisActive,
			EntryScanned:     rc.entryScanned,
			EntryScannedCrit: rc.entryScannedCrit,
			FullScannedCrit:  rc.fullScannedCrit,
			UnhealthyConn:    rc.unhealthyConn,
		},
		ScanState: scS,
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
