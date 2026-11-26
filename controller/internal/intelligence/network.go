package intelligence

import "atlas.local/control-plane/controller/internal/routing"

type NetworkEvidence struct {
	Status           NetworkStatus `json:"status"`
	EstablishedPeers int           `json:"established_peers"`
	TotalPeers       int           `json:"total_peers"`
	HasBestPath      bool          `json:"has_best_path"`
	HasKernelRoute   bool          `json:"has_kernel_route"`
	Reason           string        `json:"reason"`
}

// AssessNetwork evaluates the observed BGP path and its peer sessions.
// It describes the path from the observer to the requested destination,
// not the global health of the destination region.
func AssessNetwork(snapshot routing.Snapshot) NetworkEvidence {
	evidence := NetworkEvidence{
		TotalPeers:     len(snapshot.Peers),
		HasBestPath:    snapshot.BestPath != nil,
		HasKernelRoute: snapshot.KernelRoute != nil,
	}

	if len(snapshot.Peers) == 0 {
		evidence.Status = NetworkUnknown
		evidence.Reason = "No BGP peer telemetry is available"
		return evidence
	}

	for _, peer := range snapshot.Peers {
		if peer.State == "Established" {
			evidence.EstablishedPeers++
		}
	}

	if evidence.EstablishedPeers == 0 && !evidence.HasKernelRoute {
		evidence.Status = NetworkUnreachable
		evidence.Reason = "No BGP peers are established and no kernel route exists"
		return evidence
	}

	if evidence.EstablishedPeers != evidence.TotalPeers {
		evidence.Status = NetworkDegraded
		evidence.Reason = "One or more BGP peer sessions are not established"
		return evidence
	}

	if !evidence.HasBestPath {
		evidence.Status = NetworkDegraded
		evidence.Reason = "No BGP best path is available"
		return evidence
	}

	if !evidence.HasKernelRoute {
		evidence.Status = NetworkDegraded
		evidence.Reason = "No kernel route is available"
		return evidence
	}

	if snapshot.KernelRouteError != "" {
		evidence.Status = NetworkDegraded
		evidence.Reason = "Kernel route observation reported an error"
		return evidence
	}

	evidence.Status = NetworkHealthy
	evidence.Reason = "All observed BGP peers are established and the best path is installed"
	return evidence
}
