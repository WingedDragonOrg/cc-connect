package core

import (
	"sync"
	"time"
)

const (
	defaultChannelHistoryLimit = 20
	defaultMaxHistoryChannels  = 1000
)

// ChannelHistoryStore is a thread-safe, per-channel message history store
// with LRU eviction. Platforms use it to record recent channel messages so
// the engine can inject them as context when sending a prompt to the agent.
type ChannelHistoryStore struct {
	mu      sync.Mutex
	data    map[string][]ChannelHistoryEntry
	order   []string // insertion order for LRU eviction
	limit   int      // max entries per channel
	maxKeys int      // max number of channels tracked
}

// NewChannelHistoryStore creates a store. limit is the max messages per
// channel; maxKeys is the max number of channels tracked (LRU eviction).
// Zero or negative values fall back to defaults (20 / 1000).
func NewChannelHistoryStore(limit, maxKeys int) *ChannelHistoryStore {
	if limit <= 0 {
		limit = defaultChannelHistoryLimit
	}
	if maxKeys <= 0 {
		maxKeys = defaultMaxHistoryChannels
	}
	return &ChannelHistoryStore{
		data:    make(map[string][]ChannelHistoryEntry),
		limit:   limit,
		maxKeys: maxKeys,
	}
}

// Record appends a message to the channel's history.
func (s *ChannelHistoryStore) Record(channelID, sender, body string) {
	if channelID == "" || body == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := ChannelHistoryEntry{
		Sender:    sender,
		Body:      body,
		Timestamp: time.Now(),
	}

	history := s.data[channelID]
	history = append(history, entry)
	if len(history) > s.limit {
		history = history[len(history)-s.limit:]
	}
	s.data[channelID] = history

	// Refresh LRU order.
	s.refreshOrder(channelID)
	s.evict()
}

// Get returns a copy of the channel's history entries.
func (s *ChannelHistoryStore) Get(channelID string) []ChannelHistoryEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	history := s.data[channelID]
	if len(history) == 0 {
		return nil
	}
	out := make([]ChannelHistoryEntry, len(history))
	copy(out, history)
	return out
}

// Clear removes all history for the given channel.
func (s *ChannelHistoryStore) Clear(channelID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, channelID)
	s.removeFromOrder(channelID)
}

// refreshOrder moves channelID to the end of the order slice (most recent).
func (s *ChannelHistoryStore) refreshOrder(channelID string) {
	s.removeFromOrder(channelID)
	s.order = append(s.order, channelID)
}

func (s *ChannelHistoryStore) removeFromOrder(channelID string) {
	for i, id := range s.order {
		if id == channelID {
			s.order = append(s.order[:i], s.order[i+1:]...)
			return
		}
	}
}

// evict removes the oldest channels when the number of channels exceeds maxKeys.
func (s *ChannelHistoryStore) evict() {
	for len(s.order) > s.maxKeys {
		oldest := s.order[0]
		s.order = s.order[1:]
		delete(s.data, oldest)
	}
}
