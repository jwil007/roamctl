// Package roam: handles the main roaming loop
package roam

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jwil007/roamctl/wpac"
)

func (cfg *Config) ProcessLoop(c *wpac.Client, ctx context.Context) error {
	log.Println("Starting roamctl... exit with ctrl+c")
	cleanup, err := cfg.handleWpaSuppConfig(c)
	if err != nil {
		return fmt.Errorf("handleWpaSuppConfig: %w", err)
	}
	defer cleanup() //sets wpa_supplicant back to original state
	//Start polling signal stats
	log.Println("Waiting for trigger to enter roam decision loop...")
	var lastKnown *wpac.ConnectionStatus
	var lastRoamSuccess time.Time
	var lastRoamFailure time.Time
	var lastNoCandidates time.Time
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
			case cfg.thresholdCheck(lastKnown):
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
				if time.Since(lastNoCandidates) < cfg.NoCandidatesBackoffTime {
					log.Printf("No candidates backoff in effect. %v remaining",
						cfg.NoCandidatesBackoffTime-time.Since(lastNoCandidates))
					continue
				}
				log.Printf("Entering roam decision loop with stats: %+v", lastKnown)
				resultFlag, errR := cfg.roamDecisionLoop(c, ctx, lastKnown.BSSID)
				if errR != nil {
					return fmt.Errorf("makeRoamDecision %w", errR)
				}
				switch resultFlag {
				case success:
					lastRoamSuccess = time.Now()
					lastKnown = nil //clear lastKnown stats so they don't update until fresh poll
				case failure:
					lastRoamFailure = time.Now()
				case noCandidates:
					lastNoCandidates = time.Now()
				case unknown:
					return fmt.Errorf("unexpected roam result")
				}
			}
		case err = <-sigErrCh:
			return fmt.Errorf("c.PollSignal: %w", err)
		}
	}
}

func (cfg *Config) roamDecisionLoop(c *wpac.Client, ctx context.Context, currBSSID string) (roamResultFlag, error) {
	scoredAPs, currAP, err := cfg.prepareScoredAPs(c, ctx, cfg.SSID, currBSSID)
	if err != nil {
		return unknown, fmt.Errorf("prepareScoredAPs: %w", err)
	}
	candAP := scoredAPs[0] //scoredAP is sorted with highest score first
	switch {
	case currAP.bssid == "":
		log.Printf("current AP not in scan data, selecting AP with highest score")
		flag, err := cfg.roamToCandidate(c, ctx, candAP)
		if err != nil {
			return flag, fmt.Errorf("roamToCandidate: %w", err)
		}
		return flag, nil
	case cfg.roamReadyCheck(candAP, currAP) == true:
		log.Printf("Better AP found BSSID: %v Score: %v\n", candAP.bssid, candAP.finalScore)
		flag, err := cfg.roamToCandidate(c, ctx, candAP)
		if err != nil {
			return flag, fmt.Errorf("roamToCandidate: %w", err)
		}
		return flag, nil
	default:
		log.Println("\033[33m## No better APs found, returning to signal monitoring...\033[0m")
		return noCandidates, nil
	}
}

func (cfg *Config) roamToCandidate(c *wpac.Client, ctx context.Context, candAP scoredBSS) (roamResultFlag, error) {
	result, err := c.Roam(ctx, candAP.bssid)
	if err != nil {
		return failure, fmt.Errorf("c.Roam(%v): %w", candAP.bssid, err)
	}
	log.Printf("Roam Result // Success:%v TargetBSSID:%v FinalBSSID:%v Duration:%v Message:%v",
		result.Success,
		result.TargetBSSID,
		result.FinalBSSID,
		result.Duration,
		result.Message)
	switch result.Success {
	case true:
		log.Printf("\033[32m## Successful Roam to BSSID:%v RSSI:%v Band:%v\033[0m",
			candAP.bssid, candAP.rssi, candAP.band)
		log.Println("Waiting for next trigger...")
		return success, nil
	case false:
		log.Printf("\033[31m## Failed Roam to BSSID:%v RSSI:%v Band:%v\nReason:%v\033[0m",
			candAP.bssid, candAP.rssi, candAP.band, result.Message)
		log.Println("Waiting for next trigger...")
		return failure, nil
	}
	return unknown, fmt.Errorf("missing result.Success data from c.Roam")
}

func (cfg *Config) roamReadyCheck(candidate scoredBSS, current scoredBSS) bool {
	if candidate.finalScore-current.finalScore > cfg.ScoreDelta &&
		candidate.bssid != current.bssid &&
		candidate.age < cfg.MaxScanAge {
		return true
	}
	return false
}

func (cfg *Config) prepareScoredAPs(
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
	logScoredAPs(scoredAPs, currBSSID)
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
					}
				}
				logScoredAPs(scoredAPs, currAP.bssid)
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
		logScoredAPs(scoredAPs, currAP.bssid)
	}
	if currAP.bssid == "" {
		log.Printf("last connected AP (BSSID: %v) not in scan results", currAP.bssid)
	}
	return scoredAPs, currAP, nil
}

func (cfg *Config) rescan(c *wpac.Client, ctx context.Context, ssid string) ([]scoredBSS, error) {
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

func logScoredAPs(scoredAPs []scoredBSS, bssid string) {
	log.Println("Most recent scan data: ")
	for _, a := range scoredAPs {
		if a.bssid == bssid {
			log.Printf("%+v [CURRENT AP]", a)
		} else {
			log.Printf("%+v", a)
		}
	}
}

func (cfg *Config) thresholdCheck(lastKnown *wpac.ConnectionStatus) bool {
	rssi := lastKnown.AvgRSSIBeacon
	if lastKnown.AvgRSSIBeacon == 0 {
		//log.Printf("No RSSI BEACON available, falling back to basic RSSI")
		rssi = lastKnown.RSSI
	}
	//log.Printf("threshold RSSI recorded as: %d", rssi)
	switch {
	case rssi < cfg.Thresholds.RSSI:
		return true
	case lastKnown.LinkSpeed < cfg.Thresholds.DataRate:
		return true
	}
	return false
}

func (cfg *Config) handleWpaSuppConfig(c *wpac.Client) (func(), error) {
	//Get Current wpa_supplicant status
	storedConf, err := c.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("c.GetConfig: %v", err)
	}
	//Disable bgscan to prevent autonomous roaming
	bgscanOffConfig := wpac.WPAConfig{
		SSID:      storedConf.SSID,
		NetworkID: storedConf.NetworkID,
		BGScan:    "",
	}
	err = c.SetConfig(bgscanOffConfig)
	if err != nil {
		return nil, fmt.Errorf("c.SetConfig: %w", err)
	}
	cfg.SSID = storedConf.SSID

	cleanup := func() {
		err = c.SetConfig(storedConf)
		if err != nil {
			log.Printf("error restoring wpa_supplicant config: %v", err)
		}
	}
	return cleanup, nil
}
