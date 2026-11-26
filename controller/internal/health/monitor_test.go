package health

import "testing"

func TestInitialState(t *testing.T) {
	tracker := NewTracker()

	events := tracker.Observe([]Result{
		{
			Region:    "Mumbai",
			Healthy:   true,
			CheckedAt: "2026-11-26T10:00:00Z",
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected one initialization event, got %d", len(events))
	}

	if events[0].Event != "REGION_STATE_INITIALIZED" {
		t.Fatalf("unexpected event: %s", events[0].Event)
	}

	if events[0].CurrentStatus != "healthy" {
		t.Fatalf("expected healthy, got %s", events[0].CurrentStatus)
	}
}

func TestDetectUnhealthyTransition(t *testing.T) {
	tracker := NewTracker()

	tracker.Observe([]Result{
		{Region: "Singapore", Healthy: true},
	})

	events := tracker.Observe([]Result{
		{
			Region:  "Singapore",
			Healthy: false,
			Error:   "connection refused",
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected one transition, got %d", len(events))
	}

	if events[0].PreviousStatus != "healthy" {
		t.Fatalf("expected previous status healthy, got %s", events[0].PreviousStatus)
	}

	if events[0].CurrentStatus != "unhealthy" {
		t.Fatalf("expected current status unhealthy, got %s", events[0].CurrentStatus)
	}
}

func TestDetectRecovery(t *testing.T) {
	tracker := NewTracker()

	tracker.Observe([]Result{
		{Region: "Frankfurt", Healthy: false},
	})

	events := tracker.Observe([]Result{
		{Region: "Frankfurt", Healthy: true},
	})

	if len(events) != 1 {
		t.Fatalf("expected one recovery event, got %d", len(events))
	}

	if events[0].PreviousStatus != "unhealthy" ||
		events[0].CurrentStatus != "healthy" {
		t.Fatalf("unexpected recovery transition: %+v", events[0])
	}
}

func TestNoEventWhenStateUnchanged(t *testing.T) {
	tracker := NewTracker()

	tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: true},
	})

	events := tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: true},
	})

	if len(events) != 0 {
		t.Fatalf("expected no transition, got %d", len(events))
	}
}
