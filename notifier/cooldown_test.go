package notifier

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0x2142/frigate-notify/models"
)

func TestCooldownLabelSharedAcrossCameras(t *testing.T) {
	limits := models.NotifyLimits{Label: map[string]string{"car": "5m"}}
	var limiter cooldownTracker
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	first := []models.Event{{Camera: "door", Label: "car"}}
	second := []models.Event{{Camera: "driveway", Label: "car"}}
	if ok, _ := limiter.allowAt(first, limits, start); !ok {
		t.Fatal("initial car event should be allowed")
	}
	if ok, _ := limiter.allowAt(second, limits, start.Add(4*time.Minute)); ok {
		t.Fatal("car on another camera must be suppressed")
	}
	if ok, _ := limiter.allowAt(second, limits, start.Add(5*time.Minute)); !ok {
		t.Fatal("car should be allowed at exact cooldown boundary")
	}
}

func TestCooldownCameraAndZone(t *testing.T) {
	limits := models.NotifyLimits{
		Camera: map[string]string{"door": "1m"},
		Zone: map[string]string{"front_door": "2m"},
	}
	var limiter cooldownTracker
	start := time.Now()
	cameraZone := []models.Event{{Camera: "door", Label: "person", CurrentZones: []string{"front_door"}}}
	otherCameraSameZone := []models.Event{{Camera: "side", Label: "person", CurrentZones: []string{"front_door"}}}
	otherCameraOtherZone := []models.Event{{Camera: "side", Label: "person", CurrentZones: []string{"garden"}}}
	if ok, _ := limiter.allowAt(cameraZone, limits, start); !ok {
		t.Fatal("first notification must be allowed")
	}
	if ok, _ := limiter.allowAt(otherCameraSameZone, limits, start.Add(time.Minute)); ok {
		t.Fatal("zone cooldown must apply across cameras")
	}
	if ok, _ := limiter.allowAt(otherCameraOtherZone, limits, start.Add(time.Minute)); !ok {
		t.Fatal("unrelated camera and zone should be allowed")
	}
	if ok, _ := limiter.allowAt(otherCameraSameZone, limits, start.Add(2*time.Minute)); !ok {
		t.Fatal("zone should be allowed when two-minute cooldown expires")
	}
}

func TestCooldownOverlappingRulesAndRejectedNotExtended(t *testing.T) {
	limits := models.NotifyLimits{
		Camera: map[string]string{"door": "1m"},
		Label: map[string]string{"car": "5m"},
		Zone: map[string]string{"front_door": "2m"},
	}
	var limiter cooldownTracker
	start := time.Now()
	first := []models.Event{{Camera: "door", Label: "car", CurrentZones: []string{"front_door"}}}
	otherCar := []models.Event{{Camera: "garden", Label: "car"}}
	person := []models.Event{{Camera: "garden", Label: "person", CurrentZones: []string{"front_door"}}}
	if ok, _ := limiter.allowAt(first, limits, start); !ok {
		t.Fatal("first notification should pass all three rules")
	}
	if ok, reason := limiter.allowAt(otherCar, limits, start.Add(time.Minute)); ok || reason == "" {
		t.Fatal("label cooldown should deny second car with explanation")
	}
	if ok, _ := limiter.allowAt(person, limits, start.Add(time.Minute)); ok {
		t.Fatal("zone cooldown should block person before two minutes")
	}
	if ok, _ := limiter.allowAt(person, limits, start.Add(2*time.Minute)); !ok {
		t.Fatal("suppressed person must not extend front_door cooldown")
	}
	if ok, _ := limiter.allowAt(otherCar, limits, start.Add(5*time.Minute)); !ok {
		t.Fatal("suppressed cars must not extend label cooldown")
	}
}

func TestGlobalCooldown(t *testing.T) {
	limits := models.NotifyLimits{Global: "15s"}
	var limiter cooldownTracker
	start := time.Now()
	if ok, _ := limiter.allowAt([]models.Event{{Label: "car"}}, limits, start); !ok {
		t.Fatal("first notification should be allowed")
	}
	if ok, _ := limiter.allowAt([]models.Event{{Label: "person"}}, limits, start.Add(14*time.Second)); ok {
		t.Fatal("global rule applies to unrelated labels")
	}
	if ok, _ := limiter.allowAt([]models.Event{{Label: "person"}}, limits, start.Add(15*time.Second)); !ok {
		t.Fatal("global rule should expire after 15 seconds")
	}
}

func TestCooldownConcurrentEvents(t *testing.T) {
	limits := models.NotifyLimits{Label: map[string]string{"car": "1m"}}
	var limiter cooldownTracker
	var accepted atomic.Int32
	var wg sync.WaitGroup
	gate := make(chan struct{})
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			if ok, _ := limiter.allow([]models.Event{{Label: "car"}}, limits); ok {
				accepted.Add(1)
			}
		}()
	}
	close(gate)
	wg.Wait()
	if got := accepted.Load(); got != 1 {
		t.Fatalf("got %d accepted notifications, want exactly one", got)
	}
}

func TestCooldownMultipleDetectionsDeduplicated(t *testing.T) {
	limits := models.NotifyLimits{Label: map[string]string{"car": "5m"}}
	rules := matchingCooldownRules([]models.Event{{Label: "car"}, {Label: "car"}}, limits)
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want one unique label rule", len(rules))
	}
}

func TestCooldownDisabled(t *testing.T) {
	var limiter cooldownTracker
	for i := 0; i < 2; i++ {
		if ok, _ := limiter.allow([]models.Event{{Label: "car"}}, models.NotifyLimits{}); !ok {
			t.Fatal("omitted rules must allow every notification")
		}
	}
}
