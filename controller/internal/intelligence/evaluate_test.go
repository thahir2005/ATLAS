package intelligence

import (
	"testing"

	"atlas.local/control-plane/controller/internal/routing"
)

func healthySnapshot() routing.Snapshot {
	return routing.Snapshot{
		Namespace: "atlas-mumbai",
		Peers: map[string]routing.Peer{
			"172.20.12.2": {State: "Established"},
			"172.20.31.1": {State: "Established"},
		},
		BestPath: &routing.BestPath{
			Peer: "172.20.12.2",
		},
		KernelRoute: &routing.KernelRoute{
			Target:    "10.10.2.1",
			NextHop:   "172.20.12.2",
			Interface: "ms0",
			Source:    "172.20.12.1",
		},
	}
}

func TestEvaluateRegion(t *testing.T) {
	tests := []struct {
		name           string
		application    ApplicationStatus
		snapshot       routing.Snapshot
		expected       Classification
		expectedAction string
	}{
		{
			name:           "healthy region",
			application:    ApplicationHealthy,
			snapshot:       healthySnapshot(),
			expected:       Healthy,
			expectedAction: "No recovery action required; continue monitoring",
		},
		{
			name:        "network degraded",
			application: ApplicationHealthy,
			snapshot: routing.Snapshot{
				Peers: map[string]routing.Peer{
					"172.20.12.2": {State: "Established"},
					"172.20.31.1": {State: "Idle"},
				},
				BestPath:    &routing.BestPath{Peer: "172.20.12.2"},
				KernelRoute: &routing.KernelRoute{Target: "10.10.2.1"},
			},
			expected:       NetworkDegradedResult,
			expectedAction: "Inspect BGP peer sessions and route convergence; verify alternate paths",
		},
		{
			name:           "application degraded",
			application:    ApplicationUnhealthy,
			snapshot:       healthySnapshot(),
			expected:       ApplicationDegraded,
			expectedAction: "Investigate the regional application, process health, and service logs",
		},
		{
			name:        "both layers degraded",
			application: ApplicationUnhealthy,
			snapshot: routing.Snapshot{
				Peers: map[string]routing.Peer{
					"172.20.12.2": {State: "Idle"},
				},
			},
			expected:       RegionDegraded,
			expectedAction: "Investigate both application and network failures; evaluate a safe recovery target",
		},
		{
			name:           "insufficient telemetry",
			application:    ApplicationUnknown,
			snapshot:       routing.Snapshot{},
			expected:       Unknown,
			expectedAction: "Collect more application and network telemetry before taking action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateRegion(
				"mumbai",
				tt.application,
				tt.snapshot,
			)
			if err != nil {
				t.Fatalf("EvaluateRegion returned error: %v", err)
			}

			if result.Classification != tt.expected {
				t.Errorf("expected %s, got %s",
					tt.expected, result.Classification)
			}

			if result.Recommendation != tt.expectedAction {
				t.Errorf("unexpected recommendation: %s", result.Recommendation)
			}

			if result.Region != "mumbai" {
				t.Errorf("expected mumbai, got %s", result.Region)
			}

			if result.Reason == "" || result.Network.Reason == "" {
				t.Error("expected classification and network evidence")
			}
		})
	}
}

func TestEvaluateRegionRejectsInvalidApplicationStatus(t *testing.T) {
	_, err := EvaluateRegion(
		"mumbai",
		ApplicationStatus("BROKEN"),
		healthySnapshot(),
	)

	if err == nil {
		t.Fatal("expected invalid application status error")
	}
}
