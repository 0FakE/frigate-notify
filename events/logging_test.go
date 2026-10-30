package events

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/0x2142/frigate-notify/config"
	"github.com/0x2142/frigate-notify/models"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// mockEventMessage tests the MQTT event receive path without a broker.
type mockEventMessage struct {
	payload []byte
}
func (m mockEventMessage) Duplicate() bool { return false }
func (m mockEventMessage) Qos() byte { return 0 }
func (m mockEventMessage) Retained() bool { return false }
func (m mockEventMessage) Topic() string { return "frigate/events" }
func (m mockEventMessage) MessageID() uint16 { return 0 }
func (m mockEventMessage) Payload() []byte { return m.payload }
func (m mockEventMessage) Ack() {}

func testEventLogger(t *testing.T, level zerolog.Level) *bytes.Buffer {
	t.Helper()
	var output bytes.Buffer
	previousLogger := log.Logger
	previousLevel := zerolog.GlobalLevel()
	previousConfig := config.ConfigData
	previousEnabled := config.Internal.Status.Notifications.Enabled
	t.Cleanup(func() {
		log.Logger = previousLogger
		zerolog.SetGlobalLevel(previousLevel)
		config.ConfigData = previousConfig
		config.Internal.Status.Notifications.Enabled = previousEnabled
	})
	zerolog.SetGlobalLevel(level)
	log.Logger = zerolog.New(&output)
	config.ConfigData = config.Config{}
	config.Internal.Status.Notifications.Enabled = true
	return &output
}

func TestNormalEventReceiptAndFilteringAreDebugOnly(t *testing.T) {
	output := testEventLogger(t, zerolog.InfoLevel)
	for _, eventType := range []string{"new", "update"} {
		payload, err := json.Marshal(map[string]any{
			"type": eventType,
			"after": map[string]any{
				"id": "event-no-snapshot",
				"camera": "Garage_Innen",
				"label": "Person",
				"has_snapshot": false,
				"has_clip": false,
			},
		})
		if err != nil { t.Fatal(err) }
		handleMQTTMsg(nil, mockEventMessage{payload: payload})
	}
	for _, message := range []string{"New event received","Event update received","Processing event...","Event dropped - No snapshot or clip available"} {
		if strings.Contains(output.String(), message) {
			t.Fatalf("INFO log unexpectedly contains %q: %s", message, output.String())
		}
	}
	if output.Len() != 0 { t.Fatalf("expected no INFO logs for filtered events: %s", output.String()) }
}

func TestDebugLogShowsEventReceiptProcessingAndFiltering(t *testing.T) {
	output := testEventLogger(t, zerolog.DebugLevel)
	payload := []byte(`{"type":"update","after":{"id":"event-no-snapshot","camera":"Garage_Innen","label":"Person","has_snapshot":false,"has_clip":false}}`)
	handleMQTTMsg(nil, mockEventMessage{payload:payload})
	for _, message := range []string{"Event update received","Processing event...","Event dropped - No snapshot or clip available"} {
		if !strings.Contains(output.String(), message) {
			t.Fatalf("DEBUG log missing %q: %s", message, output.String())
		}
	}
}

func TestAlreadyNotifiedEventRemainsInfo(t *testing.T) {
	output := testEventLogger(t, zerolog.InfoLevel)
	InitZoneCache()
	defer CloseZoneCache()
	config.ConfigData.Alerts.General.NotifyOnce = true
	event := models.Event{
		ID: "previously-notified", Camera: "Garage_Innen", Label: "Person",
		CurrentZones: []string{"Innen"}, HasClip:true, HasSnapshot:true,
	}
	setZoneAlerted(event)
	if checkEventFilters(event) { t.Fatal("notify_once should prevent a repeated event") }
	if !strings.Contains(output.String(), "Event dropped - Already notified & notify_once is set") {
		t.Fatalf("notify_once decision must stay INFO: %s", output.String())
	}
}

func TestAlreadyNotifiedZoneRemainsInfo(t *testing.T) {
	output := testEventLogger(t, zerolog.InfoLevel)
	InitZoneCache()
	defer CloseZoneCache()
	event := models.Event{
		ID:"previously-notified-zone", Camera:"Garage_Innen", Label:"Person",
		CurrentZones:[]string{"Innen"}, HasClip:true, HasSnapshot:true,
	}
	setZoneAlerted(event)
	if checkEventFilters(event) { t.Fatal("already notified zone should block the same event") }
	if !strings.Contains(output.String(), "Event dropped - Already notified on this zone") {
		t.Fatalf("already notified zone decision must stay INFO: %s", output.String())
	}
}
