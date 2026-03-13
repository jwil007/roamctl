package wpac

import (
	"fmt"
	"log"
)

func constructRichBSS(wpaBSS WpasBSS, ieBSS IEBSS) RichBSS {
	band, channel, err := getBandandChanfromFreq(wpaBSS.Freq)
	if err != nil {
		log.Printf("getBandandChanfromFreq: %v", err)
	}
	return RichBSS{
		WpasBSS:    wpaBSS,
		IEBSS:      ieBSS,
		Band:       band,
		ChannelNum: channel,
	}
}

func getBandandChanfromFreq(freq int) (Band, int, error) {
	var channel int
	switch {
	case freq == 2484:
		channel = 14
		return Band2point4, channel, nil
	case freq >= 2412 && freq <= 2472:
		channel = (freq - 2407) / 5
		return Band2point4, channel, nil
	case freq >= 5180 && freq <= 5825:
		channel = (freq - 5000) / 5
		return Band5, channel, nil
	case freq >= 5955 && freq <= 7115:
		channel = (freq - 5950) / 5
		return Band6, channel, nil
	}
	return BandUnknown, channel, fmt.Errorf("failed to determine channel/band from freq: %v", freq)
}

func isMACAddress(s string) bool {
	if len(s) != 17 {
		return false
	}
	for i, c := range s {
		if i%3 == 2 {
			if c != ':' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
