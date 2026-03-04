package wpac

import (
	"fmt"
	"strings"

	"github.com/jwil007/roamctl/wpas"
)

func RunScan(c wpas.Client) error {
	out, err := c.Cmd("SCAN")
	if err != nil {
		return fmt.Errorf("c.Cmd(\"SCAN\"): %w", err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf("c.Cmd(\"SCAN\"): %s", string(out))
	}
	return nil
}

//func GetScanResults
//
//func BuildBSSList
