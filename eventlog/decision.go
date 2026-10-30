// Package eventlog keeps high-signal event decisions at INFO once per event,
// while allowing repeated MQTT updates to remain visible at DEBUG.
package eventlog

import (
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	retention = time.Hour
	maxEntries = 8192
)

type decisionKey struct {
	eventID string
	reason string
}

type tracker struct {
	mu sync.Mutex
	seen map[decisionKey]time.Time
}

var decisions tracker

// Decision reports the first occurrence of a (Frigate event ID, reason)
// at INFO, and subsequent occurrences at DEBUG. No notification state is
// changed, and a repeated event never extends the retention period.
func Decision(eventID, reason string) *zerolog.Event {
	if decisions.firstAt(eventID, reason, time.Now()) {
		return log.Info()
	}
	return log.Debug()
}

func (t *tracker) firstAt(eventID, reason string, now time.Time) bool {
	// Never deduplicate missing identifiers.
	if eventID == "" {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.seen == nil {
		t.seen = make(map[decisionKey]time.Time)
	}
	key := decisionKey{eventID: eventID, reason: reason}
	if first, found := t.seen[key]; found && now.Sub(first) < retention {
		return false
	}
	// Bound memory even for never-ending event streams. Evict expired
	// decisions first, then the oldest if the cap has been reached.
	if len(t.seen) >= maxEntries {
		var oldest decisionKey
		var oldestAt time.Time
		for k, first := range t.seen {
			if now.Sub(first) >= retention {
				delete(t.seen, k)
				continue
			}
			if oldestAt.IsZero() || first.Before(oldestAt) {
				oldest, oldestAt = k, first
			}
		}
		if len(t.seen) >= maxEntries {
			delete(t.seen, oldest)
		}
	}
	t.seen[key] = now
	return true
}
