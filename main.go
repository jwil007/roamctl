package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jwil007/roamctl/wpac"
	"github.com/jwil007/roamctl/wpas"
)

func main() {

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

	ctx, cancel := context.WithCancel(context.Background())

	config, err := wpac.GetConfig(cc)
	if err != nil {
		log.Fatalf("wpac.GetConfig: %v", err)
	}

	//events, errEvents := ce.ListenEvents(ctx)
	//signalc, errSignal := wpac.PollSignal(ctx, ce, 100*time.Millisecond)

	defer cc.Close()
	defer ce.Close()
	defer cancel()
	fmt.Printf("Current SSID: %v\n", config.SSID)
	start := time.Now()
	errScan := wpac.RunScan(cc)
	if errScan != nil {
		log.Fatalf("wpac.RunScan: %v", errScan)
	}
	errWait := ce.WaitForEvent(ctx, "CTRL-EVENT-SCAN-RESULTS", 10*time.Second)
	if errWait != nil {
		log.Printf("ce.WaitForEvent: %v", errWait)
	}
	bssids, err := wpac.GetScanResults(cc, config.SSID)
	elapsed := time.Since(start)
	for _, bssid := range bssids {
		fmt.Println(bssid)
	}
	fmt.Printf("Process duration: %v", elapsed)

	//for {
	//	select {
	//	case sig := <-signalc:
	//		fmt.Printf("%v Signal struct %+v\n", time.Now(), sig)
	//	case err := <-errSignal:
	//		log.Fatalf("errSignal: %v", err)
	//	case event := <-events:
	//		fmt.Println(event)
	//		if strings.Contains(event, "CTRL-EVENT-CONNECTED") {
	//			return
	//		}
	//	case err := <-errEvents:
	//		log.Fatalf("errEvents: %v", err)
	//	}
	//}

}
