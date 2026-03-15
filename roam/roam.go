// Package roam: handles the main roaming loop
package roam

import (
	"context"
	"fmt"
	"log"
	"strings"
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

func ProcessLoop(c *wpac.Client, ctx context.Context, thr Thresholds) error {
	fmt.Println("Starting roamctl... exit with ctrl+c")
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
	fmt.Println("Waiting for trigger to enter roam decision loop...")
	var lastKnown *wpac.ConnectionStatus
	var lastRoam time.Time
	sigCh, sigErrCh := c.PollSignal(ctx, 500*time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case con := <-sigCh:
			if con.BSSID != "" {
				lastKnown = &con
			}
			if lastKnown == nil {
				continue
			}
			switch {
			case lastKnown.AvgRSSI <= thr.RSSI:
				if time.Since(lastRoam) < 5*time.Second { //timer to wait before attemtping to roam again
					continue
				}
				bssid, err := roamDecisionLoop(c, ctx, storedConf.SSID, lastKnown)
				if err != nil {
					return fmt.Errorf("makeRoamDecision %w", err)
				}
				lastKnown.BSSID = bssid
				lastRoam = time.Now()
			}
		case err = <-sigErrCh:
			return fmt.Errorf("c.PollSignal: %w", err)
		}
	}
}

func roamDecisionLoop(c *wpac.Client, ctx context.Context, ssid string, con *wpac.ConnectionStatus) (string, error) {
	fmt.Printf("\nRoam decision loop entered with the following signal stats\n%+v\n", con)
	fmt.Println("\nRunning scan...")
	for {
		aps, err := c.Scan(ctx, ssid)
		if err != nil {
			if strings.Contains(err.Error(), "FAIL-BUSY") {
				fmt.Println("\ninterface busy, retrying scan in 2 seconds")
				time.Sleep(2 * time.Second)
				continue
			}
			return "", fmt.Errorf("c.Scan: %w", err)
		}
		for _, ap := range aps {
			if ap.RSSI-con.AvgRSSI >= 5 && ap.BSSID != con.BSSID {
				result, err := c.Roam(ctx, ap.BSSID)
				if err != nil {
					return "", fmt.Errorf("c.Roam(%v): %w", ap.BSSID, err)
				}
				fmt.Printf("\nBetter AP found BSSID: %v RSSI: %v\n", ap.BSSID, ap.RSSI)
				fmt.Printf("\nRoam Result // Success:%v TargetBSSID:%v FinalBSSID:%v Duration:%v Message:%v\n",
					result.Success,
					result.TargetBSSID,
					result.FinalBSSID,
					result.Duration,
					result.Message)
				if result.Success == true {
					fmt.Printf("\n## Successful Roam to BSSID:%v RSSI:%v Band:%v Channel:%v\n",
						ap.BSSID, ap.RSSI, ap.Band, ap.ChannelNum)
					fmt.Println("\nWaiting for next trigger...")
					return result.FinalBSSID, nil
				}
				if result.Success == false {
					fmt.Printf("\n## Failed Roam to BSSID:%v RSSI:%v Band:%v Channel:%v\nReason:%v\n",
						ap.BSSID, ap.RSSI, ap.Band, ap.ChannelNum, result.Message)
					fmt.Println("\nWaiting for next trigger...")
					return result.FinalBSSID, nil
				}
				return "", nil
			}
		}
		fmt.Println("No better APs found, returning to signal monitoring...")
		return "", nil
	}
}
