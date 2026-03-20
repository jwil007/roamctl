package wpac

import (
	"fmt"

	"github.com/mdlayher/wifi"
)

func getStationInfo(iface string) (STAInfo, error) {
	var ifi *wifi.Interface
	var idx int
	c, err := wifi.New()
	if err != nil {
		return STAInfo{}, fmt.Errorf("wifi.New: %w", err)
	}
	defer func() {
		_ = c.Close()
	}()
	ifis, err := c.Interfaces()
	if err != nil {
		return STAInfo{}, fmt.Errorf("c.Interfaces: %w", err)
	}
	for _, n := range ifis {
		if n.Name == iface {
			ifi = n
		}
	}
	if ifi != nil {
		idx = ifi.Index
	}
	sl, err := c.StationInfo(ifi)
	if err != nil {
		return STAInfo{}, fmt.Errorf("c.StationInfo: %w", err)
	}
	for _, si := range sl {
		if si.InterfaceIndex == idx {
			return STAInfo{
				RxBitrate:    si.ReceiveBitrate,
				TxBitrate:    si.TransmitBitrate,
				TxRetries:    si.TransmitRetries,
				TxFails:      si.TransmitFailed,
				RetryRate:    retryRate(si),
				BeaconLoss:   si.BeaconLoss,
				SignalAvg:    si.SignalAverage,
				ConnDuration: si.Connected,
				BSSID:        si.HardwareAddr.String(),
			}, nil
		}
	}
	return STAInfo{}, nil
}

func retryRate(s *wifi.StationInfo) int {
	if s.TransmittedPackets == 0 {
		return 0
	}
	if s.TransmitRetries > s.TransmittedPackets {
		return 100
	}
	rr := s.TransmitRetries * 100 / s.TransmittedPackets
	return rr
}
