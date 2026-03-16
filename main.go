package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jwil007/roamctl/roam"
	"github.com/jwil007/roamctl/wpac"
)

func main() {
	if err := run(); err != nil {
		log.Printf("%v", err)
		os.Exit(1)
	}
}

func run() error {
	iface := flag.String("i", "wlan0", "specify wireless interface")
	rssi := flag.Int("r", -65, "specify rssi for roaming threshold")
	flag.Parse()
	ifaceName := *iface
	rssiThr := *rssi

	//manually define config during testing
	thresholds := roam.Thresholds{
		RSSI:       rssiThr,
		DataRate:   0,
		ScoreDelta: 5,
	}
	timing := roam.Timing{
		RoamBackoffTime: 5 * time.Second,
		SigPollInterval: 500 * time.Millisecond,
		BGScanInterval:  30 * time.Second,
	}
	scoreWeights := roam.ScoreWeights{
		RSSI:         100,
		MinRSSI:      -80,
		MaxRSSI:      -40,
		SNR:          0,
		MinSNR:       0,
		MaxSNR:       0,
		Band:         50,
		ChannelWidth: 0,
		EstThruput:   0,
		QBSSUtil:     25,
		QBSSStaCt:    0,
		PHYType:      15,
	}
	cfg := roam.Config{
		Thresholds:   thresholds,
		ScoreWeights: scoreWeights,
		Timing:       timing,
	}

	//open unixsocket connection for commands
	c, err := wpac.Connect(ifaceName)
	if err != nil {
		return fmt.Errorf("wpac.Connect %v", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		err = c.Close()
		if err != nil {
			log.Printf("failed to close unix connection: %v", err)
		}
	}()
	defer cancel()
	err = cfg.ProcessLoop(c, ctx)
	if err != nil {
		return fmt.Errorf("roam.ProcessLoop: %v", err)
	}
	return nil
}
