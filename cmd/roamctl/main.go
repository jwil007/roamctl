//go:build linux

package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	charmlog "github.com/charmbracelet/log"
	"github.com/jwil007/roamctl/internal/config"
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
	//handle args
	versionFlag := flag.Bool("version", false, "print version and exit")
	edit := flag.Bool("edit", false, "edit config file")
	template := flag.String("template", "", "select config template (base, macos, ios)")
	levelStr := flag.String("level", "info", "log level (debug, info)")
	iface := flag.String("iface", "wlan0", "interface to use")
	flag.Parse()
	if *versionFlag {
		fmt.Println(version)
		os.Exit(0)
	}
	ifaceSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "iface" {
			ifaceSet = true
		}
	})

	if flag.NArg() > 0 {
		_, _ = fmt.Fprintf(os.Stderr, "error: unexpected argument(s): %v\n", flag.Args())
		flag.Usage()
		os.Exit(1)
	}
	validLevels := map[string]bool{"debug": true, "info": true}
	if !validLevels[*levelStr] {
		_, _ = fmt.Fprintf(os.Stderr, "error: invalid log level %q, must be debug or info\n", *levelStr)
		flag.Usage()
		os.Exit(1)
	}
	validTemplates := map[string]bool{"": true, "base": true, "macos": true, "ios": true}
	if !validTemplates[*template] {
		_, _ = fmt.Fprintf(os.Stderr, "error: invalid template %q, must be default, macos, or ios\n", *template)
		flag.Usage()
		os.Exit(1)
	}
	var logLevel slog.Level
	switch strings.ToLower(*levelStr) {
	case "debug":
		logLevel = slog.LevelDebug
	default:
		logLevel = slog.LevelInfo
	}
	defer func() {
		_ = os.Remove("/run/roamctl.pid")
	}()

	//check if process already running
	running, pid, err := handlePIDFile()
	if err != nil {
		return fmt.Errorf("checkPIDFromFile: %w", err)
	}
	if running {
		fmt.Printf("roamctl daemon is already running at PID %d.\n"+
			"Stop it first with sudo systemctl stop roamctl", pid)
		os.Exit(0)
	}

	//init logging
	logger := charmlog.New(os.Stdout)
	logger.SetLevel(charmlog.Level(logLevel))
	logger.SetReportTimestamp(true)
	//logger.SetTimeFormat("15:04:05.000")
	slog.SetDefault(slog.New(logger))

	//read and validate config file
	cfg, err := config.HandleConfig(template, iface, edit)
	if err != nil {
		return fmt.Errorf("roam.HandleConfig: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("cfg.Validate: %w", err)
	}
	if *edit == true || *template != "" || ifaceSet {
		fmt.Println("Changes saved to file at /etc/roamctl/config.toml.\nIf running roamctl as a daemon, " +
			"apply changes with sudo systemctl restart roamctl")
		os.Exit(0)
	}

	//open wpa_supplicant control interface unix socket
	c, err := wpac.Connect(cfg.Interface)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			return fmt.Errorf("wpac.Connect: %w\nInterface name %s may be wrong. "+
				"Rerun with -edit flag to edit interface name.", err, cfg.Interface)
		}
		return fmt.Errorf("wpac.Connect %w", err)
	}

	//init context for all concurrent functions
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer func() {
		err = c.Close()
		if err != nil {
			slog.Error("failed to close unix connection", "value", err)
		}
	}()
	defer cancel()

	//start the roamctl process
	err = roam.Proc(c, ctx, cfg)
	if err != nil {
		return fmt.Errorf("roam.ProcessLoop: %v", err)
	}
	return nil
}

func handlePIDFile() (bool, int, error) {
	path := "/run/roamctl.pid"
	pid := os.Getpid()
	if data, err := os.ReadFile(path); err == nil {
		pidRead, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil {
			if syscall.Kill(pidRead, 0) == nil {
				return true, pidRead, nil
			}
		}
	}
	err := os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return false, 0, fmt.Errorf("os.WriteFile: %w", err)
	}
	return false, 0, nil
}
