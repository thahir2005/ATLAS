package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atlas.local/control-plane/controller/internal/health"
)

type Report struct {
	Timestamp      string              `json:"timestamp"`
	Status         string              `json:"status"`
	HealthyRegions int                 `json:"healthy_regions"`
	TotalRegions   int                 `json:"total_regions"`
	RegionStates   map[string]string   `json:"region_states"`
	Results        []health.Result     `json:"results"`
	Events         []health.Transition `json:"events"`
}

func main() {
	once := flag.Bool("once", false, "Run one health-check cycle")
	interval := flag.Duration("interval", 5*time.Second, "Health-check interval")
	timeout := flag.Duration("timeout", 2*time.Second, "Per-request timeout")
	failureThreshold := flag.Int("failure-threshold", 3, "Consecutive failures before unhealthy")
	recoveryThreshold := flag.Int("recovery-threshold", 2, "Consecutive successes before healthy")
	flag.Parse()

	if *interval <= 0 || *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "interval and timeout must be positive")
		os.Exit(1)
	}

	tracker, err := health.NewTrackerWithThresholds(
		*failureThreshold,
		*recoveryThreshold,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid health thresholds:", err)
		os.Exit(1)
	}

	regions := []health.Region{
		{Name: "Mumbai", URL: "http://10.10.1.1:8080/health"},
		{Name: "Singapore", URL: "http://10.10.2.1:8080/health"},
		{Name: "Frankfurt", URL: "http://10.10.3.1:8080/health"},
	}

	client := &http.Client{
		Timeout: *timeout,
	}

	fmt.Println("ATLAS Regional Health Controller")
	fmt.Println("Mode: Continuous read-only monitoring")
	fmt.Println("Regions: Mumbai | Singapore | Frankfurt")
	fmt.Println("Interval:", *interval)
	fmt.Println("Failure threshold:", *failureThreshold)
	fmt.Println("Recovery threshold:", *recoveryThreshold)

	run := func() {
		ctx := context.Background()
		results := health.CheckAll(ctx, client, regions)
		events := tracker.Observe(results)

		report := Report{
			Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
			Status:       "healthy",
			TotalRegions: len(regions),
			RegionStates: make(map[string]string),
			Results:      results,
			Events:       events,
		}

		for _, result := range results {
			state := tracker.Status(result.Region)
			report.RegionStates[result.Region] = state

			if state == "healthy" {
				report.HealthyRegions++
			} else {
				report.Status = "degraded"
			}
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, "report encoding error:", err)
		}
	}

	if *once {
		run()
		return
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	for {
		run()

		select {
		case <-ctx.Done():
			fmt.Println("ATLAS monitoring stopped gracefully")
			return
		case <-time.After(*interval):
		}
	}
}
