package wpac // Package wpac: Reads and sets wpa_supplicant configuration options
import (
	"fmt"
	"strings"

	"github.com/jwil007/roamctl/wpas"
)

func SetConfig(c wpas.Client, config WPAConfig) error {
	err := setBGScan(c, config)
	if err != nil {
		return fmt.Errorf("setBGScan: %w", err)
	}
	return nil
}

func setBGScan(c wpas.Client, config WPAConfig) error {
	s := "SET_NETWORK " + config.NetworkID + " bgscan " + config.BGScan
	out, err := c.Cmd(s)
	if err != nil {
		return fmt.Errorf("c.Cmd(%s): %w", s, err)
	}
	if strings.TrimSpace(string(out)) != "OK" {
		return fmt.Errorf("c.Cmd(%s): %s", s, string(out))
	}
	return nil
}
