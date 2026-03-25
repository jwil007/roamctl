package roam

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jwil007/roamctl/internal/config"
	"github.com/jwil007/roamctl/internal/wpac"
)

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
}

func (s scoredBSS) String() string {
	return fmt.Sprintf(
		"bssid:%s score:%d rssi:%d(scr:%d) snr:%d(scr:%d) band:%s(scr:%d) cw:%s(scr:%d) util:%d(scr:%d) phy:%s(scr:%d) age:%s",
		s.bssid,
		s.finalScore,
		s.rssi, s.rssiScore,
		s.snr, s.snrScore,
		s.band, s.bandScore,
		s.cw, s.cwScore,
		s.util, s.utilScore,
		s.phy, s.phyScore,
		s.age,
	)
}

type roamContext struct {
	cfg              *config.Config
	ssid             string
	scoredAPs        []scoredBSS
	candidateAP      scoredBSS
	currentAP        scoredBSS
	lastKnown        *wpac.ConnectionStatus
	roamResultFlag   roamResultFlag
	lastRoamSuccess  time.Time
	lastRoamFailure  time.Time
	lastNoCandidates time.Time
	hysteresisActive bool
	lastTriggerRSSI  int
	lastEvalTime     time.Time
	entryScanned     bool //Prevents scan loop
	roamingTier      roamingTier
	scanState        scanState
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

type roamResultFlag int

const (
	success roamResultFlag = iota
	failure
	noCandidates
	unknown
)

var ErrScanRetryLimit = errors.New("scan retry limit exceeded")

var green = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
var red = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
var blue = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
