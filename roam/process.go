// Package roam: handles the main roaming loop
package roam

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jwil007/roamctl/wpac"
)

func (cfg Config) ProcessLoop(c *wpac.Client, ctx context.Context) error {
	log.Println("Starting roamctl... exit with ctrl+c")
	ssid, err := handleWpaSuppConfig(c)
	if err != nil {
		return fmt.Errorf("handleWpaSuppConfig: %w", err)
	}
	//Start polling signal stats
	log.Println("Waiting for trigger to enter roam decision loop...")
	var lastKnown *wpac.ConnectionStatus
	var lastRoam time.Time
	sigCh, sigErrCh := c.PollSignal(ctx, cfg.Timing.SigPollInterval)
	bgScanTicker := time.NewTicker(cfg.Timing.BGScanInterval)
	for {
		select {
		case <-bgScanTicker.C:
			log.Println("bgScanTicker reached 0, running scan...")
			if err = c.Scan(ctx); err != nil {
				return fmt.Errorf("c.Scan: %w", err)
			}
			log.Println("bgScan complete")
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
			case lastKnown.AvgRSSI <= cfg.Thresholds.RSSI:
				log.Printf("Entering roam decision loop with stats: %+v", lastKnown)
				if time.Since(lastRoam) < cfg.RoamBackoffTime { //timer to wait before attemtping to roam again
					log.Printf("Roam backoff in effect. %v remaining", cfg.RoamBackoffTime-time.Since(lastRoam))
					continue
				}
				err = cfg.roamDecisionLoop(c, ctx, ssid, lastKnown.BSSID)
				if err != nil {
					return fmt.Errorf("makeRoamDecision %w", err)
				}
				lastRoam = time.Now()
			}
		case err = <-sigErrCh:
			return fmt.Errorf("c.PollSignal: %w", err)
		}
	}
}

func (cfg Config) roamDecisionLoop(c *wpac.Client, ctx context.Context, ssid string, currBSSID string) error {
	aps, err := c.ScanResults(ssid)
	if err != nil {
		return fmt.Errorf("c.ScanResults: %w", err)
	}
	scoredAPs := cfg.scoreAll(aps)
	log.Println("Most recent scan data: ")
	for _, a := range scoredAPs {
		log.Printf("%+v\n", a)
	}
	maxAge := 5
	scoreDelta := cfg.ScoreDelta
	hasFreshCandidates := false
	var currAP scoredBSS
	for _, candAP := range scoredAPs {
		switch {
		case candAP.bssid == currBSSID:
			if candAP.age > maxAge {
				log.Println("Stale scan data, rerunning scan...")
				out, err := cfg.rescan(c, ctx, ssid)
				if err != nil {
					return fmt.Errorf("cfg.rescan: %w", err)
				}
				scoredAPs = out
				for _, ap := range scoredAPs {
					if ap.bssid == currBSSID {
						currAP = ap
						log.Printf("Current AP details: %+v", currAP)
					}
				}
			}
		default:
			if candAP.age < maxAge {
				hasFreshCandidates = true
			}
		}
	}
	if !hasFreshCandidates {
		log.Println("No fresh candidates, rerunning scan...")
		out, err := cfg.rescan(c, ctx, ssid)
		if err != nil {
			return fmt.Errorf("cfg.rescan: %w", err)
		}
		scoredAPs = out
		for _, a := range scoredAPs {
			if a.bssid == currAP.bssid {
				currAP = a
			}
		}
	}
	for _, candAP := range scoredAPs {
		if candAP.finalScore-currAP.finalScore > scoreDelta && candAP.bssid != currAP.bssid && candAP.age < maxAge {
			result, err := c.Roam(ctx, candAP.bssid)
			if err != nil {
				return fmt.Errorf("c.Roam(%v): %w", candAP.bssid, err)
			}
			log.Printf("Better AP found BSSID: %v Score: %v\n", candAP.bssid, candAP.finalScore)
			log.Printf("Roam Result // Success:%v TargetBSSID:%v FinalBSSID:%v Duration:%v Message:%v",
				result.Success,
				result.TargetBSSID,
				result.FinalBSSID,
				result.Duration,
				result.Message)
			if result.Success == true {
				log.Printf("## Successful Roam to BSSID:%v RSSI:%v Band:%v",
					candAP.bssid, candAP.rssi, candAP.band)
				log.Println("Waiting for next trigger...")
				return nil
			}
			if result.Success == false {
				log.Printf("## Failed Roam to BSSID:%v RSSI:%v Band:%v\nReason:%v",
					candAP.bssid, candAP.rssi, candAP.band, result.Message)
				log.Println("Waiting for next trigger...")
				return nil
			}
		}
	}
	log.Println("No better APs found, returning to signal monitoring...")
	return nil
}

func (cfg Config) rescan(c *wpac.Client, ctx context.Context, ssid string) ([]scoredBSS, error) {
	if err := c.Scan(ctx); err != nil {
		return nil, fmt.Errorf("c.Scan: %w", err)
	}
	aps, err := c.ScanResults(ssid)
	if err != nil {
		return nil, fmt.Errorf("c.ScanResults: %w", err)
	}
	scoredAPs := cfg.scoreAll(aps)
	return scoredAPs, nil
}

func handleWpaSuppConfig(c *wpac.Client) (string, error) {
	//Get Current wpa_supplicant status
	storedConf, err := c.GetConfig()
	if err != nil {
		return "", fmt.Errorf("c.GetConfig: %v", err)
	}
	//Disable bgscan to prevent autonomous roaming
	bgscanOffConfig := wpac.WPAConfig{
		SSID:      storedConf.SSID,
		NetworkID: storedConf.NetworkID,
		BGScan:    "",
	}
	err = c.SetConfig(bgscanOffConfig)
	if err != nil {
		return "", fmt.Errorf("c.SetConfig: %w", err)
	}
	//restore original config when process stops
	defer func() {
		err = c.SetConfig(storedConf)
		if err != nil {
			log.Printf("error restoring wpa_supplicant config: %v", err)
		}
	}()
	return storedConf.SSID, nil
}
