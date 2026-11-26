package health

import "fmt"

type Transition struct {
	Event                string `json:"event"`
	Region               string `json:"region"`
	PreviousStatus       string `json:"previous_status"`
	CurrentStatus        string `json:"current_status"`
	ObservedAt           string `json:"observed_at"`
	Reason               string `json:"reason,omitempty"`
	ConsecutiveFailures  int    `json:"consecutive_failures,omitempty"`
	ConsecutiveSuccesses int    `json:"consecutive_successes,omitempty"`
}

type regionState struct {
	status               string
	consecutiveFailures  int
	consecutiveSuccesses int
}

type Tracker struct {
	states            map[string]*regionState
	failureThreshold  int
	recoveryThreshold int
}

func NewTrackerWithThresholds(failureThreshold, recoveryThreshold int) (*Tracker, error) {
	if failureThreshold <= 0 {
		return nil, fmt.Errorf("failure threshold must be positive")
	}

	if recoveryThreshold <= 0 {
		return nil, fmt.Errorf("recovery threshold must be positive")
	}

	return &Tracker{
		states:            make(map[string]*regionState),
		failureThreshold:  failureThreshold,
		recoveryThreshold: recoveryThreshold,
	}, nil
}

func (t *Tracker) Observe(results []Result) []Transition {
	var transitions []Transition

	for _, result := range results {
		current := "unhealthy"
		if result.Healthy {
			current = "healthy"
		}

		state, exists := t.states[result.Region]

		if !exists {
			t.states[result.Region] = &regionState{
				status: current,
			}

			transitions = append(transitions, Transition{
				Event:         "REGION_STATE_INITIALIZED",
				Region:        result.Region,
				CurrentStatus: current,
				ObservedAt:    result.CheckedAt,
				Reason:        result.Error,
			})
			continue
		}

		if state.status == current {
			if current == "healthy" {
				state.consecutiveFailures = 0
			} else {
				state.consecutiveSuccesses = 0
			}
			continue
		}

		if state.status == "healthy" && current == "unhealthy" {
			state.consecutiveFailures++
			state.consecutiveSuccesses = 0

			if state.consecutiveFailures == 1 {
				transitions = append(transitions, Transition{
					Event:               "REGION_FAILURE_SUSPECTED",
					Region:              result.Region,
					PreviousStatus:      "healthy",
					CurrentStatus:       "suspected_unhealthy",
					ObservedAt:          result.CheckedAt,
					Reason:              result.Error,
					ConsecutiveFailures: state.consecutiveFailures,
				})
			}

			if state.consecutiveFailures >= t.failureThreshold {
				transitions = append(transitions, Transition{
					Event:               "REGION_STATE_CHANGED",
					Region:              result.Region,
					PreviousStatus:      "healthy",
					CurrentStatus:       "unhealthy",
					ObservedAt:          result.CheckedAt,
					Reason:              result.Error,
					ConsecutiveFailures: state.consecutiveFailures,
				})

				state.status = "unhealthy"
				state.consecutiveFailures = 0
				state.consecutiveSuccesses = 0
			}

			continue
		}

		if state.status == "unhealthy" && current == "healthy" {
			state.consecutiveSuccesses++
			state.consecutiveFailures = 0

			if state.consecutiveSuccesses == 1 {
				transitions = append(transitions, Transition{
					Event:                "REGION_RECOVERY_SUSPECTED",
					Region:               result.Region,
					PreviousStatus:       "unhealthy",
					CurrentStatus:        "recovery_pending",
					ObservedAt:           result.CheckedAt,
					ConsecutiveSuccesses: state.consecutiveSuccesses,
				})
			}

			if state.consecutiveSuccesses >= t.recoveryThreshold {
				transitions = append(transitions, Transition{
					Event:                "REGION_STATE_CHANGED",
					Region:               result.Region,
					PreviousStatus:       "unhealthy",
					CurrentStatus:        "healthy",
					ObservedAt:           result.CheckedAt,
					ConsecutiveSuccesses: state.consecutiveSuccesses,
				})

				state.status = "healthy"
				state.consecutiveFailures = 0
				state.consecutiveSuccesses = 0
			}
		}
	}

	return transitions
}

func (t *Tracker) Status(region string) string {
	state, exists := t.states[region]
	if !exists {
		return "unknown"
	}
	return state.status
}
