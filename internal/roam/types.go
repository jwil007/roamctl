package roam

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/jwil007/roamctl/internal/config"
	"github.com/jwil007/roamctl/internal/ipc"
	"github.com/jwil007/roamctl/internal/netlink"
	"github.com/jwil007/roamctl/internal/wpac"
)

type roamContext struct {
	cfg              *config.Config
	iface            string
	ssid             string
	richBSSList      []wpac.RichBSS
	scoredAPs        []scoredBSS
	candidateAP      scoredBSS
	currentAP        scoredBSS
	lastKnown        *ConnectionStatus
	roamResultFlag   roamResultFlag
	roamInProgress   bool
	lastRoamStats    wpac.RoamStats
	lastRoamAttempt  time.Time
	hysteresisActive bool
	lastTriggerRSSI  int
	lastEvalTime     time.Time
	entryScanned     bool
	entryScannedCrit bool
	fullScannedCrit  bool
	roamingTier      roamingTier
	scanState        scanState
	unhealthyConn    bool
	unhealthyLogged  bool
	lastConnChange   time.Time
	rssiRingBuffer   []int
	rssiWriteIdx     int
	richByBSSID      map[string]wpac.RichBSS
	bssPenalties     []bssPenalty
	ipcChan          chan ipc.ProcessState
	snapshot         atomic.Pointer[ipc.ProcessState]
	wpaDisconnect    bool
}
type scoredBSS struct {
	bssid      string
	freq       int
	channelNum int
	finalScore int
	rssiScore  int
	rssi       int
	snrScore   int
	snr        int
	bandScore  int
	band       wpac.Band
	cwScore    int
	cw         wpac.ChannelWidth
	utilScore  int
	util       uint8
	phyScore   int
	phy        wpac.PHYType
	age        time.Duration
	failCount  int
}

func (s scoredBSS) String() string {
	return fmt.Sprintf(
		"bssid:%s score:%d rssi:%d(scr:%d) snr:%d(scr:%d) chan:%d %s"+
			"(scr:%d) cw:%s(scr:%d) util:%d(scr:%d) phy:%s(scr:%d) age:%s",
		s.bssid,
		s.finalScore,
		s.rssi, s.rssiScore,
		s.snr, s.snrScore,
		s.channelNum, s.band, s.bandScore,
		s.cw, s.cwScore,
		s.util, s.utilScore,
		s.phy, s.phyScore,
		s.age,
	)
}

type ConnectionStatus struct {
	wpac.Status
	netlink.STAInfo
}

func (c ConnectionStatus) String() string {
	return fmt.Sprintf(
		"ssid: %s wpa_state: %s, bssid:%s rssi:%d avgrssi:%d "+
			"avgrssibeacon:%d freq:%d cw:%s "+
			"RxBitrate:%v RxMCS:%v RxPHY:%v TxBitrate:%v "+
			"TxMSC:%v TxPHY:%v TxRetries:%v RetryRate: %v "+
			"TxFails:%v beaconloss:%v connduration:%v",
		c.SSID,
		c.WPAState,
		c.BSSID,
		c.RSSI,
		c.AvgRSSI,
		c.AvgRSSIBeacon,
		c.Freq,
		c.ChannelWidth,
		c.RxBitrate,
		c.RxMCS,
		c.RxPHY,
		c.TxBitrate,
		c.TxMCS,
		c.TxPHY,
		c.TxRetries,
		c.RetryRate,
		c.TxFails,
		c.BeaconLoss,
		c.ConnDuration)
}

type bssPenalty struct {
	BSSID     string    `json:"bssid"`
	SSID      string    `json:"ssid"`
	Band      wpac.Band `json:"band"`
	FailCount int       `json:"fail_count"`
	LastFail  time.Time `json:"last_fail"`
}

type scanState struct {
	mu             sync.RWMutex
	cond           *sync.Cond
	scanInProgress bool
	scanMode       scanMode
	channels       []int
	scanDuration   time.Duration
	lastScanTime   time.Time
	bssidHash      uint64
	bssListStable  bool
}

type scanMode int

const (
	fastScan scanMode = iota
	fullScan
	noScan
)

func (s scanMode) String() string {
	switch s {
	case fastScan:
		return "fast_scan"
	case fullScan:
		return "full_scan"
	case noScan:
		return "scan_disabled"
	}
	return ""
}

type roamingTier int

const (
	unknownTier roamingTier = iota
	noRoam
	opportunistic
	active
	critical
)

func (s roamingTier) String() string {
	switch s {
	case unknownTier:
		return "unknown"
	case noRoam:
		return "roam_disabled"
	case opportunistic:
		return "opportunistic"
	case active:
		return "active_roaming"
	case critical:
		return "critical"
	}
	return ""
}

type roamResultFlag int

const (
	unknown roamResultFlag = iota
	success
	failure
	noCandidates
)

func (s roamResultFlag) String() string {
	switch s {
	case success:
		return "success"
	case failure:
		return "failure"
	case noCandidates:
		return "no_candidates"
	case unknown:
		return "unknown"
	}
	return ""
}

var ErrScanRetryLimit = errors.New("scan retry limit exceeded")

var green = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
var red = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
var blue = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
