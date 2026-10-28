package config

import (
    "os"
    "path/filepath"
    "reflect"
    "testing"

    "github.com/knadh/koanf/parsers/yaml"
    "github.com/knadh/koanf/providers/file"
    "github.com/knadh/koanf/providers/structs"
    "github.com/knadh/koanf/v2"
)

func TestReproUserCameraNotifyLimitsFromYAML(t *testing.T) {
    sample := `app:
  mode: events
alerts:
  general:
    title: "{{ .Extra.CameraName }} -> {{ .Label }}"
    timeformat: 02.01.2006 15:04:05
    snap_bbox: true
    snap_timestamp: false
    snap_crop: false
    snap_hires: false
    notify_once: true
  labels:
    allow:
      - Person
      - Fahrzeug
  zones:
    block:
      - Fremdgrund
      - Gehweg
  notify_limits:
    camera:
      garage_innen: 2m
      garage_eingang: 90s
      garten: 1m
  pushover:
    enabled: true
`
    p := filepath.Join(t.TempDir(), "config.yml")
    if err := os.WriteFile(p, []byte(sample), 0600); err != nil {
        t.Fatal(err)
    }
    k := koanf.New(".")
    if err := k.Load(structs.Provider(DefaultConfig, "koanf"), nil); err != nil { t.Fatalf("load defaults: %v", err) }
    if err := k.Load(file.Provider(p), yaml.Parser()); err != nil { t.Fatalf("load YAML: %v", err) }
    var cfg Config
    if err := k.Unmarshal("", &cfg); err != nil { t.Fatalf("Unmarshal: %v; raw keys: %#v", err, k.All()) }

    got := cfg.Alerts.NotifyLimits.Camera
    want := map[string]string{"garage_innen":"2m", "garage_eingang":"90s", "garten":"1m"}
    if !reflect.DeepEqual(got,want) { t.Fatalf("loaded camera cooldown rules: got %#v; want %#v (raw: %v)",got,want,k.Get("alerts.notify_limits.camera")) }
    if cfg.App.Mode != "events" {t.Errorf("mode: %s",cfg.App.Mode)}
    if cfg.Alerts.General.NotifyOnce != true {t.Errorf("notify_once not populated")}
}
