// Package model holds the shared shapes posted to RCLootCouncilApi, matching
// its ingest schema (see that repo's migrations/*.sql).
package model

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

// Award is one loot record, ready to POST to /api/loot.
type Award struct {
	AwardedAt  string  `json:"awarded_at"` // ISO 8601
	Raid       string  `json:"raid,omitempty"`
	Boss       string  `json:"boss,omitempty"`
	ItemID     int     `json:"item_id,omitempty"`
	ItemName   string  `json:"item_name"`
	Winner     string  `json:"winner"`
	Response   string  `json:"response,omitempty"`
	Difficulty string  `json:"difficulty,omitempty"` // e.g. "Normal", "Heroic" — derived from RCLootCouncil's "instance" field
	Slot       string  `json:"slot,omitempty"`       // NOT populated yet — see savedvars package doc
	Votes      float64 `json:"votes,omitempty"`
	Note       string  `json:"note,omitempty"`
	RawSource  string  `json:"raw_source,omitempty"`
}

// DedupeKey returns a stable hash used to decide whether this award has
// already been synced. RCLootCouncil's history entries have no unique ID of
// their own, so we derive one from the fields that together identify a
// single award.
func (a Award) DedupeKey() string {
	h := sha1.New()
	fmt.Fprintf(h, "%s|%s|%s|%d|%s", a.AwardedAt, a.Winner, a.ItemName, a.ItemID, a.Boss)
	return hex.EncodeToString(h.Sum(nil))
}

// Player is one roster entry, ready to POST to /api/players.
type Player struct {
	Name  string `json:"name"`
	Class string `json:"class,omitempty"`
	Spec  string `json:"spec,omitempty"` // not available from RCLootCouncil's history — always empty for now
}
