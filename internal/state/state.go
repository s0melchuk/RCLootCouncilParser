// Package state persists what's already been synced, across restarts:
// the chat-log tail offset and the set of award dedupe keys already sent.
// Stored as a small JSON file next to the binary/config, not a database —
// this tool only ever has one writer and a few thousand rows at most.
package state

import (
	"encoding/json"
	"os"
	"sync"
)

type State struct {
	ChatLogOffset int64           `json:"chat_log_offset"`
	SyncedKeys    map[string]bool `json:"synced_keys"`

	path string
	mu   sync.Mutex
}

func Load(path string) (*State, error) {
	s := &State{path: path, SyncedKeys: map[string]bool{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, s); err != nil {
		return nil, err
	}
	if s.SyncedKeys == nil {
		s.SyncedKeys = map[string]bool{}
	}
	s.path = path
	return s, nil
}

func (s *State) save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}

func (s *State) GetChatLogOffset() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ChatLogOffset
}

func (s *State) SetChatLogOffset(offset int64) {
	s.mu.Lock()
	s.ChatLogOffset = offset
	err := s.save()
	s.mu.Unlock()
	if err != nil {
		// Best-effort persistence: a failed save here just means a restart
		// might re-read a few already-seen lines, which dedupe covers.
		_ = err
	}
}

// AlreadySynced reports whether key has been marked synced.
func (s *State) AlreadySynced(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.SyncedKeys[key]
}

// MarkSynced records key as synced and persists immediately.
func (s *State) MarkSynced(key string) error {
	s.mu.Lock()
	s.SyncedKeys[key] = true
	err := s.save()
	s.mu.Unlock()
	return err
}
