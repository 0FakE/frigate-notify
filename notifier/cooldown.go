package notifier

import (
	"fmt"
	"sync"
	"time"

	"github.com/0x2142/frigate-notify/models"
)

// cooldownKey uniquely identifies one configured notification scope.
type cooldownKey struct {
	scope string
	name  string
}

type cooldownRule struct {
	key      cooldownKey
	duration time.Duration
}

// cooldownTracker stores the time of the last admitted notification per rule.
// The state is intentionally in-memory: a restart resets all cooldowns.
type cooldownTracker struct {
	mu   sync.Mutex
	last map[cooldownKey]time.Time
}

var notificationCooldown cooldownTracker

// matchingCooldownRules expands all rules that apply to this notification.
// One review containing multiple detections still reserves each rule only once.
func matchingCooldownRules(events []models.Event, limits models.NotifyLimits) []cooldownRule {
	var rules []cooldownRule
	seen := make(map[cooldownKey]bool)
	add := func(scope, name, raw string) {
		if raw == "" {
			return
		}
		key := cooldownKey{scope: scope, name: name}
		if seen[key] {
			return
		}
		duration, err := time.ParseDuration(raw)
		if err != nil || duration <= 0 {
			return // Config validation rejects invalid entries at startup.
		}
		seen[key] = true
		rules = append(rules, cooldownRule{key: key, duration: duration})
	}
	add("global", "*", limits.Global)
	for _, event := range events {
		add("camera", event.Camera, limits.Camera[event.Camera])
		add("label", event.Label, limits.Label[event.Label])
		for _, zone := range event.CurrentZones {
			add("zone", zone, limits.Zone[zone])
		}
	}
	return rules
}

func (c *cooldownTracker) allow(events []models.Event, limits models.NotifyLimits) (bool, string) {
	return c.allowAt(events, limits, time.Now())
}

// allowAt atomically checks every matched rule, then reserves them all.
// A rejected notification never extends any cooldown.
func (c *cooldownTracker) allowAt(events []models.Event, limits models.NotifyLimits, now time.Time) (bool, string) {
	rules := matchingCooldownRules(events, limits)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, rule := range rules {
		last, exists := c.last[rule.key]
		if !exists {
			continue
		}
		until := last.Add(rule.duration)
		if now.Before(until) {
			return false, fmt.Sprintf("%s:%s (%s remaining)", rule.key.scope, rule.key.name, until.Sub(now).Round(time.Second))
		}
	}
	if c.last == nil {
		c.last = make(map[cooldownKey]time.Time)
	}
	for _, rule := range rules {
		c.last[rule.key] = now
	}
	return true, ""
}
