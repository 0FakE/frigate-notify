package events

import (
	"testing"

	"github.com/0x2142/frigate-notify/config"
	"github.com/0x2142/frigate-notify/models"
)

func TestBlockedZoneCurrentlyOccupied(t *testing.T) {
	prev := config.ConfigData
	defer func(){ config.ConfigData = prev }()
	config.ConfigData.Alerts.Zones.Block = []string{"Gehweg", "Fremdgrund"}
	if isAllowedZoneWithHistory("e", []string{"Innen", "Gehweg"}, nil) {
		t.Fatal("a current blocked zone must override an allowed current zone")
	}
}

func TestBlockedZonePreviouslyEntered(t *testing.T) {
	prev := config.ConfigData
	defer func(){ config.ConfigData = prev }()
	config.ConfigData.Alerts.Zones.Block = []string{"Gehweg", "Fremdgrund"}
	config.ConfigData.Alerts.Zones.Allow = []string{"Innen"}
	if isAllowedZoneWithHistory("e", []string{"Innen"}, []string{"Gehweg"}) {
		t.Fatal("historical blocked zone must prevent an alert in a later allowed zone")
	}
	if isAllowedZoneWithHistory("e", nil, []string{"Fremdgrund"}) {
		t.Fatal("a previously entered blocked zone must also block an unzoned update")
	}
	if !isAllowedZoneWithHistory("e", []string{"Innen"}, []string{"Einfahrt"}) {
		t.Fatal("unblocked historical zones should not prevent an allowed current zone")
	}
}

func TestUnzonedOptionAndAllowListKeepOriginalMeaning(t *testing.T) {
	prev := config.ConfigData
	defer func(){ config.ConfigData = prev }()
	config.ConfigData.Alerts.Zones.Allow = []string{"Innen"}
	config.ConfigData.Alerts.Zones.Unzoned = "allow"
	if !isAllowedZoneWithHistory("e", nil, []string{"Innen"}) {
		t.Fatal("original unzoned: allow behavior must be kept")
	}
	config.ConfigData.Alerts.Zones.Unzoned = "drop"
	if isAllowedZoneWithHistory("e", nil, []string{"Innen"}) {
		t.Fatal("entered zone is history, not currently occupied; unzoned: drop must apply")
	}
	if isAllowedZoneWithHistory("e", []string{"Einfahrt"}, []string{"Innen"}) {
		t.Fatal("historical allowed zone must not override a nonallowed current zone")
	}
	if !isAllowedZoneWithHistory("e", []string{"Innen"}, nil) {
		t.Fatal("current allowed zone should be accepted")
	}
}

func TestEventFiltersRejectHistoricalBlockedZoneBeforeCooldown(t *testing.T) {
	prev := config.ConfigData
	prevEnabled := config.Internal.Status.Notifications.Enabled
	defer func() {
		config.ConfigData = prev
		config.Internal.Status.Notifications.Enabled = prevEnabled
	}()
	InitZoneCache()
	defer CloseZoneCache()
	config.Internal.Status.Notifications.Enabled = true
	config.ConfigData = config.Config{}
	config.ConfigData.Alerts.Zones.Unzoned = "drop"
	config.ConfigData.Alerts.Zones.Block = []string{"Gehweg"}
	config.ConfigData.Alerts.Labels.Allow = []string{"Person"}
	event := models.Event{
		ID:"historical-zone", Camera:"Garage_Eingang", Label:"Person",
		CurrentZones: []string{"Innenhof"}, EnteredZones: []string{"Gehweg", "Innenhof"},
		HasSnapshot:true, TopScore: 0.9,
	}
	if checkEventFilters(event) {
		t.Fatal("historical blocked zone must prevent notification before cooldown evaluation")
	}
	event.EnteredZones = []string{"Innenhof"}
	if !checkEventFilters(event) {
		t.Fatal("removing the blocked history must allow the event")
	}
}
