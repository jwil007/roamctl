package roam

import (
	"slices"

	"github.com/jwil007/roamctl/wpac"
)

func (w ScoreWeights) scoreAll(aps []wpac.RichBSS) []scoredBSS {
	var scoredList []scoredBSS
	for _, ap := range aps {
		scored := w.score(ap)
		scoredList = append(scoredList, scored)
	}
	slices.SortFunc(scoredList, func(a, b scoredBSS) int {
		return b.finalScore - a.finalScore
	})
	return scoredList
}

func (w ScoreWeights) score(bss wpac.RichBSS) scoredBSS {
	rs := w.RSSI * scoreRSSI(bss.RSSI, w.MinRSSI, w.MaxRSSI) / 100
	ss := w.SNR * scoreSNR(bss.SNR, w.MinSNR, w.MaxSNR) / 100
	bs := w.Band * scoreBand(bss.Band) / 100
	cws := w.ChannelWidth * scoreCW(bss.ChannelWidth) / 100
	//es := w.EstThruput * scoreET(bss.EstThruput) / 100
	us := w.QBSSUtil * scoreUtil(bss.QBSSUtil) / 100
	//sts := w.QBSSStaCt * scoreStaCt(bss.QBSSStaCt) / 100
	ps := w.PHYType * scorePhy(bss.PHYType) / 100
	totalWeight := w.RSSI + w.SNR + w.Band + w.ChannelWidth + w.QBSSUtil + w.PHYType
	scoreSum := rs + ss + bs + cws + us + ps
	if totalWeight == 0 {
		return scoredBSS{}
	}
	finalScore := scoreSum * 100 / totalWeight
	return scoredBSS{
		bssid:      bss.BSSID,
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

func scoreRSSI(rssi int, minRSSI int, maxRSSI int) int {
	var score int
	score = (rssi - minRSSI) * 100 / (maxRSSI - minRSSI)
	return score
}

func scoreSNR(snr int, minSNR int, maxSNR int) int {
	var score int
	if maxSNR-minSNR == 0 {
		return 0
	}
	score = (snr - minSNR) * 100 / (maxSNR - minSNR)
	return score
}

func scoreBand(band wpac.Band) int {
	var score int
	switch band {
	case wpac.BandUnknown:
		score = 0
	case wpac.Band2point4:
		score = 0
	case wpac.Band5:
		score = 85
	case wpac.Band6:
		score = 100
	}
	return score
}

func scoreCW(cw wpac.ChannelWidth) int {
	var score int
	switch cw {
	case wpac.ChannelWidthUnknown:
		score = 0
	case wpac.ChannelWidth20:
		score = 30
	case wpac.ChannelWidth40:
		score = 60
	case wpac.ChannelWidth80:
		score = 80
	case wpac.ChannelWidth80Plus80:
		score = 90
	case wpac.ChannelWidth160:
		score = 90
	case wpac.ChannelWidth320:
		score = 100
	}
	return score
}

//func scoreET(et int) int {
//	var score int
//	return score
//}

func scoreUtil(util uint8) int {
	var score int
	score = ((255 - int(util)) / 255) * 100
	return score
}

//func scoreStaCt(sc uint16) int {
//	var score int
//	return score
//}

func scorePhy(phy wpac.PHYType) int {
	var score int
	switch phy {
	case wpac.PHYUnknown:
		score = 0
	case wpac.PHYLegacy:
		score = 0
	case wpac.PHY80211n:
		score = 20
	case wpac.PHY80211ac:
		score = 50
	case wpac.PHY80211ax:
		score = 80
	case wpac.PHY80211be:
		score = 100
	}
	return score
}
