package intelligence

import (
	"errors"
	"fmt"
	"strings"
)

type ApplicationStatus string
type NetworkStatus string
type Classification string

const (
	ApplicationHealthy   ApplicationStatus = "HEALTHY"
	ApplicationUnhealthy ApplicationStatus = "UNHEALTHY"
	ApplicationUnknown   ApplicationStatus = "UNKNOWN"

	NetworkHealthy     NetworkStatus = "HEALTHY"
	NetworkDegraded    NetworkStatus = "DEGRADED"
	NetworkUnreachable NetworkStatus = "UNREACHABLE"
	NetworkUnknown     NetworkStatus = "UNKNOWN"

	Healthy               Classification = "HEALTHY"
	NetworkDegradedResult Classification = "NETWORK_DEGRADED"
	ApplicationDegraded   Classification = "APPLICATION_DEGRADED"
	RegionDegraded        Classification = "REGION_DEGRADED"
	Unknown               Classification = "UNKNOWN"
)

type Observation struct {
	Region            string
	ApplicationStatus ApplicationStatus
	NetworkStatus     NetworkStatus
}

type Result struct {
	Region            string            `json:"region"`
	Classification    Classification    `json:"classification"`
	ApplicationStatus ApplicationStatus `json:"application_status"`
	NetworkStatus     NetworkStatus     `json:"network_status"`
	Reason            string            `json:"reason"`
}

func Classify(observation Observation) (Result, error) {
	region := strings.TrimSpace(observation.Region)
	if region == "" {
		return Result{}, errors.New("region cannot be empty")
	}

	if !validApplicationStatus(observation.ApplicationStatus) {
		return Result{}, fmt.Errorf("invalid application status: %q", observation.ApplicationStatus)
	}

	if !validNetworkStatus(observation.NetworkStatus) {
		return Result{}, fmt.Errorf("invalid network status: %q", observation.NetworkStatus)
	}

	result := Result{
		Region:            region,
		ApplicationStatus: observation.ApplicationStatus,
		NetworkStatus:     observation.NetworkStatus,
	}

	switch {
	case observation.ApplicationStatus == ApplicationUnhealthy &&
		(observation.NetworkStatus == NetworkDegraded ||
			observation.NetworkStatus == NetworkUnreachable):
		result.Classification = RegionDegraded
		result.Reason = "Application and network are both failing"

	case observation.ApplicationStatus == ApplicationUnhealthy:
		result.Classification = ApplicationDegraded
		result.Reason = "Application is unhealthy"

	case observation.NetworkStatus == NetworkDegraded ||
		observation.NetworkStatus == NetworkUnreachable:
		result.Classification = NetworkDegradedResult
		result.Reason = "Network connectivity or routing is degraded"

	case observation.ApplicationStatus == ApplicationHealthy &&
		observation.NetworkStatus == NetworkHealthy:
		result.Classification = Healthy
		result.Reason = "Application and network are healthy"

	default:
		result.Classification = Unknown
		result.Reason = "Insufficient telemetry to determine regional health"
	}

	return result, nil
}

func validApplicationStatus(status ApplicationStatus) bool {
	switch status {
	case ApplicationHealthy, ApplicationUnhealthy, ApplicationUnknown:
		return true
	default:
		return false
	}
}

func validNetworkStatus(status NetworkStatus) bool {
	switch status {
	case NetworkHealthy, NetworkDegraded, NetworkUnreachable, NetworkUnknown:
		return true
	default:
		return false
	}
}
