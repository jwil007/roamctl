package roam

import (
	"slices"

	"github.com/jwil007/roamctl/internal/config"
	"github.com/jwil007/roamctl/internal/wpac"
)

func scoreAll(aps []wpac.RichBSS, cfg *config.Config) []scoredBSS {
	var scoredList []scoredBSS
	for _, ap := range aps {
		scored := score(ap, cfg)
		scoredList = append(scoredList, scored)
	}
	slices.SortFunc(scoredList, func(a, b scoredBSS) int {
		return b.finalScore - a.finalScore
	})
	return scoredList
}

func score(bss wpac.RichBSS, cfg *config.Config) scoredBSS {
	rs := cfg.ScoreWeights.RSSI * scoreRSSI(bss.RSSI, cfg) / 100
	ss := cfg.SNR * scoreSNR(bss.SNR, cfg) / 100
	bs := cfg.Band * scoreBand(bss.Band, cfg) / 100
	cws := cfg.ChannelWidth * scoreCW(bss.ChannelWidth, cfg) / 100
	//es := cfg.EstThruput * cfg.scoreET(bss.EstThruput) / 100
	us := cfg.QBSSUtil * scoreUtil(bss.QBSSUtil) / 100
	//sts := cfg.QBSSStaCt * cfg.scoreStaCt(bss.QBSSStaCt) / 100
	ps := cfg.PHYType * scorePhy(bss.PHYType, cfg) / 100
	totalWeight := cfg.ScoreWeights.RSSI + cfg.SNR + cfg.Band + cfg.ChannelWidth + cfg.QBSSUtil + cfg.PHYType
	scoreSum := rs + ss + bs + cws + us + ps
	if totalWeight == 0 {
		return scoredBSS{}
	}
	finalScore := scoreSum * 100 / totalWeight
	return scoredBSS{
		bssid:      bss.BSSID,
		freq:       bss.Freq,
		channelNum: bss.ChannelNum,
		finalScore: finalScore,
		rssiScore:  rs,
		rssi:       bss.RSSI,
		snrScore:   ss,
		snr:        bss.SNR,
		bandScore:  bs,
		band:       bss.Band,
		cwScore:    cws,
		cw:         bss.ChannelWidth,
		utilScore:  us,
		util:       bss.QBSSUtil,
		phyScore:   ps,
		phy:        bss.PHYType,
		age:        bss.Age,
	}
}

func scoreRSSI(rssi int, cfg *config.Config) int {
	var score int
	if cfg.MaxRSSI-cfg.MinRSSI == 0 {
		return 0
	}
	score = (rssi - cfg.MinRSSI) * 100 / (cfg.MaxRSSI - cfg.MinRSSI)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

func scoreSNR(snr int, cfg *config.Config) int {
	var score int
	if cfg.MaxSNR-cfg.MinSNR == 0 {
		return 0
	}
	score = (snr - cfg.MinSNR) * 100 / (cfg.MaxSNR - cfg.MinSNR)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}

func scoreBand(band wpac.Band, cfg *config.Config) int {
	var score int
	switch band {
	case wpac.BandUnknown:
		score = 0
	case wpac.Band2point4:
		score = cfg.Band2point4
	case wpac.Band5:
		score = cfg.Band5
	case wpac.Band6:
		score = cfg.Band6
	}
	return score
}

func scoreCW(cw wpac.ChannelWidth, cfg *config.Config) int {
	var score int
	switch cw {
	case wpac.ChannelWidthUnknown:
		score = 0
	case wpac.ChannelWidth20:
		score = cfg.ChannelWidth20
	case wpac.ChannelWidth40:
		score = cfg.ChannelWidth40
	case wpac.ChannelWidth80:
		score = cfg.ChannelWidth80
	case wpac.ChannelWidth80Plus80:
		score = cfg.ChannelWidth160
	case wpac.ChannelWidth160:
		score = cfg.ChannelWidth160
	case wpac.ChannelWidth320:
		score = cfg.ChannelWidth320
	}
	return score
}

//func (c *Config) scoreET(et int) int {
//	var score int
//	return score
//}

func scoreUtil(util uint8) int {
	var score int
	score = (255 - int(util)) * 100 / 255
	return score
}

//func (c *Config) scoreStaCt(sc uint16) int {
//	var score int
//	return score
//}

func scorePhy(phy wpac.PHYType, cfg *config.Config) int {
	var score int
	switch phy {
	case wpac.PHYUnknown:
		score = 0
	case wpac.PHYLegacy:
		score = cfg.PHYLegacy
	case wpac.PHY80211n:
		score = cfg.PHY80211n
	case wpac.PHY80211ac:
		score = cfg.PHY80211ac
	case wpac.PHY80211ax:
		score = cfg.PHY80211ax
	case wpac.PHY80211be:
		score = cfg.PHY80211be
	}
	return score
}
