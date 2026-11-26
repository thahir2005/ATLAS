package intelligence

import "testing"

func TestBuildRecoveryDecision(t *testing.T) {
	tests := []struct {
		name           string
		application    ApplicationStatus
		network        NetworkEvidence
		classification Classification
		action         RecoveryAction
		confidence     ConfidenceLevel
	}{
		{
			name:        "healthy region",
			application: ApplicationHealthy,
			network: NetworkEvidence{
				Status:           NetworkHealthy,
				EstablishedPeers: 2,
				TotalPeers:       2,
				HasBestPath:      true,
				HasKernelRoute:   true,
			},
			classification: Healthy,
			action:         ActionContinueMonitoring,
			confidence:     ConfidenceHigh,
		},
		{
			name:        "network degraded",
			application: ApplicationHealthy,
			network: NetworkEvidence{
				Status:           NetworkDegraded,
				EstablishedPeers: 1,
				TotalPeers:       2,
				HasBestPath:      true,
				HasKernelRoute:   true,
			},
			classification: NetworkDegradedResult,
			action:         ActionInvestigateNetwork,
			confidence:     ConfidenceMedium,
		},
		{
			name:        "application degraded",
			application: ApplicationUnhealthy,
			network: NetworkEvidence{
				Status:           NetworkHealthy,
				EstablishedPeers: 2,
				TotalPeers:       2,
				HasBestPath:      true,
				HasKernelRoute:   true,
			},
			classification: ApplicationDegraded,
			action:         ActionInvestigateApp,
			confidence:     ConfidenceHigh,
		},
		{
			name:        "application and network degraded",
			application: ApplicationUnhealthy,
			network: NetworkEvidence{
				Status:     NetworkUnreachable,
				TotalPeers: 2,
			},
			classification: RegionDegraded,
			action:         ActionInvestigateBoth,
			confidence:     ConfidenceMedium,
		},
		{
			name:        "insufficient telemetry",
			application: ApplicationUnknown,
			network: NetworkEvidence{
				Status: NetworkUnknown,
			},
			classification: Unknown,
			action:         ActionGatherTelemetry,
			confidence:     ConfidenceLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assessment := RegionAssessment{
				Region:            "mumbai",
				ApplicationStatus: tt.application,
				Network:           tt.network,
				Classification:    tt.classification,
			}

			decision, err := BuildRecoveryDecision(assessment)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if decision.Action != tt.action {
				t.Errorf("expected action %s, got %s", tt.action, decision.Action)
			}

			if decision.Confidence != tt.confidence {
				t.Errorf("expected confidence %s, got %s", tt.confidence, decision.Confidence)
			}

			if decision.AutomaticFailoverAllowed {
				t.Error("automatic failover must remain disabled")
			}

			if !decision.HumanApprovalRequiredBeforeFailover {
				t.Error("human approval must be required before failover")
			}

			if decision.Mode != DecisionModeAdvisory {
				t.Errorf("expected advisory mode, got %s", decision.Mode)
			}
		})
	}
}

func TestBuildRecoveryDecisionRejectsClassificationMismatch(t *testing.T) {
	assessment := RegionAssessment{
		Region:            "mumbai",
		ApplicationStatus: ApplicationHealthy,
		Network: NetworkEvidence{
			Status:     NetworkHealthy,
			TotalPeers: 2,
		},
		Classification: RegionDegraded,
	}

	_, err := BuildRecoveryDecision(assessment)
	if err == nil {
		t.Fatal("expected classification mismatch error")
	}
}

func TestBuildRecoveryDecisionRejectsInvalidEvidence(t *testing.T) {
	assessment := RegionAssessment{
		Region:            "",
		ApplicationStatus: ApplicationHealthy,
		Network: NetworkEvidence{
			Status: NetworkHealthy,
		},
		Classification: Healthy,
	}

	_, err := BuildRecoveryDecision(assessment)
	if err == nil {
		t.Fatal("expected invalid evidence error")
	}
}
