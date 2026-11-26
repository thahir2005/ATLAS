package intelligence

import "testing"

func healthyAssessment() RegionAssessment {
	return RegionAssessment{
		Region:            "Mumbai",
		ApplicationStatus: ApplicationHealthy,
		Network: NetworkEvidence{
			Status:           NetworkHealthy,
			TotalPeers:       2,
			EstablishedPeers: 2,
			HasBestPath:      true,
			HasKernelRoute:   true,
		},
		Classification: Healthy,
	}
}

func TestDatabaseAwareRecoveryDecisions(t *testing.T) {
	tests := []struct {
		name           string
		database       DatabaseAssessment
		wantAction     RecoveryAction
		wantConfidence ConfidenceLevel
	}{
		{
			name:           "healthy database",
			database:       DatabaseAssessment{Status: DatabaseHealthy},
			wantAction:     ActionContinueMonitoring,
			wantConfidence: ConfidenceHigh,
		},
		{
			name: "replication degraded",
			database: DatabaseAssessment{
				Status: DatabaseReplicationDegraded,
				Reason: "replica disconnected",
			},
			wantAction:     ActionInvestigateDatabase,
			wantConfidence: ConfidenceMedium,
		},
		{
			name: "database unavailable",
			database: DatabaseAssessment{
				Status: DatabaseUnavailable,
				Reason: "connection refused",
			},
			wantAction:     ActionInvestigateDatabase,
			wantConfidence: ConfidenceLow,
		},
		{
			name: "database telemetry unknown",
			database: DatabaseAssessment{
				Status: DatabaseUnknown,
				Reason: "DSN not configured",
			},
			wantAction:     ActionGatherTelemetry,
			wantConfidence: ConfidenceLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := BuildRecoveryDecisionWithDatabase(
				healthyAssessment(), tt.database,
			)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Action != tt.wantAction {
				t.Fatalf("expected action %s, got %s",
					tt.wantAction, decision.Action)
			}
			if decision.Confidence != tt.wantConfidence {
				t.Fatalf("expected confidence %s, got %s",
					tt.wantConfidence, decision.Confidence)
			}
			if decision.AutomaticFailoverAllowed {
				t.Fatal("automatic failover must remain disabled")
			}
			if !decision.HumanApprovalRequiredBeforeFailover {
				t.Fatal("human approval must remain required")
			}
		})
	}
}

func TestDatabaseDegradationCombinesWithNetworkFailure(t *testing.T) {
	assessment := healthyAssessment()
	assessment.Network.Status = NetworkDegraded
	assessment.Classification = NetworkDegradedResult

	decision, err := BuildRecoveryDecisionWithDatabase(
		assessment,
		DatabaseAssessment{Status: DatabaseReplicationDegraded},
	)
	if err != nil {
		t.Fatal(err)
	}

	if decision.Action != ActionInvestigateDatabaseAndNetwork {
		t.Fatalf("unexpected action: %s", decision.Action)
	}
}
