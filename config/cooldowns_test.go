package config

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0x2142/frigate-notify/models"
	"github.com/0x2142/frigate-notify/util"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func decodeCooldownYAML(t *testing.T, raw string) Config {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	k := koanf.New(".")
	if err := k.Load(structs.Provider(DefaultConfig, "koanf"), nil); err != nil {
		t.Fatal(err)
	}
	if err := k.Load(file.Provider(p), yaml.Parser()); err != nil {
		t.Fatal(err)
	}
	var c Config
	if err := k.Unmarshal("", &c); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCooldownYAMLNewKey(t *testing.T) {
	cfg := decodeCooldownYAML(t, `alerts:
  cooldowns:
    global: 15s
    camera:
      Garage_Innen: 2m
      Garage_Eingang: 90s
    label:
      Person: 30s
    zone:
      Innenhof: 1m
`)
	cfg.applyCooldownAlias()
	want := models.NotifyLimits{
		Global: "15s",
		Camera: map[string]string{"Garage_Innen": "2m", "Garage_Eingang": "90s"},
		Label: map[string]string{"Person": "30s"},
		Zone: map[string]string{"Innenhof": "1m"},
	}
	if !reflect.DeepEqual(cfg.Alerts.Cooldowns, want) {
		t.Fatalf("wrong parsed cooldowns: %#v; want %#v", cfg.Alerts.Cooldowns, want)
	}
	if got := cfg.validateCooldowns(); len(got) != 0 {
		t.Fatalf("unexpected errors: %v", got)
	}
}

func TestCooldownYAMLLegacyKeyMigrates(t *testing.T) {
	cfg := decodeCooldownYAML(t, `alerts:
  notify_limits:
    camera:
      Garage_Innen: 2m
`)
	if cfg.Alerts.LegacyNotifyLimits == nil {
		t.Fatal("deprecated key must still decode")
	}
	cfg.applyCooldownAlias()
	if got := cfg.Alerts.Cooldowns.Camera["Garage_Innen"]; got != "2m" {
		t.Fatalf("legacy cooldown lost: %q", got)
	}
	if cfg.Alerts.LegacyNotifyLimits != nil {
		t.Fatal("legacy key should not be serialized again")
	}
	raw, err := json.Marshal(cfg.Alerts)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("notify_limits")) {
		t.Fatalf("legacy key leaked into API response: %s", raw)
	}
}

func TestCooldownNewKeyWinsOverLegacy(t *testing.T) {
	cfg := decodeCooldownYAML(t, `alerts:
  notify_limits:
    camera:
      Garage_Innen: 5m
  cooldowns:
    camera:
      Garage_Innen: 2m
`)
	cfg.applyCooldownAlias()
	if got := cfg.Alerts.Cooldowns.Camera["Garage_Innen"]; got != "2m" {
		t.Fatalf("new key lost priority: %q", got)
	}
}

func TestCooldownWarningForCaseSensitiveNames(t *testing.T) {
	var known frigateCooldownConfig
	err := json.Unmarshal([]byte(`{
	  "cameras": {
	    "Garage_Innen": {"zones": {"Innen": {}, "Gehweg": {}}},
	    "Garage_Eingang": {"zones": {"Innenhof": {}, "Einfahrt": {}}},
	    "Garten": {"zones": {"Terrasse": {}}}
	  }
	}`), &known)
	if err != nil {
		t.Fatal(err)
	}
	if len(known.Cameras) != 3 {
		t.Fatalf("bad fixture: %d cameras", len(known.Cameras))
	}
	limits := models.NotifyLimits{
		Camera: map[string]string{"garage_innen": "2m", "Garage_Eingang": "90s"},
		Zone: map[string]string{"innenhof": "2m", "Einfahrt": "30s"},
	}
	warnings := unknownCooldownNames(limits, known)
	want := []cooldownNameWarning{
		{Scope: "camera", Configured: "garage_innen", Suggestion: "Garage_Innen"},
		{Scope: "zone", Configured: "innenhof", Suggestion: "Innenhof"},
	}
	if !reflect.DeepEqual(warnings, want) {
		t.Fatalf("warnings: %#v; want %#v", warnings, want)
	}
}

func TestCooldownTypoSuggestion(t *testing.T) {
	names := map[string]bool{"Garage_Innen": true, "Garage_Eingang": true}
	if got := suggestCooldownName("Garage_Inen", names); got != "Garage_Innen" {
		t.Fatalf("unexpected suggestion: %q", got)
	}
	if got := suggestCooldownName("entirely_missing", names); got != "" {
		t.Fatalf("unexpected remote suggestion: %q", got)
	}
}

func TestCooldownStartupNameCheckUsesFrigateAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config" {
			t.Errorf("unexpected API path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"cameras":{"Garage_Innen":{"zones":{"Innen":{}}}}}`))
	}))
	defer server.Close()

	// Capture only this diagnostic; restore global logger and HTTP state afterwards.
	oldLogger := log.Logger
	oldTimeout := util.HTTPTimeout
	oldAuth := util.AuthEnabled
	oldServer := util.FrigateServer
	t.Cleanup(func() {
		log.Logger = oldLogger
		util.HTTPTimeout = oldTimeout
		util.AuthEnabled = oldAuth
		util.FrigateServer = oldServer
	})
	util.HTTPTimeout = 5
	util.AuthEnabled = false
	util.FrigateServer = server.URL
	var output bytes.Buffer
	log.Logger = zerolog.New(&output)
	cfg := Config{
		Frigate: models.Frigate{Server: server.URL},
		Alerts: models.Alerts{Cooldowns: models.NotifyLimits{
			Camera: map[string]string{"garage_innen": "2m"},
			Zone: map[string]string{"innen": "1m"},
		}},
	}
	warnUnknownCooldownNames(&cfg)
	text := output.String()
	for _, expected := range []string{`"scope":"camera"`, `"suggested_name":"Garage_Innen"`, `"scope":"zone"`, `"suggested_name":"Innen"`} {
		if !strings.Contains(text, expected) {
			t.Errorf("missing %s in logs: %s", expected, text)
		}
	}
}
