// Package sync wires the two data sources (SavedVariables reconciliation,
// live chat-log tailing) to the API client and local dedupe state.
package sync

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/s0melchuk/RCLootCouncilParser/internal/apiclient"
	"github.com/s0melchuk/RCLootCouncilParser/internal/chatlog"
	"github.com/s0melchuk/RCLootCouncilParser/internal/config"
	"github.com/s0melchuk/RCLootCouncilParser/internal/model"
	"github.com/s0melchuk/RCLootCouncilParser/internal/savedvars"
	"github.com/s0melchuk/RCLootCouncilParser/internal/state"
)

// ReconcileSavedVariables parses the SavedVariables file (if configured) and
// pushes any awards not already recorded in st. Safe to call repeatedly —
// dedupe is keyed by award content, not file position.
func ReconcileSavedVariables(cfg *config.Config, client *apiclient.Client, st *state.State) error {
	if cfg.SavedVariablesPath == "" {
		return nil
	}
	awards, err := savedvars.ParseFile(cfg.SavedVariablesPath)
	if err != nil {
		return fmt.Errorf("parse saved variables: %w", err)
	}
	return pushNew(client, st, awards)
}

func pushNew(client *apiclient.Client, st *state.State, awards []model.Award) error {
	var toSend []model.Award
	var keys []string
	for _, a := range awards {
		key := a.DedupeKey()
		if st.AlreadySynced(key) {
			continue
		}
		toSend = append(toSend, a)
		keys = append(keys, key)
	}
	if len(toSend) == 0 {
		return nil
	}
	log.Printf("syncing %d new award(s)", len(toSend))
	if err := client.PostAwards(toSend); err != nil {
		return err
	}
	for _, key := range keys {
		if err := st.MarkSynced(key); err != nil {
			return fmt.Errorf("mark synced: %w", err)
		}
	}
	return nil
}

// WatchChatLog tails the chat log (if configured) forever, pushing each
// award announcement it finds as soon as it appears. Runs until stop is
// closed.
func WatchChatLog(cfg *config.Config, client *apiclient.Client, st *state.State, stop <-chan struct{}) error {
	if cfg.ChatLogPath == "" {
		return nil
	}
	matcher, err := chatlog.NewMatcher(cfg.AwardAnnouncePattern)
	if err != nil {
		return err
	}
	tailer := &chatlog.Tailer{
		Path:      cfg.ChatLogPath,
		PollEvery: time.Duration(cfg.PollIntervalSeconds) * time.Second,
		GetOffset: st.GetChatLogOffset,
		SetOffset: st.SetChatLogOffset,
	}
	return tailer.Run(stop, func(line string) {
		award, ok := matcher.ParseLine(line)
		if !ok {
			return
		}
		if err := pushNew(client, st, []model.Award{award}); err != nil {
			log.Printf("failed to sync live award for %s: %v", award.Winner, err)
		}
	})
}

// WatchSavedVariables polls the SavedVariables file's mtime (if configured)
// and reconciles whenever it changes — i.e. whenever a natural
// logout/reload happens, without this tool ever forcing one. Runs until
// stop is closed.
func WatchSavedVariables(cfg *config.Config, client *apiclient.Client, st *state.State, stop <-chan struct{}) error {
	if cfg.SavedVariablesPath == "" {
		return nil
	}
	var lastMod time.Time
	interval := time.Duration(cfg.PollIntervalSeconds) * time.Second
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		if info, err := os.Stat(cfg.SavedVariablesPath); err == nil {
			if info.ModTime().After(lastMod) {
				lastMod = info.ModTime()
				if err := ReconcileSavedVariables(cfg, client, st); err != nil {
					log.Printf("saved variables reconcile failed: %v", err)
				}
			}
		}
		select {
		case <-stop:
			return nil
		case <-time.After(interval):
		}
	}
}
