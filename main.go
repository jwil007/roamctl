package main

import (
	"context"
	"fmt"
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
	aps, errScan := c.Scan(ctx, config.SSID)
	if errScan != nil {
		log.Fatalf("wpac.Scan: %v", errScan)
	}
	fmt.Println("\nRoaming to all BSSIDs in sequence")
	for _, bss := range aps {
		result, err := c.Roam(ctx, bss.BSSID)
		if err != nil {
			log.Printf("error roaming to BSSID %v: %v", bss.BSSID, err)
		}
		fmt.Printf("Success:%v TargetBSSID:%v FinalBSSID:%v Duration:%v\n",
			result.Success,
			result.TargetBSSID,
			result.FinalBSSID,
			result.Duration)
	}
}
