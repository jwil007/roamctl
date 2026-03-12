package main

import (
	"context"
	"log"

	"github.com/jwil007/roamctl/wpac"
)

func main() {
	//open unixsocket connection for commands
	c, err := wpac.Connect("wlan0")
	if err != nil {
		log.Fatalf("wpac.Connect %v", err)
	}
	config, err := c.GetConfig()
	if err != nil {
		log.Fatalf("wpac.GetConfig: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		err := c.Close()
		if err != nil {
			log.Fatalf("failed to close unix connection: %v", err)
		}
	}()
	defer cancel()

	_, errScan := c.Scan(ctx, config.SSID)
	if errScan != nil {
		log.Fatalf("wpac.Scan: %v", errScan)
	}

}
