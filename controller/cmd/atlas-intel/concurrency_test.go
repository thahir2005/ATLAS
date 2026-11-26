package main

import (
	"testing"
	"time"

	"atlas.local/control-plane/controller/internal/health"
)

func TestRoutingStartsWhileApplicationCheckIsBlocked(t *testing.T) {
	probes := []regionProbe{
		{Name: "Mumbai"},
		{Name: "Singapore"},
		{Name: "Frankfurt"},
	}

	releaseApplication := make(chan struct{})
	routingStarted := make(chan struct{}, len(probes))
	completed := make(chan observationBatch, 1)

	go func() {
		batch := collectConcurrently(
			func() []health.Result {
				<-releaseApplication
				return []health.Result{{Region: "Mumbai", Healthy: true}}
			},
			probes,
			func(probe regionProbe) routingResult {
				routingStarted <- struct{}{}
				return routingResult{Probe: probe}
			},
			nil,
			nil,
		)
		completed <- batch
	}()

	// Every routing observation must start while the app check is blocked.
	for range probes {
		select {
		case <-routingStarted:
		case <-time.After(2 * time.Second):
			close(releaseApplication)
			t.Fatal("routing observations did not start independently")
		}
	}

	// The batch must still wait for the application result.
	select {
	case <-completed:
		close(releaseApplication)
		t.Fatal("batch completed before the application check was released")
	default:
	}

	close(releaseApplication)

	select {
	case batch := <-completed:
		if len(batch.Applications) != 1 {
			t.Fatalf("expected 1 application result, got %d", len(batch.Applications))
		}
		if len(batch.Routing) != len(probes) {
			t.Fatalf("expected %d routing results, got %d", len(probes), len(batch.Routing))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent observation batch did not complete")
	}
}

func TestDatabaseStartsWhileApplicationCheckIsBlocked(t *testing.T) {
	releaseApplication := make(chan struct{})
	databaseStarted := make(chan struct{}, 1)
	completed := make(chan observationBatch, 1)

	go func() {
		batch := collectConcurrently(
			func() []health.Result {
				<-releaseApplication
				return nil
			},
			nil,
			nil,
			[]databaseProbe{{Region: "Mumbai"}},
			func(probe databaseProbe) databaseResult {
				databaseStarted <- struct{}{}
				return databaseResult{Probe: probe}
			},
		)
		completed <- batch
	}()

	select {
	case <-databaseStarted:
	case <-time.After(2 * time.Second):
		close(releaseApplication)
		t.Fatal("database observation did not start independently")
	}

	select {
	case <-completed:
		close(releaseApplication)
		t.Fatal("batch completed before application check was released")
	default:
	}

	close(releaseApplication)

	select {
	case batch := <-completed:
		if len(batch.Databases) != 1 {
			t.Fatalf("expected 1 database result, got %d", len(batch.Databases))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent observation batch did not complete")
	}
}
