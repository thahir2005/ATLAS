package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Region struct {
	Name string
	URL  string
}

type HealthPayload struct {
	Region string `json:"region"`
	Status string `json:"status"`
}

type Result struct {
	Region     string  `json:"region"`
	Endpoint   string  `json:"endpoint"`
	Healthy    bool    `json:"healthy"`
	StatusCode int     `json:"status_code"`
	LatencyMS  float64 `json:"latency_ms"`
	CheckedAt  string  `json:"checked_at"`
	Error      string  `json:"error,omitempty"`
}

func Check(ctx context.Context, client *http.Client, region Region) Result {
	start := time.Now()

	result := Result{
		Region:    region.Name,
		Endpoint:  region.URL,
		CheckedAt: start.UTC().Format(time.RFC3339Nano),
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		region.URL,
		nil,
	)
	if err != nil {
		result.Error = err.Error()
		return finish(result, start)
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		return finish(result, start)
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		result.Error = fmt.Sprintf("unexpected HTTP status: %d", resp.StatusCode)
		return finish(result, start)
	}

	var payload HealthPayload

	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		result.Error = fmt.Sprintf("invalid health response: %v", err)
		return finish(result, start)
	}

	if payload.Region != region.Name {
		result.Error = fmt.Sprintf(
			"region mismatch: expected %s, received %s",
			region.Name,
			payload.Region,
		)
		return finish(result, start)
	}

	if payload.Status != "healthy" {
		result.Error = fmt.Sprintf(
			"unhealthy service status: %s",
			payload.Status,
		)
		return finish(result, start)
	}

	result.Healthy = true
	return finish(result, start)
}

func finish(result Result, start time.Time) Result {
	result.LatencyMS = float64(time.Since(start).Microseconds()) / 1000
	return result
}

func CheckAll(
	ctx context.Context,
	client *http.Client,
	regions []Region,
) []Result {
	results := make([]Result, len(regions))

	done := make(chan struct{}, len(regions))

	for i, region := range regions {
		go func(index int, current Region) {
			results[index] = Check(ctx, client, current)
			done <- struct{}{}
		}(i, region)
	}

	for range regions {
		<-done
	}

	return results
}
