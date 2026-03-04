package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jwil007/roamctl/wpac"
	"github.com/jwil007/roamctl/wpas"
)

func main() {
	start := time.Now()

	//open unixsocket connection for comamands
	cc, err := wpas.Connect("wlan0")
	if err != nil {
		log.Fatalf("wpas.Connect %v", err)
	}
	//open unixsocket connection for events
	ce, err := wpas.Connect("wlan0")
	if err != nil {
		log.Fatalf("wpas.Connect %v", err)
	}

	storedConfig, err := wpac.GetConfig(cc)
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
	errS := wpac.SetConfig(cc, noRoamConfig)
	if errS != nil {
		log.Fatalf("wpac.SetConfig %v", errS)
	}

	fmt.Println("restoring config")
	errR := wpac.SetConfig(cc, storedConfig)
	if errR != nil {
		log.Fatalf("wpac.SetConfig %v", errR)
	}
	elapsed := time.Since(start)

	signal, err := wpac.GetSignal(cc)
	if err != nil {
		log.Fatalf("wpac.GetSignal: %v", err)
	}
	fmt.Printf("Signal struct %+v\n", signal)

	fmt.Printf("Process duration: %v", elapsed)

	//ctx, cancel := context.WithCancel(context.Background())
	//events := make(chan string)
	//errc := make(chan error)
	//
	defer cc.Close()
	defer ce.Close()
	//defer cancel()
	//
	////attach to wpa_supp event stream
	//go ce.ListenEvents(ctx, events, errc)
	//
	//for {
	//	select {
	//	case event := <-events:
	//		fmt.Println(event)
	//		if strings.Contains(event, "CTRL-EVENT-CONNECTED") {
	//			return
	//		}
	//	case err := <-errc:
	//		log.Fatalf("event listener error: %v", err)
	//	}
	//}

}
