package savedvars

import (
	"os"
	"path/filepath"
	"testing"
)

// sample mirrors the exact shape RCLootCouncil's AceDB-3.0 factionrealm
// history table + ml_core.lua's TrackAndLogLoot fields produce.
const sample = `
RCLootCouncilLootDB = {
	["Alliance - TestRealm"] = {
		["Thrallpull"] = {
			[1] = {
				["lootWon"] = "|cffa335ee|Hitem:17182::::::::60:::::|h[Sulfuras, Hand of Ragnaros]|h|r",
				["date"] = "07/09/26",
				["time"] = "20:14:32",
				["instance"] = "Molten Core-Normal",
				["boss"] = "Ragnaros",
				["votes"] = 5,
				["response"] = "Main Spec",
				["responseID"] = 1,
				["class"] = "WARRIOR",
				["isAwardReason"] = false,
			},
		},
		["Healbot"] = {
			[1] = {
				["lootWon"] = "|cff0070dd|Hitem:19379::::::::60:::::|h[Cloak of the Shrouded Mists]|h|r",
				["date"] = "07/09/26",
				["time"] = "20:20:00",
				["instance"] = "Molten Core-Normal",
				["boss"] = "Majordomo Executus",
				["votes"] = 3,
				["response"] = "Off Spec",
				["responseID"] = 2,
				["class"] = "PRIEST",
				["isAwardReason"] = false,
			},
		},
	},
}
`

func TestParseFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "RCLootCouncilLootDB.lua")
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}

	parsed, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(parsed.Awards) != 2 {
		t.Fatalf("expected 2 awards, got %d: %+v", len(parsed.Awards), parsed.Awards)
	}

	byWinner := map[string]bool{}
	for _, a := range parsed.Awards {
		byWinner[a.Winner] = true
		if a.ItemID == 0 {
			t.Errorf("award for %s: expected a decoded item ID, got 0", a.Winner)
		}
		if a.ItemName == "" || a.Boss == "" || a.Raid == "" || a.Response == "" {
			t.Errorf("award for %s missing fields: %+v", a.Winner, a)
		}
		if a.AwardedAt == "" {
			t.Errorf("award for %s: AwardedAt not set", a.Winner)
		}
		if a.Difficulty != "Normal" {
			t.Errorf("award for %s: Difficulty = %q, want Normal", a.Winner, a.Difficulty)
		}
		if a.Raid != "Molten Core" {
			t.Errorf("award for %s: Raid = %q, want Molten Core", a.Winner, a.Raid)
		}
	}
	if !byWinner["Thrallpull"] || !byWinner["Healbot"] {
		t.Fatalf("expected awards for Thrallpull and Healbot, got %+v", parsed.Awards)
	}

	players := map[string]string{}
	for _, p := range parsed.Players {
		players[p.Name] = p.Class
	}
	if players["Thrallpull"] != "WARRIOR" {
		t.Errorf("Thrallpull class = %q, want WARRIOR", players["Thrallpull"])
	}
	if players["Healbot"] != "PRIEST" {
		t.Errorf("Healbot class = %q, want PRIEST", players["Healbot"])
	}
}
