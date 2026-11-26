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
	"atlas.local/control-plane/controller/internal/intelligence"
	"atlas.local/control-plane/controller/internal/routing"
)

type regionProbe struct {
	Name      string
	Namespace string
	Prefix    string
	Target    string
}

type routingResult struct {
	Probe    regionProbe
	Snapshot routing.Snapshot
	Error    string
}

type regionReport struct {
	Assessment intelligence.RegionAssessment `json:"assessment"`
	Routing    routing.Snapshot              `json:"routing"`
	BGPError   string                        `json:"bgp_error,omitempty"`
}

type report struct {
	Timestamp          string              `json:"timestamp"`
	ApplicationResults []health.Result     `json:"application_results"`
	HealthTransitions  []health.Transition `json:"health_transitions"`
	Regions            []regionReport      `json:"regions"`
}

func applicationStatus(status string) intelligence.ApplicationStatus {
	switch status {
	case "healthy":
		return intelligence.ApplicationHealthy
	case "unhealthy":
		return intelligence.ApplicationUnhealthy
	default:
		return intelligence.ApplicationUnknown
	}
}

func main() {
	once := flag.Bool("once", false, "Run one assessment and exit")
	interval := flag.Duration("interval", 5*time.Second, "Monitoring interval")
	timeout := flag.Duration("timeout", 4*time.Second, "Per-observation timeout")
	failureThreshold := flag.Int("failure-threshold", 3, "Consecutive failures before declaring unhealthy")
	recoveryThreshold := flag.Int("recovery-threshold", 2, "Consecutive successes before declaring healthy")
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
		fmt.Fprintln(os.Stderr, "health tracker:", err)
		os.Exit(1)
	}

	probes := []regionProbe{
		{
			Name:      "Mumbai",
			Namespace: "atlas-mumbai",
			Prefix:    "10.10.2.1/32",
			Target:    "10.10.2.1",
		},
		{
			Name:      "Singapore",
			Namespace: "atlas-singapore",
			Prefix:    "10.10.3.1/32",
			Target:    "10.10.3.1",
		},
		{
			Name:      "Frankfurt",
			Namespace: "atlas-frankfurt",
			Prefix:    "10.10.1.1/32",
			Target:    "10.10.1.1",
		},
	}

	appRegions := []health.Region{
		{Name: "Mumbai", URL: "http://10.10.1.1:8080/health"},
		{Name: "Singapore", URL: "http://10.10.2.1:8080/health"},
		{Name: "Frankfurt", URL: "http://10.10.3.1:8080/health"},
	}

	client := &http.Client{Timeout: *timeout}
	observer := routing.Observer{Runner: routing.ExecRunner{}}
	encoder := json.NewEncoder(os.Stdout)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	run := func() {
		checkCtx, cancel := context.WithTimeout(ctx, *timeout+2*time.Second)
		defer cancel()

		appResults := health.CheckAll(checkCtx, client, appRegions)
		transitions := tracker.Observe(appResults)

		resultChannel := make(chan routingResult, len(probes))

		for _, probe := range probes {
			go func(p regionProbe) {
				probeCtx, probeCancel := context.WithTimeout(ctx, *timeout)
				defer probeCancel()

				snapshot, err := observer.Snapshot(
					probeCtx,
					p.Namespace,
					p.Prefix,
					p.Target,
				)

				result := routingResult{
					Probe:    p,
					Snapshot: snapshot,
				}

				if err != nil {
					result.Error = err.Error()
				}

				resultChannel <- result
			}(probe)
		}

		appStates := make([]regionReport, 0, len(probes))

		for range probes {
			result := <-resultChannel

			assessment, err := intelligence.EvaluateRegion(
				result.Probe.Name,
				applicationStatus(tracker.Status(result.Probe.Name)),
				result.Snapshot,
			)

			if err != nil {
				fmt.Fprintln(os.Stderr, "assessment error:", err)
				continue
			}

			appStates = append(appStates, regionReport{
				Assessment: assessment,
				Routing:    result.Snapshot,
				BGPError:   result.Error,
			})
		}

		output := report{
			Timestamp:          time.Now().UTC().Format(time.RFC3339Nano),
			ApplicationResults: appResults,
			HealthTransitions:  transitions,
			Regions:            appStates,
		}

		if err := encoder.Encode(output); err != nil {
			fmt.Fprintln(os.Stderr, "JSON encoding error:", err)
		}
	}

	fmt.Fprintln(os.Stderr, "ATLAS live recovery intelligence started")
	run()

	if *once {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "ATLAS intelligence controller stopped")
			return
		case <-ticker.C:
			run()
		}
	}
}
