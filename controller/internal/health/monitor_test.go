package health

import "testing"

func newTestTracker(t *testing.T) *Tracker {
	t.Helper()

	tracker, err := NewTrackerWithThresholds(3, 2)
	if err != nil {
		t.Fatal(err)
	}

	return tracker
}

func TestInitialState(t *testing.T) {
	tracker := newTestTracker(t)

	events := tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: true},
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

func TestFailureRequiresThreeConsecutiveChecks(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.Observe([]Result{
		{Region: "Singapore", Healthy: true},
	})

	first := tracker.Observe([]Result{
		{Region: "Singapore", Healthy: false, Error: "connection refused"},
	})

	if len(first) != 1 || first[0].Event != "REGION_FAILURE_SUSPECTED" {
		t.Fatalf("expected suspected failure, got %+v", first)
	}
	if tracker.Status("Singapore") != "healthy" {
		t.Fatal("one failure must not change the confirmed region state")
	}

	second := tracker.Observe([]Result{
		{Region: "Singapore", Healthy: false},
	})

	if len(second) != 0 {
		t.Fatalf("expected no state change after two failures, got %+v", second)
	}
	if tracker.Status("Singapore") != "healthy" {
		t.Fatal("two failures must not change the confirmed region state")
	}

	third := tracker.Observe([]Result{
		{Region: "Singapore", Healthy: false},
	})

	if len(third) != 1 {
		t.Fatalf("expected one state change after three failures, got %+v", third)
	}

	if third[0].CurrentStatus != "unhealthy" {
		t.Fatalf("expected unhealthy, got %s", third[0].CurrentStatus)
	}
	if tracker.Status("Singapore") != "unhealthy" {
		t.Fatal("three failures must change the confirmed region state")
	}

	if third[0].ConsecutiveFailures != 3 {
		t.Fatalf("expected three consecutive failures, got %d", third[0].ConsecutiveFailures)
	}
}

func TestRecoveryRequiresTwoConsecutiveChecks(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.Observe([]Result{
		{Region: "Frankfurt", Healthy: false},
	})

	first := tracker.Observe([]Result{
		{Region: "Frankfurt", Healthy: true},
	})

	if len(first) != 1 || first[0].Event != "REGION_RECOVERY_SUSPECTED" {
		t.Fatalf("expected suspected recovery, got %+v", first)
	}

	second := tracker.Observe([]Result{
		{Region: "Frankfurt", Healthy: true},
	})

	if len(second) != 1 {
		t.Fatalf("expected recovery transition, got %+v", second)
	}

	if second[0].PreviousStatus != "unhealthy" ||
		second[0].CurrentStatus != "healthy" {
		t.Fatalf("unexpected recovery transition: %+v", second[0])
	}
	if tracker.Status("Frankfurt") != "healthy" {
		t.Fatal("two successes must restore the confirmed region state")
	}

	if second[0].ConsecutiveSuccesses != 2 {
		t.Fatalf("expected two consecutive successes, got %d", second[0].ConsecutiveSuccesses)
	}
}

func TestHealthyCheckResetsFailureCounter(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: true},
	})

	tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: false},
	})

	tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: false},
	})

	tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: true},
	})

	first := tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: false},
	})

	if len(first) != 1 || first[0].Event != "REGION_FAILURE_SUSPECTED" {
		t.Fatalf("expected suspected failure after counter reset, got %+v", first)
	}

	second := tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: false},
	})

	if len(second) != 0 {
		t.Fatalf("expected no state change after two failures, got %+v", second)
	}
}

func TestUnhealthyCheckResetsRecoveryCounter(t *testing.T) {
	tracker := newTestTracker(t)

	tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: false},
	})

	tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: true},
	})

	tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: false},
	})

	events := tracker.Observe([]Result{
		{Region: "Mumbai", Healthy: true},
	})

	if len(events) != 1 || events[0].Event != "REGION_RECOVERY_SUSPECTED" {
		t.Fatalf("recovery counter should have reset, got %+v", events)
	}
}

func TestInvalidThresholds(t *testing.T) {
	_, err := NewTrackerWithThresholds(0, 2)

	if err == nil {
		t.Fatal("expected error for zero failure threshold")
	}

	_, err = NewTrackerWithThresholds(3, -1)

	if err == nil {
		t.Fatal("expected error for negative recovery threshold")
	}
}
