//go:build linux

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jwil007/roamctl/internal/roam"
	"github.com/jwil007/roamctl/internal/wpac"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		log.Printf("%v", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		os.Exit(0)
	}
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	edit := flag.Bool("edit", false, "edit config file")
	reset := flag.Bool("reset", false, "reset default config")
	flag.Parse()

	cfg, err := roam.HandleConfig(reset, edit)
	if err != nil {
		return fmt.Errorf("roam.HandleConfig: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("cfg.Validate: %w", err)
	}
	//open unixsocket connection for commands
	c, err := wpac.Connect(cfg.Interface)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("wpac.Connect: %w\nInterface name %s may be wrong. "+
				"Rerun with -edit flag to edit interface name.", err, cfg.Interface)
		}
		return fmt.Errorf("wpac.Connect %w", err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		err = c.Close()
		if err != nil {
			log.Printf("failed to close unix connection: %v", err)
		}
	}()
	defer cancel()
	err = cfg.ProcessLoop(c, ctx)
	if err != nil {
		return fmt.Errorf("roam.ProcessLoop: %v", err)
	}
	return nil
}
