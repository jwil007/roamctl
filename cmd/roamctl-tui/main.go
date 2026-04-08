package main

import (
	"flag"
	"fmt"
	"os"

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
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}

	err := tui.Tui()
	if err != nil {
		return fmt.Errorf("tui.Tui: %w", err)
	}
	return nil
}
