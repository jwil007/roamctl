package main

import (
	"flag"
	"log"

	"github.com/jwil007/roamctl/roam"
)

func main() {
	iface := flag.String("i", "wlan0", "specify wireless interface")
	flag.Parse()
	ifaceName := *iface
	err := roam.Autoroam(ifaceName)
	if err != nil {
		log.Fatalf("roam.Autoroam: %v", err)
	}
}
