package main

import (
	"fmt"
	"log"

	"github.com/jwil007/roamctl/wpac"
	"github.com/jwil007/roamctl/wpas"
)

func main() {

	c, err := wpas.Connect("wlan0")
	if err != nil {
		log.Fatalf("wpas.Connect %v", err)
	}

	storedConfig, err := wpac.GetConfig(c)
	if err != nil {
		log.Fatalf("wpac.GetConfig %v", err)
	}
	fmt.Printf("Current wpa_supp config: %+v\n", storedConfig)

	noRoamConfig := wpac.WPAConfig{
		SSID:      storedConfig.SSID,
		NetworkID: storedConfig.NetworkID,
		BGScan:    "",
		Iface:     storedConfig.Iface,
	}

	fmt.Println("disabling bgscan")
	errS := wpac.SetConfig(c, noRoamConfig)
	if errS != nil {
		log.Fatalf("wpac.SetConfig %v", errS)
	}

	fmt.Println("restoring config")
	errR := wpac.SetConfig(c, storedConfig)
	if errR != nil {
		log.Fatalf("wpac.SetConfig %v", errR)
	}

	defer c.Close()
}
