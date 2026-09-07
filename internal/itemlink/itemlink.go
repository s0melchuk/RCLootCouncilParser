// Package itemlink decodes WoW chat item hyperlinks, e.g.:
//
//	|cffa335ee|Hitem:17182:0:0:0:0:0:0:0:0:0:0|h[Sulfuras, Hand of Ragnaros]|h|r
//
// The colored/hyperlink wrapper is locale-independent, so this is a reliable
// way to pull an item ID and display name out of any chat text or
// SavedVariables field that embeds one.
package itemlink

import "regexp"

var pattern = regexp.MustCompile(`\|Hitem:(\d+)(?::[^|]*)?\|h\[([^\]]+)\]\|h`)

// Decoded is the result of parsing one item link.
type Decoded struct {
	ItemID int
	Name   string
}

// Find returns the first item link in s, if any.
func Find(s string) (Decoded, bool) {
	m := pattern.FindStringSubmatch(s)
	if m == nil {
		return Decoded{}, false
	}
	id := 0
	for _, c := range m[1] {
		id = id*10 + int(c-'0')
	}
	return Decoded{ItemID: id, Name: m[2]}, true
}
