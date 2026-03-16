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
	ssid, cleanup, err := handleWpaSuppConfig(c)
	if err != nil {
		return fmt.Errorf("handleWpaSuppConfig: %w", err)
	}
	defer cleanup() //sets wpa_supplicant back to original state
	//Start polling signal stats
	log.Println("Waiting for trigger to enter roam decision loop...")
	var lastKnown *wpac.ConnectionStatus
	var lastRoamSuccess time.Time
	var lastRoamFailure time.Time
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
				if time.Since(lastRoamFailure) < cfg.FailureBackoffTime {
					log.Printf("Roam failure backoff in effect. %v remaining",
						cfg.FailureBackoffTime-time.Since(lastRoamFailure))
					continue
				}
				if time.Since(lastRoamSuccess) < cfg.SuccessBackoffTime {
					log.Printf("Roam success backoff in effect. %v remaining",
						cfg.SuccessBackoffTime-time.Since(lastRoamSuccess))
					continue
				}
				log.Printf("Entering roam decision loop with stats: %+v", lastKnown)
				success, errR := cfg.roamDecisionLoop(c, ctx, ssid, lastKnown.BSSID)
				if errR != nil {
					return fmt.Errorf("makeRoamDecision %w", errR)
				}
				switch success {
				case true:
					lastRoamSuccess = time.Now()
					lastKnown = nil //clear lastKnown stats so they don't update until fresh poll
				case false:
					lastRoamFailure = time.Now()
				}
			}
		case err = <-sigErrCh:
			return fmt.Errorf("c.PollSignal: %w", err)
		}
	}
}

func (cfg Config) roamDecisionLoop(c *wpac.Client, ctx context.Context, ssid string, currBSSID string) (bool, error) {
	scoredAPs, currAP, err := cfg.prepareScoredAPs(c, ctx, ssid, currBSSID)
	if err != nil {
		return false, fmt.Errorf("prepareScoredAPs: %w", err)
	}
	for _, candAP := range scoredAPs {
		if cfg.roamReadyCheck(candAP, currAP) {
			result, err := c.Roam(ctx, candAP.bssid)
			if err != nil {
				return false, fmt.Errorf("c.Roam(%v): %w", candAP.bssid, err)
			}
			log.Printf("Better AP found BSSID: %v Score: %v\n", candAP.bssid, candAP.finalScore)
			log.Printf("Roam Result // Success:%v TargetBSSID:%v FinalBSSID:%v Duration:%v Message:%v",
				result.Success,
				result.TargetBSSID,
				result.FinalBSSID,
				result.Duration,
				result.Message)
			if result.Success == true {
				log.Printf("\033[32m## Successful Roam to BSSID:%v RSSI:%v Band:%v\033[0m",
					candAP.bssid, candAP.rssi, candAP.band)
				log.Println("Waiting for next trigger...")
				return true, nil
			}
			if result.Success == false {
				log.Printf("\033[31m## Failed Roam to BSSID:%v RSSI:%v Band:%v\nReason:%v\033[0m",
					candAP.bssid, candAP.rssi, candAP.band, result.Message)
				log.Println("Waiting for next trigger...")
				return false, nil
			}
		}
	}
	log.Println("\033[33mNo better APs found, returning to signal monitoring...\033[0m")
	return false, nil
}

func (cfg Config) roamReadyCheck(candidate scoredBSS, current scoredBSS) bool {
	if candidate.finalScore-current.finalScore > cfg.ScoreDelta &&
		candidate.bssid != current.bssid &&
		candidate.age < cfg.MaxScanAge {
		return true
	}
	return false
}

func (cfg Config) prepareScoredAPs(
	c *wpac.Client,
	ctx context.Context,
	ssid string,
	currBSSID string,
) ([]scoredBSS, scoredBSS, error) {
	aps, err := c.ScanResults(ssid)
	if err != nil {
		return nil, scoredBSS{}, fmt.Errorf("c.ScanResults: %w", err)
	}
	scoredAPs := cfg.scoreAll(aps)
	log.Println("Most recent scan data: ")
	for _, a := range scoredAPs {
		log.Printf("%+v\n", a)
	}
	hasFreshCandidates := false
	var currAP scoredBSS
	for _, candAP := range scoredAPs {
		switch {
		case candAP.bssid == currBSSID:
			currAP = candAP
			if candAP.age > cfg.MaxScanAge {
				log.Println("Stale scan data, rerunning scan...")
				out, err := cfg.rescan(c, ctx, ssid)
				if err != nil {
					return nil, scoredBSS{}, fmt.Errorf("cfg.rescan: %w", err)
				}
				scoredAPs = out
				hasFreshCandidates = true
				for _, ap := range scoredAPs {
					if ap.bssid == currBSSID {
						currAP = ap
						log.Printf("Current AP details: %+v", currAP)
					}
				}
			}
		default:
			if candAP.age < cfg.MaxScanAge {
				hasFreshCandidates = true
			}
		}
	}
	if !hasFreshCandidates {
		log.Println("No fresh candidates, rerunning scan...")
		out, err := cfg.rescan(c, ctx, ssid)
		if err != nil {
			return nil, scoredBSS{}, fmt.Errorf("cfg.rescan: %w", err)
		}
		scoredAPs = out
		for _, a := range scoredAPs {
			if a.bssid == currAP.bssid {
				currAP = a
			}
		}
	}
	return scoredAPs, currAP, nil
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

func handleWpaSuppConfig(c *wpac.Client) (string, func(), error) {
	//Get Current wpa_supplicant status
	storedConf, err := c.GetConfig()
	if err != nil {
		return "", nil, fmt.Errorf("c.GetConfig: %v", err)
	}
	//Disable bgscan to prevent autonomous roaming
	bgscanOffConfig := wpac.WPAConfig{
		SSID:      storedConf.SSID,
		NetworkID: storedConf.NetworkID,
		BGScan:    "",
	}
	err = c.SetConfig(bgscanOffConfig)
	if err != nil {
		return "", nil, fmt.Errorf("c.SetConfig: %w", err)
	}
	cleanup := func() {
		err = c.SetConfig(storedConf)
		if err != nil {
			log.Printf("error restoring wpa_supplicant config: %v", err)
		}
	}
	return storedConf.SSID, cleanup, nil
}
