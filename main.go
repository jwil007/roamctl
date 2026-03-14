package main

import (
	"context"
	"flag"
	"log"

	"github.com/jwil007/roamctl/roam"
	"github.com/jwil007/roamctl/wpac"
)

func main() {
	iface := flag.String("i", "wlan0", "specify wireless interface")
	flag.Parse()
	ifaceName := *iface

	//open unixsocket connection for commands
	c, err := wpac.Connect(ifaceName)
	if err != nil {
		log.Fatalf("wpac.Connect %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		err := c.Close()
		if err != nil {
			log.Fatalf("failed to close unix connection: %v", err)
		}
	}()
	defer cancel()

	errA := roam.Autoroam(c, ctx)
	if errA != nil {
		log.Fatalf("roam.Autoroam: %v", errA)
	}
}
