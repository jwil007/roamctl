package roam

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"slices"
	"time"
)

func (rc *roamContext) readBSSPenaltyFile() error {
	path := "/run/roamctl/" + rc.iface + "_penalty.json"
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("os.Open(%s): %w", path, err)
	}
	defer func() {
		_ = f.Close()
	}()
	b, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("io.ReadAll(%s): %w", path, err)
	}
	if len(b) == 0 {
		return nil
	}
	err = json.Unmarshal(b, &rc.bssPenalties)
	if err != nil {
		return fmt.Errorf("json.Unmarshal(%q): %w", path, err)
	}
	return nil
}

// recordBSSPenalty records (fail) or clears a roam failure for candidate, the
// AP the roam was actually attempted to.
func (rc *roamContext) recordBSSPenalty(fail bool, candidate scoredBSS) error {
	rc.apMu.Lock()
	defer rc.apMu.Unlock()
	matched := false
	for i, bp := range rc.bssPenalties {
		if bp.BSSID == candidate.bssid &&
			bp.SSID == rc.ssid &&
			bp.Band == candidate.band {
			if fail {
				rc.bssPenalties[i].FailCount++
				rc.bssPenalties[i].LastFail = time.Now()
				matched = true
				break
			} else {
				slog.Info("Removing entry from penalty.json",
					"entry", rc.bssPenalties[i].BSSID)
				rc.bssPenalties = slices.Delete(rc.bssPenalties, i, i+1)
				break
			}
		}
	}
	if !matched && fail {
		rc.bssPenalties = append(rc.bssPenalties, bssPenalty{
			BSSID:     candidate.bssid,
			SSID:      rc.ssid,
			Band:      candidate.band,
			FailCount: 1,
			LastFail:  time.Now(),
		})
	}
	err := rc.writeBSSPenaltyFile()
	if err != nil {
		return fmt.Errorf("writeBSSPenaltyFile: %w", err)
	}
	return nil
}

func (rc *roamContext) writeBSSPenaltyFile() error {
	path := "/run/roamctl/" + rc.iface + "_penalty.json"
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("os.OpenFile(%s): %w", path, err)
	}
	defer func() {
		_ = f.Close()
	}()
	err = json.NewEncoder(f).Encode(rc.bssPenalties)
	if err != nil {
		return fmt.Errorf("json.Encode(%q): %w", path, err)
	}
	return nil
}
