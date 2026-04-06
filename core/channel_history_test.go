package core

import (
	"fmt"
	"sync"
	"testing"
)

func TestChannelHistoryStore_RecordAndGet(t *testing.T) {
	s := NewChannelHistoryStore(3, 100)

	s.Record("ch1", "alice", "hello")
	s.Record("ch1", "bob", "world")

	got := s.Get("ch1")
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	if got[0].Sender != "alice" || got[0].Body != "hello" {
		t.Errorf("entry[0] = %+v", got[0])
	}
	if got[1].Sender != "bob" || got[1].Body != "world" {
		t.Errorf("entry[1] = %+v", got[1])
	}
}

func TestChannelHistoryStore_Limit(t *testing.T) {
	s := NewChannelHistoryStore(2, 100)

	s.Record("ch1", "a", "1")
	s.Record("ch1", "b", "2")
	s.Record("ch1", "c", "3")

	got := s.Get("ch1")
	if len(got) != 2 {
		t.Fatalf("expected 2 entries (limit), got %d", len(got))
	}
	// Oldest should be evicted.
	if got[0].Body != "2" || got[1].Body != "3" {
		t.Errorf("expected [2,3], got [%s,%s]", got[0].Body, got[1].Body)
	}
}

func TestChannelHistoryStore_Clear(t *testing.T) {
	s := NewChannelHistoryStore(5, 100)

	s.Record("ch1", "a", "msg")
	s.Clear("ch1")

	got := s.Get("ch1")
	if len(got) != 0 {
		t.Fatalf("expected empty after clear, got %d", len(got))
	}
}

func TestChannelHistoryStore_LRUEviction(t *testing.T) {
	s := NewChannelHistoryStore(5, 3)

	s.Record("ch1", "a", "1")
	s.Record("ch2", "b", "2")
	s.Record("ch3", "c", "3")
	// ch1 is the oldest; adding ch4 should evict ch1.
	s.Record("ch4", "d", "4")

	if got := s.Get("ch1"); len(got) != 0 {
		t.Errorf("ch1 should be evicted, got %d entries", len(got))
	}
	if got := s.Get("ch4"); len(got) != 1 {
		t.Errorf("ch4 should have 1 entry, got %d", len(got))
	}
}

func TestChannelHistoryStore_LRURefresh(t *testing.T) {
	s := NewChannelHistoryStore(5, 3)

	s.Record("ch1", "a", "1")
	s.Record("ch2", "b", "2")
	s.Record("ch3", "c", "3")
	// Touch ch1 to refresh it.
	s.Record("ch1", "a", "1b")
	// Now ch2 is the oldest; adding ch4 should evict ch2.
	s.Record("ch4", "d", "4")

	if got := s.Get("ch2"); len(got) != 0 {
		t.Errorf("ch2 should be evicted, got %d entries", len(got))
	}
	if got := s.Get("ch1"); len(got) != 2 {
		t.Errorf("ch1 should have 2 entries (refreshed), got %d", len(got))
	}
}

func TestChannelHistoryStore_GetReturnsCopy(t *testing.T) {
	s := NewChannelHistoryStore(5, 100)
	s.Record("ch1", "a", "msg")

	got := s.Get("ch1")
	got[0].Body = "mutated"

	original := s.Get("ch1")
	if original[0].Body == "mutated" {
		t.Error("Get should return a copy, not a reference")
	}
}

func TestChannelHistoryStore_EmptyInputs(t *testing.T) {
	s := NewChannelHistoryStore(5, 100)

	s.Record("", "a", "msg")   // empty channel, should be ignored
	s.Record("ch1", "a", "")   // empty body, should be ignored
	s.Record("ch1", "", "msg") // empty sender is OK

	got := s.Get("ch1")
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}
	if got[0].Sender != "" || got[0].Body != "msg" {
		t.Errorf("unexpected entry: %+v", got[0])
	}
}

func TestChannelHistoryStore_GetEmptyChannel(t *testing.T) {
	s := NewChannelHistoryStore(5, 100)
	got := s.Get("nonexistent")
	if got != nil {
		t.Errorf("expected nil for nonexistent channel, got %v", got)
	}
}

func TestChannelHistoryStore_ConcurrentAccess(t *testing.T) {
	s := NewChannelHistoryStore(10, 100)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ch := fmt.Sprintf("ch%d", i%5)
			s.Record(ch, "user", fmt.Sprintf("msg%d", i))
			s.Get(ch)
			if i%10 == 0 {
				s.Clear(ch)
			}
		}(i)
	}
	wg.Wait()
}

func TestNewChannelHistoryStore_Defaults(t *testing.T) {
	s := NewChannelHistoryStore(0, 0)
	if s.limit != defaultChannelHistoryLimit {
		t.Errorf("expected default limit %d, got %d", defaultChannelHistoryLimit, s.limit)
	}
	if s.maxKeys != defaultMaxHistoryChannels {
		t.Errorf("expected default maxKeys %d, got %d", defaultMaxHistoryChannels, s.maxKeys)
	}
}
