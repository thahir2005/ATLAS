package intelligence

import (
	"testing"

	"atlas.local/control-plane/controller/internal/routing"
)

func TestAssessNetwork(t *testing.T) {
	tests := []struct {
		name     string
		snapshot routing.Snapshot
		expected NetworkStatus
	}{
		{
			name: "all peers and route healthy",
			snapshot: routing.Snapshot{
				Peers: map[string]routing.Peer{
					"172.20.12.2": {State: "Established"},
					"172.20.31.1": {State: "Established"},
				},
				BestPath:    &routing.BestPath{Peer: "172.20.12.2"},
				KernelRoute: &routing.KernelRoute{Target: "10.10.2.1"},
			},
			expected: NetworkHealthy,
		},
		{
			name: "one peer down",
			snapshot: routing.Snapshot{
				Peers: map[string]routing.Peer{
					"172.20.12.2": {State: "Established"},
					"172.20.31.1": {State: "Idle"},
				},
				BestPath:    &routing.BestPath{Peer: "172.20.12.2"},
				KernelRoute: &routing.KernelRoute{Target: "10.10.2.1"},
			},
			expected: NetworkDegraded,
		},
		{
			name: "best path missing",
			snapshot: routing.Snapshot{
				Peers: map[string]routing.Peer{
					"172.20.12.2": {State: "Established"},
				},
				KernelRoute: &routing.KernelRoute{Target: "10.10.2.1"},
			},
			expected: NetworkDegraded,
		},
		{
			name: "kernel route missing",
			snapshot: routing.Snapshot{
				Peers: map[string]routing.Peer{
					"172.20.12.2": {State: "Established"},
				},
				BestPath: &routing.BestPath{Peer: "172.20.12.2"},
			},
			expected: NetworkDegraded,
		},
		{
			name: "no peers and no route",
			snapshot: routing.Snapshot{
				Peers: map[string]routing.Peer{
					"172.20.12.2": {State: "Idle"},
					"172.20.31.1": {State: "Active"},
				},
			},
			expected: NetworkUnreachable,
		},
		{
			name:     "no peer telemetry",
			snapshot: routing.Snapshot{},
			expected: NetworkUnknown,
		},
		{
			name: "route observation error",
			snapshot: routing.Snapshot{
				Peers: map[string]routing.Peer{
					"172.20.12.2": {State: "Established"},
				},
				BestPath:         &routing.BestPath{Peer: "172.20.12.2"},
				KernelRouteError: "route lookup failed",
			},
			expected: NetworkDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AssessNetwork(tt.snapshot)

			if result.Status != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result.Status)
			}

			if result.Reason == "" {
				t.Error("expected an assessment reason")
			}
		})
	}
}
