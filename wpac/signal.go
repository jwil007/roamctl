package wpac

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jwil007/roamctl/wpas"
)

func GetSignal(c wpas.Client) (Signal, error) {
	out, err := c.Cmd("SIGNAL_POLL")
	if err != nil {
		return Signal{}, fmt.Errorf("c.Cmd(\"SIGNAL_POLL\") %w", err)
	}
	var s Signal
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "RSSI="):
			rssi, err := strconv.Atoi(line[5:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.RSSI = rssi
		case strings.HasPrefix(line, "LINKSPEED="):
			linkspeed, err := strconv.Atoi(line[10:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.LinkSpeed = linkspeed
		case strings.HasPrefix(line, "NOISE="):
			noise, err := strconv.Atoi(line[6:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.Noise = noise
		case strings.HasPrefix(line, "FREQUENCY="):
			freq, err := strconv.Atoi(line[10:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.Freq = freq
		case strings.HasPrefix(line, "WIDTH="):
			width := line[6:]
			s.ChannelWidth = width
		case strings.HasPrefix(line, "AVG_RSSI="):
			avgRSSI, err := strconv.Atoi(line[9:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.AvgRSSI = avgRSSI
		case strings.HasPrefix(line, "AVG_BEACON_RSSI="):
			avgRSSIbeacon, err := strconv.Atoi(line[16:])
			if err != nil {
				return Signal{}, fmt.Errorf("strconv.Atoi: %w", err)
			}
			s.AvgRSSIBeacon = avgRSSIbeacon
		}
	}
	return s, nil
}
