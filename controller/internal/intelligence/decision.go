package intelligence

import "fmt"

type RecoveryAction string
type ConfidenceLevel string

const (
	ActionContinueMonitoring            RecoveryAction = "CONTINUE_MONITORING"
	ActionInvestigateNetwork            RecoveryAction = "INVESTIGATE_NETWORK"
	ActionInvestigateApp                RecoveryAction = "INVESTIGATE_APPLICATION"
	ActionInvestigateBoth               RecoveryAction = "INVESTIGATE_APPLICATION_AND_NETWORK"
	ActionGatherTelemetry               RecoveryAction = "GATHER_TELEMETRY"
	ActionInvestigateDatabase           RecoveryAction = "INVESTIGATE_DATABASE"
	ActionInvestigateDatabaseAndNetwork RecoveryAction = "INVESTIGATE_DATABASE_AND_NETWORK"
	ActionInvestigateAppAndDatabase     RecoveryAction = "INVESTIGATE_APPLICATION_AND_DATABASE"
	ActionInvestigateAllLayers          RecoveryAction = "INVESTIGATE_ALL_LAYERS"

	ConfidenceHigh   ConfidenceLevel = "HIGH"
	ConfidenceMedium ConfidenceLevel = "MEDIUM"
	ConfidenceLow    ConfidenceLevel = "LOW"

	DecisionModeAdvisory = "ADVISORY_ONLY"
)

type RecoveryDecision struct {
	Region                              string          `json:"region"`
	Classification                      Classification  `json:"classification"`
	DatabaseStatus                      DatabaseStatus  `json:"database_status,omitempty"`
	Action                              RecoveryAction  `json:"action"`
	Confidence                          ConfidenceLevel `json:"confidence"`
	Reason                              string          `json:"reason"`
	Mode                                string          `json:"mode"`
	AutomaticFailoverAllowed            bool            `json:"automatic_failover_allowed"`
	HumanApprovalRequiredBeforeFailover bool            `json:"human_approval_required_before_failover"`
}

// BuildRecoveryDecision produces an advisory decision from observed evidence.
// It never performs or authorizes automatic failover.
func BuildRecoveryDecision(assessment RegionAssessment) (RecoveryDecision, error) {
	classification, err := Classify(Observation{
		Region:            assessment.Region,
		ApplicationStatus: assessment.ApplicationStatus,
		NetworkStatus:     assessment.Network.Status,
	})
	if err != nil {
		return RecoveryDecision{}, err
	}

	if classification.Classification != assessment.Classification {
		return RecoveryDecision{}, fmt.Errorf(
			"assessment classification mismatch: expected %s, received %s",
			classification.Classification,
			assessment.Classification,
		)
	}

	decision := RecoveryDecision{
		Region:                              classification.Region,
		Classification:                      classification.Classification,
		Mode:                                DecisionModeAdvisory,
		AutomaticFailoverAllowed:            false,
		HumanApprovalRequiredBeforeFailover: true,
		Reason:                              classification.Reason,
	}

	switch classification.Classification {
	case Healthy:
		decision.Action = ActionContinueMonitoring
	case NetworkDegradedResult:
		decision.Action = ActionInvestigateNetwork
	case ApplicationDegraded:
		decision.Action = ActionInvestigateApp
	case RegionDegraded:
		decision.Action = ActionInvestigateBoth
	default:
		decision.Action = ActionGatherTelemetry
	}

	decision.Confidence = decisionConfidence(assessment)

	return decision, nil
}

func decisionConfidence(assessment RegionAssessment) ConfidenceLevel {
	if assessment.ApplicationStatus == ApplicationUnknown ||
		assessment.Network.Status == NetworkUnknown ||
		assessment.Network.TotalPeers == 0 {
		return ConfidenceLow
	}

	if assessment.Network.Status == NetworkDegraded ||
		assessment.Network.Status == NetworkUnreachable {
		return ConfidenceMedium
	}

	return ConfidenceHigh
}

// BuildRecoveryDecisionWithDatabase extends the existing advisory decision
// with database evidence. It never authorizes or performs failover.
func BuildRecoveryDecisionWithDatabase(
	assessment RegionAssessment,
	database DatabaseAssessment,
) (RecoveryDecision, error) {
	decision, err := BuildRecoveryDecision(assessment)
	if err != nil {
		return RecoveryDecision{}, err
	}

	decision.DatabaseStatus = database.Status

	switch database.Status {
	case DatabaseHealthy:
		return decision, nil

	case DatabaseUnknown:
		decision.Action = ActionGatherTelemetry
		decision.Confidence = ConfidenceLow
		decision.Reason = "Database telemetry is unknown: " + database.Reason

	case DatabaseUnavailable:
		decision.Action = ActionInvestigateDatabase
		decision.Confidence = ConfidenceLow
		decision.Reason = "Database is unavailable: " + database.Reason

	case DatabaseReplicationDegraded:
		appUnhealthy := assessment.ApplicationStatus == ApplicationUnhealthy
		networkDegraded := assessment.Network.Status == NetworkDegraded ||
			assessment.Network.Status == NetworkUnreachable

		switch {
		case appUnhealthy && networkDegraded:
			decision.Action = ActionInvestigateAllLayers
		case appUnhealthy:
			decision.Action = ActionInvestigateAppAndDatabase
		case networkDegraded:
			decision.Action = ActionInvestigateDatabaseAndNetwork
		default:
			decision.Action = ActionInvestigateDatabase
		}

		decision.Confidence = ConfidenceMedium
		decision.Reason = "Database replication is degraded: " + database.Reason

	default:
		return RecoveryDecision{}, fmt.Errorf(
			"invalid database status: %q", database.Status,
		)
	}

	return decision, nil
}
