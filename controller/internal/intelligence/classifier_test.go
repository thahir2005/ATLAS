package intelligence

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		name        string
		application ApplicationStatus
		network     NetworkStatus
		expected    Classification
	}{
		{
			name:        "both healthy",
			application: ApplicationHealthy,
			network:     NetworkHealthy,
			expected:    Healthy,
		},
		{
			name:        "network degraded",
			application: ApplicationHealthy,
			network:     NetworkDegraded,
			expected:    NetworkDegradedResult,
		},
		{
			name:        "network unreachable",
			application: ApplicationHealthy,
			network:     NetworkUnreachable,
			expected:    NetworkDegradedResult,
		},
		{
			name:        "application degraded",
			application: ApplicationUnhealthy,
			network:     NetworkHealthy,
			expected:    ApplicationDegraded,
		},
		{
			name:        "application unhealthy and network degraded",
			application: ApplicationUnhealthy,
			network:     NetworkDegraded,
			expected:    RegionDegraded,
		},
		{
			name:        "application unhealthy and network unreachable",
			application: ApplicationUnhealthy,
			network:     NetworkUnreachable,
			expected:    RegionDegraded,
		},
		{
			name:        "application unhealthy but network unknown",
			application: ApplicationUnhealthy,
			network:     NetworkUnknown,
			expected:    ApplicationDegraded,
		},
		{
			name:        "application unknown and network degraded",
			application: ApplicationUnknown,
			network:     NetworkDegraded,
			expected:    NetworkDegradedResult,
		},
		{
			name:        "both unknown",
			application: ApplicationUnknown,
			network:     NetworkUnknown,
			expected:    Unknown,
		},
		{
			name:        "application unknown and network healthy",
			application: ApplicationUnknown,
			network:     NetworkHealthy,
			expected:    Unknown,
		},
		{
			name:        "application healthy and network unknown",
			application: ApplicationHealthy,
			network:     NetworkUnknown,
			expected:    Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Classify(Observation{
				Region:            "mumbai",
				ApplicationStatus: tt.application,
				NetworkStatus:     tt.network,
			})

			if err != nil {
				t.Fatalf("Classify returned error: %v", err)
			}

			if result.Classification != tt.expected {
				t.Errorf("expected %s, got %s",
					tt.expected, result.Classification)
			}

			if result.Region != "mumbai" {
				t.Errorf("expected region mumbai, got %s", result.Region)
			}

			if result.Reason == "" {
				t.Error("expected a classification reason")
			}
		})
	}
}

func TestClassifyRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		observation Observation
	}{
		{
			name: "empty region",
			observation: Observation{
				ApplicationStatus: ApplicationHealthy,
				NetworkStatus:     NetworkHealthy,
			},
		},
		{
			name: "invalid application status",
			observation: Observation{
				Region:            "mumbai",
				ApplicationStatus: "BROKEN",
				NetworkStatus:     NetworkHealthy,
			},
		},
		{
			name: "invalid network status",
			observation: Observation{
				Region:            "mumbai",
				ApplicationStatus: ApplicationHealthy,
				NetworkStatus:     "BROKEN",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Classify(tt.observation)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
