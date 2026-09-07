// Package savedvars reads RCLootCouncil's own persisted loot history out of
// the addon's SavedVariables file, rather than scraping chat text.
//
// Note: WoW writes every SavedVariable an addon declares into ONE file named
// after the addon's folder — WTF/Account/<ACCOUNT>/SavedVariables/RCLootCouncil.lua
// — as multiple top-level assignments. RCLootCouncilLootDB is not its own
// file; it's a second `RCLootCouncilLootDB = { ... }` assignment inside that
// same RCLootCouncil.lua, appearing after the much larger `RCLootCouncilDB`
// (addon settings/profile) assignment. This has been confirmed against a
// real client. Point ParseFile at RCLootCouncil.lua — luatable.ParseAssignment
// finds the right top-level variable by name regardless of what else is in
// the file.
//
// The addon writes the history table as:
//
//	RCLootCouncilLootDB = {
//	    ["Faction - Realm"] = {
//	        ["PlayerName"] = {
//	            { ["lootWon"] = "...", ["date"] = "07/09/26", ["time"] = "20:14:32", ... },
//	            ...
//	        },
//	    },
//	}
//
// (see RCLootCouncil's ml_core.lua:TrackAndLogLoot, which builds exactly
// these fields: lootWon, date, time, instance, boss, votes, response,
// responseID, class, isAwardReason — note there is no equip-slot field here.
// `equipLoc` only exists on the transient in-session loot table used while
// the loot window is open; it is never written to history. So Award.Slot
// can't be populated from this source — it would need a separate itemID ->
// equip-slot lookup table, which is a deliberate follow-up, not guessed
// here.)
//
// This file is only flushed to disk on logout or /reload — there is no way
// around that from outside the game client — so treat reads of it as a
// periodic reconciliation pass, not a live feed.
package savedvars

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/s0melchuk/RCLootCouncilParser/internal/itemlink"
	"github.com/s0melchuk/RCLootCouncilParser/internal/luatable"
	"github.com/s0melchuk/RCLootCouncilParser/internal/model"
)

const savedVariableName = "RCLootCouncilLootDB"

// Parsed is the result of parsing one RCLootCouncilLootDB.lua file.
type Parsed struct {
	Awards []model.Award
	// Players is deduped by name, built from the "class" field each history
	// entry carries for its winner. Spec isn't available from this data.
	Players []model.Player
}

// ParseFile reads and decodes the given RCLootCouncilLootDB.lua path across
// every faction-realm and player bucket it contains.
func ParseFile(path string) (Parsed, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Parsed{}, fmt.Errorf("read %s: %w", path, err)
	}
	root, err := luatable.ParseAssignment(string(data), savedVariableName)
	if err != nil {
		return Parsed{}, fmt.Errorf("parse %s: %w", path, err)
	}
	rootTable, ok := root.(luatable.Table)
	if !ok {
		return Parsed{}, fmt.Errorf("%s: %s was not a table", path, savedVariableName)
	}

	var result Parsed
	seenPlayers := map[string]bool{}
	for _, factionRealmVal := range rootTable {
		factionRealm, ok := factionRealmVal.(luatable.Table)
		if !ok {
			continue
		}
		for playerKey, entriesVal := range factionRealm {
			playerName, ok := playerKey.(string)
			if !ok {
				continue
			}
			entries, ok := entriesVal.(luatable.Table)
			if !ok {
				continue
			}
			for _, entryVal := range entries {
				entry, ok := entryVal.(luatable.Table)
				if !ok {
					continue
				}
				award, class, ok := toAward(playerName, entry)
				if !ok {
					continue
				}
				result.Awards = append(result.Awards, award)
				if class != "" && !seenPlayers[playerName] {
					seenPlayers[playerName] = true
					result.Players = append(result.Players, model.Player{Name: playerName, Class: class})
				}
			}
		}
	}
	return result, nil
}

func toAward(player string, entry luatable.Table) (award model.Award, class string, ok bool) {
	lootWon, _ := entry.GetString("lootWon")
	if lootWon == "" {
		return model.Award{}, "", false
	}
	decoded, hasLink := itemlink.Find(lootWon)
	itemName := lootWon
	itemID := 0
	if hasLink {
		itemName = decoded.Name
		itemID = decoded.ItemID
	}

	dateStr, _ := entry.GetString("date") // "DD/MM/YY"
	timeStr, _ := entry.GetString("time") // "HH:MM:SS"
	awardedAt := combineDateTime(dateStr, timeStr)

	boss, _ := entry.GetString("boss")
	instance, _ := entry.GetString("instance")
	raid, difficulty := splitInstance(instance)
	response, _ := entry.GetString("response")
	votes, _ := entry.GetNumber("votes")
	class, _ = entry.GetString("class")

	return model.Award{
		AwardedAt:  awardedAt,
		Raid:       raid,
		Boss:       boss,
		ItemID:     itemID,
		ItemName:   itemName,
		Winner:     player,
		Response:   response,
		Difficulty: difficulty,
		Votes:      votes,
		RawSource:  lootWon,
	}, class, true
}

// splitInstance undoes RCLootCouncil's own
// `instance = instanceName.."-"..difficultyName` concatenation
// (ml_core.lua:TrackAndLogLoot) to recover raid name and difficulty
// separately. Splits on the *last* hyphen, since difficulty names (e.g.
// "10 Player Normal") don't contain one, while a small number of raid names
// theoretically could.
func splitInstance(instance string) (raid, difficulty string) {
	idx := strings.LastIndex(instance, "-")
	if idx < 0 {
		return instance, ""
	}
	return instance[:idx], instance[idx+1:]
}

// combineDateTime turns RCLootCouncil's "DD/MM/YY" + "HH:MM:SS" history
// fields into an ISO 8601 timestamp. Falls back to the raw concatenation if
// the format ever doesn't match (e.g. a locale variant), so a parse hiccup
// degrades gracefully instead of dropping the record.
//
// Note: these fields are the WoW client's local wall-clock time with no
// timezone attached. time.Parse defaults to UTC, so the resulting timestamp
// is labeled "Z" but is really "whatever timezone the game machine was set
// to" — fine for display/sorting, but don't treat it as true UTC.
func combineDateTime(dateStr, timeStr string) string {
	t, err := time.Parse("02/01/06 15:04:05", strings.TrimSpace(dateStr)+" "+strings.TrimSpace(timeStr))
	if err != nil {
		return strings.TrimSpace(dateStr + " " + timeStr)
	}
	return t.Format(time.RFC3339)
}
