package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atlas.local/control-plane/controller/internal/database"
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
	Assessment       intelligence.RegionAssessment   `json:"assessment"`
	RecoveryDecision intelligence.RecoveryDecision   `json:"recovery_decision"`
	Database         intelligence.DatabaseAssessment `json:"database"`
	DatabaseError    string                          `json:"database_error,omitempty"`
	Routing          routing.Snapshot                `json:"routing"`
	BGPError         string                          `json:"bgp_error,omitempty"`
}

type report struct {
	Timestamp          string              `json:"timestamp"`
	CycleStartedAt     string              `json:"cycle_started_at"`
	CycleCompletedAt   string              `json:"cycle_completed_at"`
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
	metricsAddress := flag.String("metrics-address", ":9090", "Prometheus metrics listen address")
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

	databaseProbes := []databaseProbe{
		{
			Region:  "Mumbai",
			DSN:     os.Getenv("ATLAS_PG_DSN_MUMBAI"),
			EnvName: "ATLAS_PG_DSN_MUMBAI",
		},
		{
			Region:  "Singapore",
			DSN:     os.Getenv("ATLAS_PG_DSN_SINGAPORE"),
			EnvName: "ATLAS_PG_DSN_SINGAPORE",
		},
		{
			Region:  "Frankfurt",
			DSN:     os.Getenv("ATLAS_PG_DSN_FRANKFURT"),
			EnvName: "ATLAS_PG_DSN_FRANKFURT",
		},
	}

	client := &http.Client{Timeout: *timeout}
	observer := routing.Observer{Runner: routing.ExecRunner{}}
	encoder := json.NewEncoder(os.Stdout)
	metrics := &metricsHandler{}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	metricsServer := &http.Server{
		Addr:              *metricsAddress,
		Handler:           metrics,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "metrics server:", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = metricsServer.Shutdown(shutdownCtx)
	}()

	run := func() {
		cycleStarted := time.Now()

		batch := collectConcurrently(
			func() []health.Result {
				appCtx, appCancel := context.WithTimeout(ctx, *timeout+2*time.Second)
				defer appCancel()
				return health.CheckAll(appCtx, client, appRegions)
			},
			probes,
			func(p regionProbe) routingResult {
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

				return result
			},
			databaseProbes,
			func(p databaseProbe) databaseResult {
				result := databaseResult{Probe: p}

				if p.DSN == "" {
					result.Error = p.EnvName + " is not configured"
					return result
				}

				result.Configured = true

				dbCtx, dbCancel := context.WithTimeout(ctx, *timeout)
				defer dbCancel()

				dbObserver, err := database.Open(dbCtx, p.DSN)
				if err != nil {
					result.Error = err.Error()
					return result
				}
				defer dbObserver.Close()

				snapshot, err := dbObserver.Snapshot(dbCtx, p.Region)
				if err != nil {
					result.Error = err.Error()
					return result
				}

				result.Snapshot = snapshot
				return result
			},
		)

		// Collect application results and update the tracker in this goroutine.
		appResults := batch.Applications
		transitions := tracker.Observe(appResults)

		// Collect routing observations and combine both monitoring layers.
		regionStates := make([]regionReport, 0, len(probes))

		for _, result := range batch.Routing {

			assessment, err := intelligence.EvaluateRegion(
				result.Probe.Name,
				applicationStatus(tracker.Status(result.Probe.Name)),
				result.Snapshot,
			)

			if err != nil {
				fmt.Fprintln(os.Stderr, "assessment error:", err)
				continue
			}

			decision, err := intelligence.BuildRecoveryDecision(assessment)
			if err != nil {
				fmt.Fprintln(os.Stderr, "recovery decision error:", err)
				continue
			}

			dbAssessment := intelligence.AssessDatabase(
				result.Probe.Name,
				database.Snapshot{},
				nil,
				2,
			)
			dbError := ""

			for _, dbResult := range batch.Databases {
				if dbResult.Probe.Region != result.Probe.Name {
					continue
				}

				dbError = dbResult.Error

				if !dbResult.Configured {
					// Missing configuration is unknown, not a database outage.
					dbAssessment.Reason = dbResult.Error
				} else if dbResult.Error != "" {
					dbAssessment = intelligence.AssessDatabase(
						result.Probe.Name,
						dbResult.Snapshot,
						errors.New(dbResult.Error),
						2,
					)
				} else {
					dbAssessment = intelligence.AssessDatabase(
						result.Probe.Name,
						dbResult.Snapshot,
						nil,
						2,
					)
				}
				break
			}

			decision, err = intelligence.BuildRecoveryDecisionWithDatabase(
				assessment,
				dbAssessment,
			)
			if err != nil {
				fmt.Fprintln(os.Stderr, "database-aware recovery decision error:", err)
				continue
			}

			regionStates = append(regionStates, regionReport{
				Assessment:       assessment,
				RecoveryDecision: decision,
				Database:         dbAssessment,
				DatabaseError:    dbError,
				Routing:          result.Snapshot,
				BGPError:         result.Error,
			})
		}

		cycleCompleted := time.Now()

		output := report{
			Timestamp:          cycleCompleted.UTC().Format(time.RFC3339Nano),
			CycleStartedAt:     cycleStarted.UTC().Format(time.RFC3339Nano),
			CycleCompletedAt:   cycleCompleted.UTC().Format(time.RFC3339Nano),
			ApplicationResults: appResults,
			HealthTransitions:  transitions,
			Regions:            regionStates,
		}

		metrics.Update(output)

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
