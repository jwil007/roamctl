package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	//manually set roam thresholds during testing
	thr := roam.Thresholds{
		RSSI:     rssiThr,
		DataRate: 54,
	}
	err = roam.ProcessLoop(c, ctx, thr)
	if err != nil {
		return fmt.Errorf("roam.ProcessLoop: %v", err)
	}
	return nil
}
