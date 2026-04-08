package main

import (
	"fmt"
	"os"

	"github.com/jwil007/roamctl/internal/tui"
)

func main() {
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() error {
	err := tui.Tui()
	if err != nil {
		return fmt.Errorf("tui.Tui: %w", err)
	}
	return nil
}
