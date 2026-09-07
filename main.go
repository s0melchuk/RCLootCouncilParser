// Command rclootparser scans RCLootCouncil's own data (SavedVariables +,
// optionally, live chat) and syncs loot awards to RCLootCouncilApi.
// See README.md for setup and the `init` command for a starting config.
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/s0melchuk/RCLootCouncilParser/internal/apiclient"
	"github.com/s0melchuk/RCLootCouncilParser/internal/config"
	"github.com/s0melchuk/RCLootCouncilParser/internal/state"
	rcsync "github.com/s0melchuk/RCLootCouncilParser/internal/sync"
)

const defaultConfigPath = "config.json"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		cmdInit()
	case "sync-once":
		cmdSyncOnce()
	case "watch":
		cmdWatch()
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `rclootparser — syncs RCLootCouncil loot awards to RCLootCouncilApi

Usage:
  rclootparser init        Write a starter %s in the current directory
  rclootparser sync-once   Parse SavedVariables once and sync any new awards, then exit
  rclootparser watch       Run continuously: live chat-log tailing + periodic
                            SavedVariables reconciliation. Ctrl+C to stop.
`, defaultConfigPath)
}

func cmdInit() {
	if _, err := os.Stat(defaultConfigPath); err == nil {
		log.Fatalf("%s already exists — remove it first if you want to regenerate it", defaultConfigPath)
	}
	if err := os.WriteFile(defaultConfigPath, []byte(config.Example), 0o644); err != nil {
		log.Fatalf("write %s: %v", defaultConfigPath, err)
	}
	fmt.Printf("Wrote %s — edit it with your WoW paths and API key, then run `rclootparser sync-once`.\n", defaultConfigPath)
}

func loadEverything() (*config.Config, *apiclient.Client, *state.State) {
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		log.Fatalf("%v\n(run `rclootparser init` first if you haven't set up %s)", err, defaultConfigPath)
	}
	st, err := state.Load(cfg.StatePath)
	if err != nil {
		log.Fatalf("load state: %v", err)
	}
	client := apiclient.New(cfg.APIBaseURL, cfg.APIKey)
	return cfg, client, st
}

func cmdSyncOnce() {
	cfg, client, st := loadEverything()
	if err := rcsync.ReconcileSavedVariables(cfg, client, st); err != nil {
		log.Fatalf("sync failed: %v", err)
	}
	fmt.Println("Sync complete.")
}

func cmdWatch() {
	cfg, client, st := loadEverything()

	stop := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		close(stop)
	}()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := rcsync.WatchChatLog(cfg, client, st, stop); err != nil {
			log.Printf("chat log watcher stopped: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := rcsync.WatchSavedVariables(cfg, client, st, stop); err != nil {
			log.Printf("saved variables watcher stopped: %v", err)
		}
	}()

	log.Println("watching for new awards (Ctrl+C to stop)...")
	wg.Wait()
}
