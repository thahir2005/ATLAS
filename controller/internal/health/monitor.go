package health

type Transition struct {
	Event          string `json:"event"`
	Region         string `json:"region"`
	PreviousStatus string `json:"previous_status"`
	CurrentStatus  string `json:"current_status"`
	ObservedAt     string `json:"observed_at"`
	Reason         string `json:"reason,omitempty"`
}

type Tracker struct {
	states map[string]string
}

func NewTracker() *Tracker {
	return &Tracker{
		states: make(map[string]string),
	}
}

func (t *Tracker) Observe(results []Result) []Transition {
	var transitions []Transition

	for _, result := range results {
		current := "unhealthy"
		if result.Healthy {
			current = "healthy"
		}

		previous, exists := t.states[result.Region]

		if !exists {
			transitions = append(transitions, Transition{
				Event:         "REGION_STATE_INITIALIZED",
				Region:        result.Region,
				CurrentStatus: current,
				ObservedAt:    result.CheckedAt,
				Reason:        result.Error,
			})
		} else if previous != current {
			transitions = append(transitions, Transition{
				Event:          "REGION_STATE_CHANGED",
				Region:         result.Region,
				PreviousStatus: previous,
				CurrentStatus:  current,
				ObservedAt:     result.CheckedAt,
				Reason:         result.Error,
			})
		}

		t.states[result.Region] = current
	}

	return transitions
}
