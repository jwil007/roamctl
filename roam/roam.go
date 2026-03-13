// Package roam: handles the main roaming loop
package roam

import (
	"context"
	"fmt"
	"log"

	"github.com/jwil007/roamctl/wpac"
)

func Autoroam() error {
	//open unixsocket connection for commands
	c, err := wpac.Connect("wlan0")
	if err != nil {
		return fmt.Errorf("wpac.Connect %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		err := c.Close()
		if err != nil {
			log.Fatalf("failed to close unix connection: %v", err)
		}
	}()
	defer cancel()
	//retrive current wpa_supplicant configuration
	config, err := c.GetConfig()
	if err != nil {
		return fmt.Errorf("wpac.GetConfig: %v", err)
	}
	//run scan
	aps, errScan := c.Scan(ctx, config.SSID)
	if errScan != nil {
		return fmt.Errorf("wpac.Scan: %v", errScan)
	}
	//roam through all bssids in scan data
	fmt.Println("\nRoaming to all BSSIDs in sequence")
	for _, bss := range aps {
		result, err := c.Roam(ctx, bss.BSSID)
		if err != nil {
			log.Printf("error roaming to BSSID %v: %v", bss.BSSID, err)
		}
		fmt.Printf("Success:%v TargetBSSID:%v FinalBSSID:%v Duration:%v Message:%v\n",
			result.Success,
			result.TargetBSSID,
			result.FinalBSSID,
			result.Duration,
			result.Message)
	}
	return nil
}
