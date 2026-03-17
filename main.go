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
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	iface := flag.String("i", "", "specify wireless interface")
	rssi := flag.Int("r", 0, "specify rssi for roaming threshold")
	flag.Parse()
	ifaceName := *iface
	rssiThr := *rssi

	cfg, err := roam.HandleConfig()
	if err != nil {
		return fmt.Errorf("roam.HandleConfig: %w", err)
	}

	if ifaceName != "" {
		cfg.Interface = ifaceName
	}

	if rssiThr != 0 {
		cfg.Thresholds.RSSI = rssiThr
	}

	//open unixsocket connection for commands
	c, err := wpac.Connect(cfg.Interface)
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
