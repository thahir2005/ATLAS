package intelligence

import "fmt"

type RecoveryAction string
type ConfidenceLevel string

const (
	ActionContinueMonitoring RecoveryAction = "CONTINUE_MONITORING"
	ActionInvestigateNetwork RecoveryAction = "INVESTIGATE_NETWORK"
	ActionInvestigateApp     RecoveryAction = "INVESTIGATE_APPLICATION"
	ActionInvestigateBoth    RecoveryAction = "INVESTIGATE_APPLICATION_AND_NETWORK"
	ActionGatherTelemetry    RecoveryAction = "GATHER_TELEMETRY"

	ConfidenceHigh   ConfidenceLevel = "HIGH"
	ConfidenceMedium ConfidenceLevel = "MEDIUM"
	ConfidenceLow    ConfidenceLevel = "LOW"

	DecisionModeAdvisory = "ADVISORY_ONLY"
)

type RecoveryDecision struct {
	Region                              string          `json:"region"`
	Classification                      Classification  `json:"classification"`
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
