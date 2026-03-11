package main

import (
	"log"

	"github.com/jwil007/roamctl/wpac"
)

func main() {
	//open unixsocket connection for commands
	cc, err := wpac.Connect("wlan0")
	if err != nil {
		log.Fatalf("wpac.Connect %v", err)
	}
	config, err := cc.GetConfig()
	if err != nil {
		log.Fatalf("wpac.GetConfig: %v", err)
	}
	defer cc.Close()

	_, errScan := wpac.Scan("wlan0", config.SSID)
	if errScan != nil {
		log.Fatalf("wpac.Scan: %v", errScan)
	}

}
