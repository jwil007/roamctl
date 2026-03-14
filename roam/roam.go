// Package roam: handles the main roaming loop
package roam

import (
	"context"
	"fmt"
	"log"
	"time"

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

func ProcessLoop(c *wpac.Client, ctx context.Context, thr RoamThresholds) error {
	//Get Current wpa_supplicant status
	storedConf, err := c.GetConfig()
	if err != nil {
		return fmt.Errorf("c.GetConfig: %v", err)
	}
	//Disable bgscan to prevent autonomous roaming
	bgscanOffConfig := wpac.WPAConfig{
		SSID:      storedConf.SSID,
		NetworkID: storedConf.NetworkID,
		BGScan:    "",
	}
	err = c.SetConfig(bgscanOffConfig)
	if err != nil {
		return fmt.Errorf("c.SetConfig: %w", err)
	}
	//restore original config when process stops
	defer func() {
		err = c.SetConfig(storedConf)
		if err != nil {
			log.Printf("error restoring wpa_supplicant config: %v", err)
		}
	}()
	//Start polling signal stats
	sigCh, sigErrCh := c.PollSignal(ctx, 1*time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sig := <-sigCh:
			switch {

			case sig.RSSI <= thr.RSSI:
				err = roamDecisionLoop(c, ctx, sig)
				if err != nil {
					return fmt.Errorf("makeRoamDecision %w", err)
				}
			case sig.LinkSpeed <= thr.DataRate:
				err = roamDecisionLoop(c, ctx, sig)
				if err != nil {
					return fmt.Errorf("makeRoamDecision %w", err)
				}
			}
		case err = <-sigErrCh:
			return fmt.Errorf("c.PollSignal: %w", err)
		}
	}
}

func roamDecisionLoop(c *wpac.Client, ctx context.Context, sig wpac.Signal) error {
	fmt.Printf("congrats, roam decision loop entered with the following signal stats\n%+v\n", sig)
	fmt.Println("sleeping for 5 seconds...")
	time.Sleep(5 * time.Second)
	return nil
}
