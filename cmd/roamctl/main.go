//go:build linux

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	charmlog "github.com/charmbracelet/log"
	"github.com/jwil007/roamctl/internal/roam"
	"github.com/jwil007/roamctl/internal/wpac"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		if strings.Contains(err.Error(), "context canceled") {
			slog.Info("Exiting...")
			os.Exit(0)
		}
		slog.Error("Error occurred", "value", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(version)
		os.Exit(0)
	}
	edit := flag.Bool("edit", false, "edit config file")
	reset := flag.Bool("reset", false, "reset default config")
	levelStr := flag.String("level", "info", "log level (debug, info)")
	flag.Parse()

	var logLevel slog.Level
	switch strings.ToLower(*levelStr) {
	case "debug":
		logLevel = slog.LevelDebug
	default:
		logLevel = slog.LevelInfo
	}

	logger := charmlog.New(os.Stdout)
	logger.SetLevel(charmlog.Level(logLevel))
	slog.SetDefault(slog.New(logger))

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
			slog.Error("failed to close unix connection", "value", err)
		}
	}()
	defer cancel()
	err = cfg.ProcessLoop(c, ctx)
	if err != nil {
		return fmt.Errorf("roam.ProcessLoop: %v", err)
	}
	return nil
}
