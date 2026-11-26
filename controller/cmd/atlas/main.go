package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"atlas.local/control-plane/controller/internal/health"
)

type Report struct {
	Timestamp      string          `json:"timestamp"`
	Status         string          `json:"status"`
	HealthyRegions int             `json:"healthy_regions"`
	TotalRegions   int             `json:"total_regions"`
	Results        []health.Result `json:"results"`
}

func main() {
	once := flag.Bool("once", false, "Run one health-check cycle")
	interval := flag.Duration("interval", 5*time.Second, "Health-check interval")
	timeout := flag.Duration("timeout", 2*time.Second, "Per-request timeout")
	flag.Parse()

	if *interval <= 0 || *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "interval and timeout must be positive")
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
	fmt.Println("Mode: Read-only monitoring")
	fmt.Println("Regions: Mumbai | Singapore | Frankfurt")

	run := func() {
		ctx := context.Background()
		results := health.CheckAll(ctx, client, regions)

		report := Report{
			Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
			Status:       "healthy",
			TotalRegions: len(regions),
			Results:      results,
		}

		for _, result := range results {
			if result.Healthy {
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

	for {
		run()

		if *once {
			return
		}

		time.Sleep(*interval)
	}
}
