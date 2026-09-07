// Package model holds the shared shape of a loot award, matching the
// RCLootCouncilApi ingest schema (see migrations/0001_init.sql in that repo).
package model

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

// Award is one loot record, ready to POST to /api/loot.
type Award struct {
	AwardedAt string  `json:"awarded_at"` // ISO 8601
	Raid      string  `json:"raid,omitempty"`
	Boss      string  `json:"boss,omitempty"`
	ItemID    int     `json:"item_id,omitempty"`
	ItemName  string  `json:"item_name"`
	Winner    string  `json:"winner"`
	Response  string  `json:"response,omitempty"`
	Votes     float64 `json:"votes,omitempty"`
	Note      string  `json:"note,omitempty"`
	RawSource string  `json:"raw_source,omitempty"`
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
