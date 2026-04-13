package netlink

import (
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
	"golang.org/x/sys/unix"
)

// New creates a new Client.
func New() (*Client, error) {
	c, err := newClient()
	if err != nil {
		return nil, err
	}

	return &Client{
		c: c,
	}, nil
}

func GetStationInfo(iface string) (STAInfo, error) {
	var ifi *Interface
	var idx int
	c, err := New()
	if err != nil {
		return STAInfo{}, fmt.Errorf("wifi.New: %w", err)
	}
	defer func() {
		_ = c.c.Close()
	}()
	ifis, err := c.c.Interfaces()
	if err != nil {
		return STAInfo{}, fmt.Errorf("c.Interfaces: %w", err)
	}
	for _, n := range ifis {
		if n.Name == iface {
			ifi = n
		}
	}
	var cw string
	var freq int
	if ifi != nil {
		idx = ifi.Index
		cw = ifi.ChannelWidth.String()
		freq = ifi.Frequency
	}
	sl, err := c.c.StationInfo(ifi)
	if err != nil {
		return STAInfo{}, fmt.Errorf("c.StationInfo: %w", err)
	}

	for _, si := range sl {
		//fmt.Printf("staInfo RSSI: %v, staInfo AvgRSSI: "+
		//	"%v staInfoBeaconRSSI: %v", si.Signal, si.SignalAverage,
		//	si.BeaconSignalAverage) //DEBUG
		if si.InterfaceIndex == idx {
			return STAInfo{
				RxBitrate:     si.ReceiveBitrate,
				RxMCS:         si.ReceiveMCS,
				RxPHY:         si.ReceivePHY,
				TxBitrate:     si.TransmitBitrate,
				TxMCS:         si.TransmitMCS,
				TxPHY:         si.TransmitPHY,
				TxRetries:     si.TransmitRetries,
				RetryRate:     retryRate(si),
				TxFails:       si.TransmitFailed,
				BeaconLoss:    si.BeaconLoss,
				RSSI:          si.Signal,
				AvgRSSI:       si.SignalAverage,
				AvgRSSIBeacon: si.BeaconSignalAverage,
				ConnDuration:  si.Connected,
				BSSID:         si.HardwareAddr.String(),
				Freq:          freq,
				ChannelWidth:  cw,
			}, nil
		}
	}
	return STAInfo{}, nil
}

func (c *client) Close() error { return c.c.Close() }

func newClient() (*client, error) {
	c, err := genetlink.Dial(nil)
	if err != nil {
		return nil, err
	}

	closeOnErr := true
	defer func() {
		if closeOnErr {
			_ = c.Close()
		}
	}()

	for _, o := range []netlink.ConnOption{
		netlink.ExtendedAcknowledge,
		netlink.GetStrictCheck,
	} {
		_ = c.SetOption(o, true)
	}
	cl, err := initClient(c)
	if err != nil {
		_ = c.Close()
		return nil, err
	}

	closeOnErr = false
	return cl, nil
}

func initClient(c *genetlink.Conn) (*client, error) {
	family, err := c.GetFamily(unix.NL80211_GENL_NAME)
	if err != nil {
		return nil, err
	}

	return &client{
		c:             c,
		familyID:      family.ID,
		familyVersion: family.Version,

		scan: sync.Mutex{},
	}, nil
}

func (c *client) get(
	cmd uint8,
	flags netlink.HeaderFlags,
	ifi *Interface,
	// May be nil; used to apply optional parameters.
	params func(ae *netlink.AttributeEncoder),
) ([]genetlink.Message, error) {
	ae := netlink.NewAttributeEncoder()
	ifi.encode(ae)
	if params != nil {
		// Optionally apply more parameters to the attribute encoder.
		params(ae)
	}

	// Note: don't send netlink.Acknowledge or we get an extra message back from
	// the kernel which doesn't seem useful as of now.
	return c.execute(cmd, flags, ae)
}

func (c *client) Interfaces() ([]*Interface, error) {
	// Ask nl80211 to dump a list of all Wi-Fi interfaces
	msgs, err := c.get(
		unix.NL80211_CMD_GET_INTERFACE,
		netlink.Dump,
		nil,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return ParseInterfaces(msgs)
}

func (ifi *Interface) encode(ae *netlink.AttributeEncoder) {
	if ifi == nil {
		return
	}

	// Mandatory.
	ae.Uint32(unix.NL80211_ATTR_IFINDEX, uint32(ifi.Index))
}

func ParseInterfaces(msgs []genetlink.Message) ([]*Interface, error) {
	ifis := make([]*Interface, 0, len(msgs))
	for _, m := range msgs {
		attrs, err := netlink.UnmarshalAttributes(m.Data)
		if err != nil {
			return nil, err
		}

		var ifi Interface
		if err := (&ifi).parseAttributes(attrs); err != nil {
			return nil, err
		}

		ifis = append(ifis, &ifi)
	}

	return ifis, nil
}

func (ifi *Interface) parseAttributes(attrs []netlink.Attribute) error {
	for _, a := range attrs {
		switch a.Type {
		case unix.NL80211_ATTR_IFINDEX:
			ifi.Index = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_ATTR_IFNAME:
			ifi.Name = nlenc.String(a.Data)
		case unix.NL80211_ATTR_MAC:
			ifi.HardwareAddr = a.Data
		case unix.NL80211_ATTR_WIPHY:
			ifi.PHY = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_ATTR_IFTYPE:
			// NOTE: InterfaceType copies the ordering of nl80211's interface type
			// constants.  This may not be the case on other operating systems.
			ifi.Type = InterfaceType(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_ATTR_WDEV:
			ifi.Device = int(binary.NativeEndian.Uint64(a.Data))
		case unix.NL80211_ATTR_WIPHY_FREQ:
			ifi.Frequency = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_ATTR_CHANNEL_WIDTH:
			ifi.ChannelWidth = ChannelWidth(binary.NativeEndian.Uint32(a.Data))
		}
	}

	return nil
}

func (c *client) execute(
	cmd uint8,
	flags netlink.HeaderFlags,
	ae *netlink.AttributeEncoder,
) ([]genetlink.Message, error) {
	b, err := ae.Encode()
	if err != nil {
		return nil, err
	}

	return c.c.Execute(
		genetlink.Message{
			Header: genetlink.Header{
				Command: cmd,
				Version: c.familyVersion,
			},
			Data: b,
		},
		// Always pass the genetlink family ID and request flag.
		c.familyID,
		netlink.Request|flags,
	)
}

func (c *client) StationInfo(ifi *Interface) ([]*StationInfo, error) {
	msgs, err := c.get(
		unix.NL80211_CMD_GET_STATION,
		netlink.Dump,
		ifi,
		func(ae *netlink.AttributeEncoder) {
			if ifi.HardwareAddr != nil {
				ae.Bytes(unix.NL80211_ATTR_MAC, ifi.HardwareAddr)
			}
		},
	)
	if err != nil {
		return nil, err
	}

	stations := make([]*StationInfo, len(msgs))
	for i := range msgs {
		if stations[i], err = ParseStationInfo(msgs[i].Data); err != nil {
			return nil, err
		}
	}

	return stations, nil
}

func ParseStationInfo(b []byte) (*StationInfo, error) {
	attrs, err := netlink.UnmarshalAttributes(b)
	if err != nil {
		return nil, err
	}

	var info StationInfo
	for _, a := range attrs {
		switch a.Type {
		case unix.NL80211_ATTR_IFINDEX:
			info.InterfaceIndex = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_ATTR_MAC:
			info.HardwareAddr = a.Data
		case unix.NL80211_ATTR_STA_INFO:
			nattrs, err := netlink.UnmarshalAttributes(a.Data)
			if err != nil {
				return nil, err
			}

			if err := (&info).parseAttributes(nattrs); err != nil {
				return nil, err
			}

			// Parsed the necessary data.
			return &info, nil
		}
	}

	// No station info found
	return nil, os.ErrNotExist
}

func (info *StationInfo) parseAttributes(attrs []netlink.Attribute) error {
	for _, a := range attrs {
		switch a.Type {
		case unix.NL80211_STA_INFO_CONNECTED_TIME:
			// Though nl80211 does not specify, this value appears to be in seconds:
			// * @NL80211_STA_INFO_CONNECTED_TIME: time since the station is last connected
			info.Connected = time.Duration(binary.NativeEndian.Uint32(a.Data)) * time.Second
		case unix.NL80211_STA_INFO_INACTIVE_TIME:
			// * @NL80211_STA_INFO_INACTIVE_TIME: time since last activity (u32, msecs)
			info.Inactive = time.Duration(binary.NativeEndian.Uint32(a.Data)) * time.Millisecond
		case unix.NL80211_STA_INFO_RX_BYTES64:
			info.ReceivedBytes = int(binary.NativeEndian.Uint64(a.Data))
		case unix.NL80211_STA_INFO_TX_BYTES64:
			info.TransmittedBytes = int(binary.NativeEndian.Uint64(a.Data))
		case unix.NL80211_STA_INFO_SIGNAL:
			//  * @NL80211_STA_INFO_SIGNAL: signal strength of last received PPDU (u8, dBm)
			// Should just be cast to int8, see code here: https://git.kernel.org/pub/scm/linux/kernel/git/jberg/iw.git/tree/station.c#n378
			info.Signal = int(int8(a.Data[0]))
		case unix.NL80211_STA_INFO_SIGNAL_AVG:
			info.SignalAverage = int(int8(a.Data[0]))
		case unix.NL80211_STA_INFO_BEACON_SIGNAL_AVG:
			info.BeaconSignalAverage = int(int8(a.Data[0]))
		case unix.NL80211_STA_INFO_RX_PACKETS:
			info.ReceivedPackets = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_STA_INFO_TX_PACKETS:
			info.TransmittedPackets = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_STA_INFO_TX_RETRIES:
			info.TransmitRetries = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_STA_INFO_TX_FAILED:
			info.TransmitFailed = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_STA_INFO_BEACON_LOSS:
			info.BeaconLoss = int(binary.NativeEndian.Uint32(a.Data))
		case unix.NL80211_STA_INFO_RX_BITRATE, unix.NL80211_STA_INFO_TX_BITRATE:
			rate, err := parseRateInfo(a.Data)
			if err != nil {
				return err
			}
			switch a.Type {
			case unix.NL80211_STA_INFO_RX_BITRATE:
				info.ReceiveBitrate = rate.Bitrate

				//additions for MCS
				info.ReceiveMCS = rate.MCS
				info.ReceivePHY = rate.PHY
			case unix.NL80211_STA_INFO_TX_BITRATE:
				info.TransmitBitrate = rate.Bitrate

				//additions for MCS
				info.TransmitMCS = rate.MCS
				info.TransmitPHY = rate.PHY
			}
		}

		// Only use 32-bit counters if the 64-bit counters are not present.
		// If the 64-bit counters appear later in the slice, they will overwrite
		// these values.
		if info.ReceivedBytes == 0 && a.Type == unix.NL80211_STA_INFO_RX_BYTES {
			info.ReceivedBytes = int(binary.NativeEndian.Uint32(a.Data))
		}
		if info.TransmittedBytes == 0 && a.Type == unix.NL80211_STA_INFO_TX_BYTES {
			info.TransmittedBytes = int(binary.NativeEndian.Uint32(a.Data))
		}
	}

	return nil
}

func parseRateInfo(b []byte) (*rateInfo, error) {
	attrs, err := netlink.UnmarshalAttributes(b)
	if err != nil {
		return nil, err
	}

	var info rateInfo
	for _, a := range attrs {
		switch a.Type {
		case unix.NL80211_RATE_INFO_BITRATE32:
			info.Bitrate = int(binary.NativeEndian.Uint32(a.Data))
		// Begin MCS additions
		case unix.NL80211_RATE_INFO_MCS:
			info.MCS = int(a.Data[0])
			info.PHY = "HT"
		case unix.NL80211_RATE_INFO_VHT_MCS:
			info.MCS = int(a.Data[0])
			info.PHY = "VHT"
		case unix.NL80211_RATE_INFO_HE_MCS:
			info.MCS = int(a.Data[0])
			info.PHY = "HE"
		case unix.NL80211_RATE_INFO_EHT_MCS:
			info.MCS = int(a.Data[0])
			info.PHY = "EHT"
		}

		// Only use 16-bit counters if the 32-bit counters are not present.
		// If the 32-bit counters appear later in the slice, they will overwrite
		// these values.
		if info.Bitrate == 0 && a.Type == unix.NL80211_RATE_INFO_BITRATE {
			info.Bitrate = int(binary.NativeEndian.Uint16(a.Data))
		}
	}

	// Scale bitrate to bits/second as base unit instead of 100kbits/second.
	// * @NL80211_RATE_INFO_BITRATE: total bitrate (u16, 100kbit/s)
	info.Bitrate *= 100 * 1000

	return &info, nil
}

func retryRate(s *StationInfo) int {
	if s.TransmittedPackets == 0 {
		return 0
	}
	if s.TransmitRetries > s.TransmittedPackets {
		return 100
	}
	rr := s.TransmitRetries * 100 / s.TransmittedPackets
	return rr
}
