// Package roam: handles the main roaming loop
package roam

import (
	"context"
	"fmt"
	"log"

	"github.com/jwil007/roamctl/wpac"
)

func Autoroam(c *wpac.Client, ctx context.Context) error {
	//retrive current wpa_supplicant configuration
	config, err := c.GetConfig()
	if err != nil {
		return fmt.Errorf("c.GetConfig: %v", err)
	}
	//shut off bgscan
	bgscanOffConfig := wpac.WPAConfig{
		SSID:      config.SSID,
		NetworkID: config.NetworkID,
		BGScan:    "",
	}
	errBG := c.SetConfig(bgscanOffConfig)
	if errBG != nil {
		return fmt.Errorf("c.SetConfig: %w", errBG)
	}
	//run scan
	aps, errScan := c.Scan(ctx, config.SSID)
	if errScan != nil {
		return fmt.Errorf("wpac.Scan: %v", errScan)
	}
	//roam through all bssids in scan data
	fmt.Printf("\nRoaming to %v BSSIDs\n", len(aps))
	for _, bss := range aps {
		result, errRoam := c.Roam(ctx, bss.BSSID)
		if errRoam != nil {
			log.Printf("error roaming to BSSID %v: %v", bss.BSSID, errRoam)
		}
		fmt.Printf("Success:%v TargetBSSID:%v FinalBSSID:%v Duration:%v Message:%v\n",
			result.Success,
			result.TargetBSSID,
			result.FinalBSSID,
			result.Duration,
			result.Message)
	}
	//restore original bgscan config
	errRes := c.SetConfig(config)
	if errRes != nil {
		return fmt.Errorf("c.SetConfig: %w", errRes)
	}
	return nil
}

//func AlertForRSSI(rssi int, iface string) (string, error) {
//	c, err := wpac.Connect(iface)
//	if err != nil {
//		return fmt.Errorf("wpac.Connect %v", err)
//	}
//}
