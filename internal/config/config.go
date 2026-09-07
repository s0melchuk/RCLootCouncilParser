// Package config loads the scanner's JSON config file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	// SavedVariablesPath points at the addon's SavedVariables file. WoW
	// writes ALL of an addon's declared SavedVariables into one file named
	// after the addon's folder — RCLootCouncilLootDB is not its own file,
	// it's a second top-level assignment inside RCLootCouncil.lua (verified
	// against a real client: the file contains both `RCLootCouncilDB = {...}`
	// and, further down, `RCLootCouncilLootDB = {...}`). So this should be:
	// "C:/Program Files (x86)/World of Warcraft/_classic_/WTF/Account/YOURACCOUNT/SavedVariables/RCLootCouncil.lua"
	SavedVariablesPath string `json:"saved_variables_path"`

	// ChatLogPath points at the WoW chat log file to tail — typically
	// Logs/WoWChatLog.txt, a single file appended across sessions (observed
	// behavior: WoWCombatLog.txt in the same Logs folder isn't
	// session-rotated either, just continuously appended — chat logging is
	// presumed to work the same way, though not yet confirmed against a
	// real WoWChatLog.txt). Requires running `/run LoggingChat(1)` in-game
	// every session, since it's a runtime toggle, not a saved setting.
	// Update this path if your setup ever does use a different filename.
	ChatLogPath string `json:"chat_log_path"`

	// AwardAnnouncePattern overrides chatlog.DefaultAwardPattern if your
	// guild customized RCLootCouncil's awardText. Leave empty to use the
	// default ("&p was awarded with &i for &r!").
	AwardAnnouncePattern string `json:"award_announce_pattern,omitempty"`

	APIBaseURL string `json:"api_base_url"`
	APIKey     string `json:"api_key"`

	// PollIntervalSeconds controls both chat-log tail polling and how often
	// the SavedVariables file's mtime is checked for reconciliation.
	PollIntervalSeconds int `json:"poll_interval_seconds"`

	// StatePath is where sync progress (chat log offset, dedupe keys) is
	// persisted. Defaults to "state.json" next to the config if empty.
	StatePath string `json:"state_path,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if c.PollIntervalSeconds <= 0 {
		c.PollIntervalSeconds = 15
	}
	if c.StatePath == "" {
		c.StatePath = "state.json"
	}
	if c.SavedVariablesPath == "" && c.ChatLogPath == "" {
		return nil, fmt.Errorf("config must set at least one of saved_variables_path or chat_log_path")
	}
	if c.APIBaseURL == "" {
		return nil, fmt.Errorf("config: api_base_url is required")
	}
	if c.APIKey == "" {
		return nil, fmt.Errorf("config: api_key is required")
	}
	return &c, nil
}

// Example is written out by `rclootparser init` as a starting point.
const Example = `{
  "saved_variables_path": "C:/Program Files (x86)/World of Warcraft/_classic_/WTF/Account/YOURACCOUNT/SavedVariables/RCLootCouncil.lua",
  "chat_log_path": "C:/Program Files (x86)/World of Warcraft/_classic_/Logs/WoWChatLog.txt",
  "api_base_url": "https://rclootcouncil-api.pages.dev",
  "api_key": "REPLACE_WITH_YOUR_INGEST_API_KEY",
  "poll_interval_seconds": 15
}
`
