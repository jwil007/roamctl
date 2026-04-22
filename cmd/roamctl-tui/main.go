package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jwil007/roamctl/internal/tui"
)

var version = "dev"

func main() {
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	versionFlag := flag.Bool("version", false,
		"print version and exit")
	iface := flag.String("iface", "", "interface to bind to")
	flag.Parse()
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}
	// automatically set interface if only one active
	if *iface == "" {
		files, err := os.ReadDir("/run/roamctl")
		if err != nil {
			return fmt.Errorf("os.ReadDir: %w", err)
		}
		var pidFiles []string
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".sock") {
				pidFiles = append(pidFiles, f.Name())
			}
		}
		switch len(pidFiles) {
		case 0:
			fmt.Println("No roamctl process found.\n " +
				"Make sure roamctl is running.")
			os.Exit(0)
		case 1:
			*iface = strings.TrimSuffix(pidFiles[0], ".sock")
		default:
			fmt.Println("More than one iface running roamctl.\n" +
				"Specify iface with sudo roamctl-tui -iface <iface_name>")
			os.Exit(0)
		}
	}

	err := tui.Tui(iface)
	if err != nil {
		return fmt.Errorf("tui.Tui: %w", err)
	}
	return nil
}
