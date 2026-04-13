package netlink

import (
	"net"
	"sync"
	"time"

	"github.com/mdlayher/genetlink"
)

type STAInfo struct {
	RxBitrate    int
	RxMCS        int
	RxPHY        string
	TxBitrate    int
	TxMCS        int
	TxPHY        string
	TxRetries    int
	RetryRate    int
	TxFails      int
	BeaconLoss   int
	SignalAvg    int
	ConnDuration time.Duration
	BSSID        string
	ChannelWidth string
}

//Everything below I borrowed from mdlayher/wifi

type StationInfo struct {
	// The interface that this station is associated with.
	InterfaceIndex int

	// The hardware address of the station.
	HardwareAddr net.HardwareAddr

	// The time since the station last connected.
	Connected time.Duration

	// The time since wireless activity last occurred.
	Inactive time.Duration

	// The number of bytes received by this station.
	ReceivedBytes int

	// The number of bytes transmitted by this station.
	TransmittedBytes int

	// The number of packets received by this station.
	ReceivedPackets int

	// The number of packets transmitted by this station.
	TransmittedPackets int

	// The current data receive bitrate, in bits/second.
	ReceiveBitrate int

	// The current data transmit bitrate, in bits/second.
	TransmitBitrate int

	// The signal strength of the last received PPDU, in dBm.
	Signal int

	// The average signal strength, in dBm.
	SignalAverage int

	// The number of times the station has had to retry while sending a packet.
	TransmitRetries int

	// The number of times a packet transmission failed.
	TransmitFailed int

	// The number of times a beacon loss was detected.
	BeaconLoss int

	// adding this stuff in
	TransmitMCS int
	ReceiveMCS  int
	TransmitPHY string
	ReceivePHY  string
	PHY         string
}

type Interface struct {
	// The index of the interface.
	Index int

	// The name of the interface.
	Name string

	// The hardware address of the interface.
	HardwareAddr net.HardwareAddr

	// The physical device that this interface belongs to.
	PHY int

	// The virtual device number of this interface within a PHY.
	Device int

	// The operating mode of the interface.
	Type InterfaceType

	// The interface's wireless frequency in MHz.
	Frequency int

	// The interface's wireless channel width.
	ChannelWidth ChannelWidth
}

type ChannelWidth int

const (
	ChannelWidth20NoHT ChannelWidth = iota
	ChannelWidth20
	ChannelWidth40
	ChannelWidth80
	ChannelWidth80P80
	ChannelWidth160
	ChannelWidth5
	ChannelWidth10
	ChannelWidth1
	ChannelWidth2
	ChannelWidth4
	ChannelWidth8
	ChannelWidth16
	ChannelWidth320
)

func (c ChannelWidth) String() string {
	switch c {
	case ChannelWidth20NoHT:
		return "20MHz"
	case ChannelWidth20:
		return "20MHz"
	case ChannelWidth40:
		return "40MHz"
	case ChannelWidth80:
		return "80MHz"
	case ChannelWidth80P80:
		return "80MHz"
	case ChannelWidth160:
		return "160MHz"
	case ChannelWidth5:
		return "5MHz"
	case ChannelWidth10:
		return "10MHz"
	case ChannelWidth1:
		return "1MHz"
	case ChannelWidth2:
		return "2MHz"
	case ChannelWidth4:
		return "4MHz"
	case ChannelWidth8:
		return "8MHz"
	case ChannelWidth16:
		return "16MHz"
	case ChannelWidth320:
		return "320MHz"
	default:
		return "Unknown"
	}
}

type Client struct {
	c *client
}

type client struct {
	c             *genetlink.Conn
	familyID      uint16
	familyVersion uint8

	// scan is used to synchronize access to the Scan method.
	scan sync.Mutex
}

type InterfaceType int

type rateInfo struct {
	// Bitrate in bits per second.
	Bitrate int
	MCS     int
	PHY     string
}
