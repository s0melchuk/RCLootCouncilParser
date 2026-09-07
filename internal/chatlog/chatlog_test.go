package chatlog

import "testing"

func TestParseLine_DefaultPattern(t *testing.T) {
	matcher, err := NewMatcher("")
	if err != nil {
		t.Fatal(err)
	}

	// A realistic line as it might appear in WoWChatLog.txt — we don't parse
	// the surrounding date/channel/from/to metadata at all, just search for
	// the announcement text, so the exact wrapper format doesn't matter here.
	line := `9/7 20:14:32.123  PARTY  "Thrallpull was awarded with |cffa335ee|Hitem:17182::::::::60:::::|h[Sulfuras, Hand of Ragnaros]|h|r for Main Spec!"  from  Ragefire`

	award, ok := matcher.ParseLine(line)
	if !ok {
		t.Fatalf("expected line to match, it didn't")
	}
	if award.Winner != "Thrallpull" {
		t.Errorf("Winner = %q, want Thrallpull", award.Winner)
	}
	if award.ItemName != "Sulfuras, Hand of Ragnaros" {
		t.Errorf("ItemName = %q, want Sulfuras, Hand of Ragnaros", award.ItemName)
	}
	if award.ItemID != 17182 {
		t.Errorf("ItemID = %d, want 17182", award.ItemID)
	}
	if award.Response != "Main Spec" {
		t.Errorf("Response = %q, want Main Spec", award.Response)
	}
}

func TestParseLine_NoMatch(t *testing.T) {
	matcher, err := NewMatcher("")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := matcher.ParseLine(`9/7 20:14:32.123  PARTY  "gz on loot!"  from  Ragefire`); ok {
		t.Fatalf("expected unrelated chat line not to match")
	}
}
