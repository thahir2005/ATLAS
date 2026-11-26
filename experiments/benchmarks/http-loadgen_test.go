package main

import "testing"

func TestPercentile(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	tests := []struct {
		p    float64
		want float64
	}{
		{0.50, 5},
		{0.95, 10},
		{0.99, 10},
	}

	for _, test := range tests {
		got := percentile(values, test.p)
		if got != test.want {
			t.Errorf("got %v, want %v", got, test.want)
		}
	}
}

func TestEmptyPercentile(t *testing.T) {
	if percentile(nil, 0.95) != 0 {
		t.Error("expected zero for empty values")
	}
}
