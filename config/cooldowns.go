package config

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/0x2142/frigate-notify/models"
	"github.com/0x2142/frigate-notify/util"
	"github.com/rs/zerolog/log"
)

// applyCooldownAlias migrates the legacy key after decoding the configuration.
// Both the JSON config API and YAML configuration support the legacy alias.
// The canonical key always wins if both contain values.
func (c *Config) applyCooldownAlias() {
	legacy := c.Alerts.LegacyNotifyLimits
	if legacy == nil {
		return
	}
	if cooldownsEmpty(c.Alerts.Cooldowns) {
		c.Alerts.Cooldowns = *legacy
		log.Warn().Msg("Deprecated config key alerts.notify_limits: rename it to alerts.cooldowns")
	} else {
		log.Warn().Msg("Both alerts.cooldowns and deprecated alerts.notify_limits are set; using alerts.cooldowns only")
	}
	// Preserve a single canonical configuration for API replies and saves.
	c.Alerts.LegacyNotifyLimits = nil
}

func cooldownsEmpty(limits models.NotifyLimits) bool {
	return limits.Global == "" && len(limits.Camera) == 0 && len(limits.Label) == 0 && len(limits.Zone) == 0
}

// frigateCooldownConfig only decodes the relevant names from Frigate's
// /api/config response. It is compatible with event-based Frigate 0.13.
type frigateCooldownConfig struct {
	Cameras map[string]struct {
		Zones map[string]json.RawMessage `json:"zones"`
	} `json:"cameras"`
}

type cooldownNameWarning struct {
	Scope, Configured, Suggestion string
}

// warnUnknownCooldownNames makes a single best-effort startup check.
// This never prevents notifications: Frigate config may be temporarily
// unavailable even when the event MQTT connection is healthy.
func warnUnknownCooldownNames(c *Config) {
	if len(c.Alerts.Cooldowns.Camera) == 0 && len(c.Alerts.Cooldowns.Zone) == 0 {
		return
	}
	url := strings.TrimRight(c.Frigate.Server, "/") + "/api/config"
	data, err := util.HTTPGet(url, c.Frigate.Insecure, "", c.Frigate.Headers...)
	if err != nil {
		log.Warn().Err(err).Msg("Could not check cooldown camera/zone names against Frigate /api/config; skipping name validation")
		return
	}
	var known frigateCooldownConfig
	if err := json.Unmarshal(data, &known); err != nil || len(known.Cameras) == 0 {
		log.Warn().Err(err).Msg("Could not read camera/zone names from Frigate /api/config; skipping name validation")
		return
	}
	for _, warning := range unknownCooldownNames(c.Alerts.Cooldowns, known) {
		entry := log.Warn().Str("scope", warning.Scope).Str("configured_name", warning.Configured)
		if warning.Suggestion != "" {
			entry = entry.Str("suggested_name", warning.Suggestion)
		}
		entry.Msg("Unknown cooldown name in Frigate configuration (case-sensitive)")
	}
}

func unknownCooldownNames(limits models.NotifyLimits, frigate frigateCooldownConfig) []cooldownNameWarning {
	knownCameras := make(map[string]bool, len(frigate.Cameras))
	knownZones := make(map[string]bool)
	for cam, conf := range frigate.Cameras {
		knownCameras[cam] = true
		for zone := range conf.Zones {
			knownZones[zone] = true
		}
	}
	var warnings []cooldownNameWarning
	for _, entry := range []struct {
		scope string
		configured map[string]string
		known map[string]bool
	}{
		{"camera", limits.Camera, knownCameras},
		{"zone", limits.Zone, knownZones},
	} {
		names := make([]string, 0, len(entry.configured))
		for name := range entry.configured {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if entry.known[name] {
				continue
			}
			warnings = append(warnings, cooldownNameWarning{
				Scope: entry.scope, Configured: name,
				Suggestion: suggestCooldownName(name, entry.known),
			})
		}
	}
	return warnings
}

// Exact case-insensitive matches take precedence over fuzzy typo corrections.
// We never auto-correct names, since that could silently change alert behavior.
func suggestCooldownName(want string, known map[string]bool) string {
	keys := make([]string, 0, len(known))
	for key := range known { keys = append(keys, key) }
	sort.Strings(keys)
	for _, key := range keys {
		if strings.EqualFold(want, key) {
			return key
		}
	}
	best, bestDistance := "", 100000
	lowerWant := []rune(strings.ToLower(want))
	for _, key := range keys {
		dist := cooldownEditDistance(lowerWant, []rune(strings.ToLower(key)))
		if dist < bestDistance {
			best, bestDistance = key, dist
		}
	}
	maxDistance := 2
	if len(lowerWant)/4 > maxDistance { maxDistance = len(lowerWant)/4 }
	if bestDistance > maxDistance { return "" }
	return best
}

func cooldownEditDistance(a, b []rune) int {
	previous := make([]int, len(b)+1)
	for j := range previous { previous[j] = j }
	for i, x := range a {
		row := make([]int, len(b)+1)
		row[0] = i+1
		for j, y := range b {
			cost := 0
			if x != y { cost = 1 }
			row[j+1] = min(row[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = row
	}
	return previous[len(b)]
}

// This is used in tests and diagnostic logs for predictable output.
func (w cooldownNameWarning) String() string {
	if w.Suggestion != "" {
		return fmt.Sprintf("%s %q not found (did you mean %q?)", w.Scope, w.Configured, w.Suggestion)
	}
	return fmt.Sprintf("%s %q not found", w.Scope, w.Configured)
}
