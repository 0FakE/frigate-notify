package eventlog

import (
	"bytes"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestReasonAndEventIndependence(t *testing.T) {
	var d tracker
	start := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	if !d.firstAt("a", "cooldown", start) { t.Fatal("first occurrence should be INFO") }
	if d.firstAt("a", "cooldown", start.Add(time.Second)) { t.Fatal("repeated decision should be DEBUG") }
	if !d.firstAt("a", "notify_once", start.Add(time.Second)) { t.Fatal("different reason should be INFO") }
	if !d.firstAt("b", "cooldown", start.Add(time.Second)) { t.Fatal("different event should be INFO") }
	if !d.firstAt("a", "cooldown", start.Add(time.Hour)) { t.Fatal("expired decision should be INFO") }
	if !d.firstAt("", "cooldown", start) || !d.firstAt("", "cooldown", start) { t.Fatal("missing ID should not be shared") }
}

func TestDecisionLogLevels(t *testing.T) {
	var output bytes.Buffer
	prevLogger, prevLevel := log.Logger, zerolog.GlobalLevel()
	decisions.mu.Lock()
	oldSeen := decisions.seen
	decisions.seen = nil
	decisions.mu.Unlock()
	defer func() {
		log.Logger = prevLogger
		zerolog.SetGlobalLevel(prevLevel)
		decisions.mu.Lock()
		decisions.seen = oldSeen
		decisions.mu.Unlock()
	}()
	log.Logger = zerolog.New(&output)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	Decision("e", "cooldown").Msg("Notification suppressed by cooldown")
	Decision("e", "cooldown").Msg("Notification suppressed by cooldown")
	if count := strings.Count(output.String(), "Notification suppressed by cooldown"); count != 1 {
		t.Fatalf("expected one INFO occurrence, got %d: %s", count, output.String())
	}
	output.Reset()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	Decision("new", "cooldown").Msg("Notification suppressed by cooldown")
	Decision("new", "cooldown").Msg("Notification suppressed by cooldown")
	if count := strings.Count(output.String(), "Notification suppressed by cooldown"); count != 2 {
		t.Fatalf("DEBUG must include both occurrences, got %d: %s", count, output.String())
	}
}

func TestConcurrentDecisionDeduplication(t *testing.T) {
	var d tracker
	var first atomic.Int32
	now := time.Now()
	var wg sync.WaitGroup
	for i:=0; i<50; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); if d.firstAt("same","cooldown",now) { first.Add(1) } }()
	}
	wg.Wait()
	if got:=first.Load(); got!=1 { t.Fatalf("expected exactly one first decision; got %d", got) }
}

func TestBoundedStorage(t *testing.T) {
	var d tracker
	now := time.Now()
	for i:=0; i<maxEntries+25; i++ {
		// Use distinct reasons for a single event to verify bound independently of IDs.
		if !d.firstAt("same", time.Unix(int64(i),0).String(),now) { t.Fatal("new reason rejected") }
	}
	if len(d.seen)>maxEntries { t.Fatalf("cache grew to %d entries",len(d.seen)) }
}
