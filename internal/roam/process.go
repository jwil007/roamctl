// Package roam: handles the main roaming loop
package roam

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jwil007/roamctl/internal/wpac"
)

func (cfg *Config) ProcessLoop(c *wpac.Client, ctx context.Context) error {
	rc := &roamContext{}
	log.Println("Starting roamctl... exit with ctrl+c")
	cleanup, err := cfg.handleWpaSuppConfig(c)
	if err != nil {
		return fmt.Errorf("handleWpaSuppConfig: %w", err)
	}
	defer cleanup() //sets wpa_supplicant back to original state
	//Start polling signal stats
	log.Println("Waiting for trigger to enter roam decision loop...")
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
			//aps, err := c.ScanResults(cfg.SSID)
			//if err != nil {
			//	return fmt.Errorf("c.ScanResults: %w", err)
			//}
			//rc.bgScanAPs = cfg.scoreAll(aps)
			//logScoredAPs(rc.bgScanAPs, rc.lastKnown.BSSID)
		case <-ctx.Done():
			return ctx.Err()
		case con := <-sigCh:
			if con.BSSID != "" {
				rc.lastKnown = &con
			}
			if rc.lastKnown == nil {
				continue
			}
			//log.Printf("DEBUG: last polled signal stats %+v\n", rc.lastKnown)
			switch {
			case cfg.thresholdCheck(rc):
				cont, err := cfg.roamProcessWrapper(c, ctx, rc)
				if err != nil {
					return fmt.Errorf("roamProcessWrapper: %w", err)
				}
				if cont {
					continue
				}
			}
		case err = <-sigErrCh:
			return fmt.Errorf("c.PollSignal: %w", err)
		}
	}
}

func (cfg *Config) thresholdCheck(rc *roamContext) bool {
	rssi := rc.lastKnown.AvgRSSIBeacon
	if rc.lastKnown.AvgRSSIBeacon == 0 {
		//log.Printf("No RSSI BEACON available, falling back to basic RSSI")
		rssi = rc.lastKnown.RSSI
	}
	//log.Printf("threshold RSSI recorded as: %d", rssi)
	switch {
	case rc.noCandCounter > cfg.MaxNoCandidates && rssi < cfg.Thresholds.RSSI:
		log.Printf("Roam attempts with no candidates (%v) exceeds threshold (%v). "+
			"Falling back to bgscan...", rc.noCandCounter, cfg.MaxNoCandidates)
		rc.thresholdFlag = noCandidateLimit
		return true
	case rssi < cfg.Thresholds.RSSI:
		log.Printf("Last polled RSSI (%vdBm) below threshold (%vdBm). "+
			"Entering roam decision loop...", rssi, cfg.Thresholds.RSSI)
		rc.thresholdFlag = lowRSSI
		return true
	case rc.lastKnown.LinkSpeed < cfg.Thresholds.DataRate:
		log.Printf("Last polled data rate (%vMbps) below threshold (%vMbps). "+
			"Entering roam decision loop...", rc.lastKnown.LinkSpeed, cfg.Thresholds.DataRate)
		rc.thresholdFlag = lowDataRate
		return true
	}
	rc.thresholdFlag = noValue
	return false
}

func (cfg *Config) roamProcessWrapper(
	c *wpac.Client,
	ctx context.Context,
	rc *roamContext,
) (bool, error) {
	if time.Since(rc.lastRoamFailure) < cfg.FailureBackoffTime {
		log.Printf("Roam failure backoff in effect. %v remaining",
			cfg.FailureBackoffTime-time.Since(rc.lastRoamFailure))
		return true, nil //continue
	}
	if time.Since(rc.lastRoamSuccess) < cfg.SuccessBackoffTime {
		log.Printf("Roam success backoff in effect. %v remaining",
			cfg.SuccessBackoffTime-time.Since(rc.lastRoamSuccess))
		return true, nil //continue
	}
	if time.Since(rc.lastNoCandidates) < cfg.NoCandidatesBackoffTime {
		log.Printf("No candidates backoff in effect. %v remaining",
			cfg.NoCandidatesBackoffTime-time.Since(rc.lastNoCandidates))
		return true, nil //continue
	}
	resultFlag, errR := cfg.roamDecisionLoop(c, ctx, rc)
	if errR != nil {
		return false, fmt.Errorf("makeRoamDecision %w", errR)
	}
	switch resultFlag {
	case success:
		rc.lastRoamSuccess = time.Now()
		rc.lastKnown = nil //clear lastKnown stats
		rc.noCandCounter = 0
	case failure:
		rc.lastRoamFailure = time.Now()
	case noCandidates:
		rc.lastNoCandidates = time.Now()
		rc.noCandCounter++
		log.Println("\033[33m## No better APs found, returning to signal monitoring...\033[0m")
		log.Printf("No candidates counter at %v. Max threshold is %v", rc.noCandCounter, cfg.MaxNoCandidates)
	case unknown:
		return false, fmt.Errorf("unexpected roam result")
	}
	return false, nil
}

func (cfg *Config) roamDecisionLoop(
	c *wpac.Client,
	ctx context.Context,
	rc *roamContext,
) (roamResultFlag, error) {
	var scoredAPs []scoredBSS
	var currAP scoredBSS
	if rc.thresholdFlag == noCandidateLimit {
		log.Printf("Roaming using background scan...")
		aps, err := c.ScanResults(cfg.SSID)
		if err != nil {
			return unknown, fmt.Errorf("c.ScanResults: %w", err)
		}
		saps := cfg.scoreAll(aps)
		scoredAPs = saps
		logScoredAPs(scoredAPs, rc.lastKnown.BSSID)
		for _, ap := range scoredAPs {
			if ap.bssid == rc.lastKnown.BSSID {
				currAP = ap
			}
		}
	} else {
		prepAPs, err := cfg.prepareScoredAPs(c, ctx, rc)
		if err != nil {
			return unknown, fmt.Errorf("prepareScoredAPs: %w", err)
		}
		scoredAPs = prepAPs
		for _, ap := range scoredAPs {
			if ap.bssid == rc.lastKnown.BSSID {
				currAP = ap
			}
		}
	}
	if len(scoredAPs) == 0 {
		return unknown, fmt.Errorf("scored APs array is nil")
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
	case cfg.roamReadyCheck(candAP, currAP, rc) == true:
		log.Printf("Better AP found BSSID: %v Score: %v\n", candAP.bssid, candAP.finalScore)
		flag, err := cfg.roamToCandidate(c, ctx, candAP)
		if err != nil {
			return flag, fmt.Errorf("roamToCandidate: %w", err)
		}
		return flag, nil
	default:
		return noCandidates, nil
	}
}

func (cfg *Config) prepareScoredAPs(
	c *wpac.Client,
	ctx context.Context,
	rc *roamContext,
) ([]scoredBSS, error) {
	aps, err := c.ScanResults(cfg.SSID)
	if err != nil {
		return nil, fmt.Errorf("c.ScanResults: %w", err)
	}
	scoredAPs := cfg.scoreAll(aps)
	logScoredAPs(scoredAPs, rc.lastKnown.BSSID)
	hasFreshCandidate := false

	for _, candAP := range scoredAPs {
		if candAP.age < cfg.MaxScanAge {
			hasFreshCandidate = true
			break
		}
	}
	if !hasFreshCandidate {
		log.Println("Stale scan data, rerunning scan...")
		out, err := cfg.rescan(c, ctx, cfg.SSID)
		if err != nil {
			return nil, fmt.Errorf("cfg.rescan: %w", err)
		}
		scoredAPs = out
		logScoredAPs(scoredAPs, rc.lastKnown.BSSID)
	}
	return scoredAPs, nil
}

func (cfg *Config) roamToCandidate(
	c *wpac.Client,
	ctx context.Context,
	candAP scoredBSS,
) (roamResultFlag, error) {
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

func (cfg *Config) roamReadyCheck(candidate scoredBSS, current scoredBSS, rc *roamContext) bool {
	switch rc.thresholdFlag {
	case noValue:
	case noCandidateLimit:
		log.Printf("Ignoring scan age limit, selecting best AP available.")
		if candidate.finalScore-current.finalScore > cfg.ScoreDelta &&
			candidate.bssid != current.bssid {
			return true
		}
		return false
	case lowRSSI:
		if candidate.finalScore-current.finalScore > cfg.ScoreDelta &&
			candidate.bssid != current.bssid &&
			candidate.age < cfg.MaxScanAge {
			return true
		}
		return false
	case lowDataRate:
		if candidate.finalScore-current.finalScore > cfg.ScoreDelta &&
			candidate.bssid != current.bssid &&
			candidate.age < cfg.MaxScanAge {
			return true
		}
		return false
	}
	return false
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
