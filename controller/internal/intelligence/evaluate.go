package intelligence

import "atlas.local/control-plane/controller/internal/routing"

type RegionAssessment struct {
	Region            string            `json:"region"`
	ApplicationStatus ApplicationStatus `json:"application_status"`
	Network           NetworkEvidence   `json:"network"`
	Classification    Classification    `json:"classification"`
	Reason            string            `json:"reason"`
	Recommendation    string            `json:"recommendation"`
}

func EvaluateRegion(
	region string,
	application ApplicationStatus,
	snapshot routing.Snapshot,
) (RegionAssessment, error) {
	network := AssessNetwork(snapshot)

	classification, err := Classify(Observation{
		Region:            region,
		ApplicationStatus: application,
		NetworkStatus:     network.Status,
	})
	if err != nil {
		return RegionAssessment{}, err
	}

	return RegionAssessment{
		Region:            classification.Region,
		ApplicationStatus: application,
		Network:           network,
		Classification:    classification.Classification,
		Reason:            classification.Reason,
		Recommendation:    recommendationFor(classification.Classification),
	}, nil
}

func recommendationFor(classification Classification) string {
	switch classification {
	case Healthy:
		return "No recovery action required; continue monitoring"
	case NetworkDegradedResult:
		return "Inspect BGP peer sessions and route convergence; verify alternate paths"
	case ApplicationDegraded:
		return "Investigate the regional application, process health, and service logs"
	case RegionDegraded:
		return "Investigate both application and network failures; evaluate a safe recovery target"
	default:
		return "Collect more application and network telemetry before taking action"
	}
}
