// Package chatlog gives near-real-time award detection by tailing WoW's
// chat log file (enabled client-side by running `/run LoggingChat(1)` — this
// is a Lua API toggle, not a saved CVar, so it has to be re-run every
// session). Every SendChatMessage the addon sends lands in this file
// immediately, unlike SavedVariables which only flush on logout/reload —
// this is the only piece that can be live.
//
// We deliberately don't try to parse the full chat-log line format (its
// exact shape has varied across client versions and isn't worth pinning
// down). Instead we regex-search each line's raw text for RCLootCouncil's
// award-announcement message — the item link inside it is a fixed,
// locale-independent token, so matching just the announcement fragment is
// both simpler and more robust than parsing the whole log line.
package chatlog

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/s0melchuk/RCLootCouncilParser/internal/itemlink"
	"github.com/s0melchuk/RCLootCouncilParser/internal/model"
)

// DefaultAwardPattern matches RCLootCouncil's default award announcement:
// "&p was awarded with &i for &r!" -> "<player> was awarded with <item link> for <reason>!"
// If a guild customizes `awardText` in the addon, override this via config.
const DefaultAwardPattern = `(?P<player>[\p{L}'-]{2,24}) was awarded with (?P<item>\|c[0-9a-fA-F]{8}\|Hitem:\d+[^|]*\|h\[[^\]]+\]\|h\|r) for (?P<reason>[^!]+)!`

// Matcher finds award announcements in raw chat-log lines.
type Matcher struct {
	re *regexp.Regexp
}

func NewMatcher(pattern string) (*Matcher, error) {
	if pattern == "" {
		pattern = DefaultAwardPattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid award pattern: %w", err)
	}
	required := []string{"player", "item", "reason"}
	for _, name := range required {
		found := false
		for _, n := range re.SubexpNames() {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("award pattern must have a %q named group", name)
		}
	}
	return &Matcher{re: re}, nil
}

// ParseLine returns the award described by line, if it matches, along with
// the current wall-clock time as its timestamp (chat log lines are tailed
// live, so "now" is an accurate award time — the log's own date/time prefix
// format isn't parsed, see the package doc).
func (m *Matcher) ParseLine(line string) (model.Award, bool) {
	match := m.re.FindStringSubmatch(line)
	if match == nil {
		return model.Award{}, false
	}
	groups := map[string]string{}
	for i, name := range m.re.SubexpNames() {
		if name != "" && i < len(match) {
			groups[name] = match[i]
		}
	}

	award := model.Award{
		AwardedAt: time.Now().Format(time.RFC3339),
		Winner:    groups["player"],
		Response:  groups["reason"],
		ItemName:  groups["item"],
		RawSource: line,
	}
	if decoded, ok := itemlink.Find(groups["item"]); ok {
		award.ItemName = decoded.Name
		award.ItemID = decoded.ItemID
	}
	return award, true
}

// Tailer follows a growing text file, calling onLine for each newly
// appended, complete line. It remembers its byte offset across restarts via
// the offset get/set callbacks so a restart doesn't re-process (or miss)
// lines.
type Tailer struct {
	Path      string
	GetOffset func() int64
	SetOffset func(int64)
	PollEvery time.Duration
}

// Run blocks, polling Path for growth until stop is closed.
func (t *Tailer) Run(stop <-chan struct{}, onLine func(string)) error {
	if t.PollEvery <= 0 {
		t.PollEvery = 2 * time.Second
	}
	for {
		select {
		case <-stop:
			return nil
		default:
		}
		if err := t.pollOnce(onLine); err != nil && !os.IsNotExist(err) {
			return err
		}
		select {
		case <-stop:
			return nil
		case <-time.After(t.PollEvery):
		}
	}
}

func (t *Tailer) pollOnce(onLine func(string)) error {
	f, err := os.Open(t.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	offset := t.GetOffset()
	if offset > info.Size() {
		// File shrank (e.g. a new session's log rotated to a fresh file
		// reusing the same name) — start over from the beginning.
		offset = 0
	}
	if offset >= info.Size() {
		return nil // nothing new
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	reader := bufio.NewReader(f)
	newOffset := offset
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 && (err == nil || err == io.EOF) {
			if err == nil { // only count/consume complete lines
				newOffset += int64(len(line))
				onLine(line)
				continue
			}
		}
		break
	}
	t.SetOffset(newOffset)
	return nil
}
