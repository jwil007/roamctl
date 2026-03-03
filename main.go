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

	currentConfig, err := wpac.GetConfig(c)
	if err != nil {
		log.Fatalf("wpac.GetConfig %v", err)
	}

	fmt.Printf("Current wpa_supp config: %+v\n", currentConfig)

	defer c.Close()
}
