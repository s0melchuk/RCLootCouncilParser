// Package savedvars reads RCLootCouncil's own persisted loot history out of
// the addon's SavedVariables file, rather than scraping chat text. The addon
// writes it as:
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
// these fields). This file is only flushed to disk on logout or /reload —
// there is no way around that from outside the game client — so treat reads
// of it as a periodic reconciliation pass, not a live feed.
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

// ParseFile reads and decodes the given RCLootCouncilLootDB.lua path into a
// flat list of awards, across every faction-realm and player bucket it
// contains.
func ParseFile(path string) ([]model.Award, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	root, err := luatable.ParseAssignment(string(data), savedVariableName)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	rootTable, ok := root.(luatable.Table)
	if !ok {
		return nil, fmt.Errorf("%s: %s was not a table", path, savedVariableName)
	}

	var awards []model.Award
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
				award, ok := toAward(playerName, entry)
				if ok {
					awards = append(awards, award)
				}
			}
		}
	}
	return awards, nil
}

func toAward(player string, entry luatable.Table) (model.Award, bool) {
	lootWon, _ := entry.GetString("lootWon")
	if lootWon == "" {
		return model.Award{}, false
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
	response, _ := entry.GetString("response")
	votes, _ := entry.GetNumber("votes")

	return model.Award{
		AwardedAt: awardedAt,
		Raid:      instance,
		Boss:      boss,
		ItemID:    itemID,
		ItemName:  itemName,
		Winner:    player,
		Response:  response,
		Votes:     votes,
		RawSource: lootWon,
	}, true
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
