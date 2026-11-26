package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"time"
)

type Result struct {
	Target         string         `json:"target"`
	Duration       float64        `json:"duration_seconds"`
	Concurrency    int            `json:"concurrency"`
	Total          int            `json:"total_requests"`
	Successful     int            `json:"successful_requests"`
	Failed         int            `json:"failed_requests"`
	SuccessRate    float64        `json:"success_rate_percent"`
	FailureRate    float64        `json:"failure_rate_percent"`
	AverageLatency float64        `json:"average_latency_ms"`
	P50            float64        `json:"p50_latency_ms"`
	P95            float64        `json:"p95_latency_ms"`
	P99            float64        `json:"p99_latency_ms"`
	MinLatency     float64        `json:"min_latency_ms"`
	MaxLatency     float64        `json:"max_latency_ms"`
	Errors         map[string]int `json:"errors,omitempty"`
}

func percentile(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	i := int(float64(len(v))*p+0.999999) - 1
	if i < 0 {
		i = 0
	}
	if i >= len(v) {
		i = len(v) - 1
	}
	return v[i]
}

func main() {
	target := flag.String("url", "http://10.10.2.1:8080/region", "Target URL")
	source := flag.String("source", "10.10.1.1", "Source IP")
	duration := flag.Duration("duration", 30*time.Second, "Test duration")
	concurrency := flag.Int("concurrency", 5, "Concurrent workers")
	timeout := flag.Duration("timeout", 2*time.Second, "Request timeout")
	output := flag.String("output", "", "JSON output file")
	flag.Parse()

	if *duration <= 0 || *concurrency <= 0 || *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "Duration, concurrency and timeout must be positive")
		os.Exit(1)
	}

	u, err := url.ParseRequestURI(*target)
	if err != nil || u.Host == "" {
		fmt.Fprintln(os.Stderr, "Invalid target URL")
		os.Exit(1)
	}

	ip := net.ParseIP(*source)
	if ip == nil {
		fmt.Fprintln(os.Stderr, "Invalid source IP")
		os.Exit(1)
	}

	dialer := &net.Dialer{
		Timeout:   *timeout,
		LocalAddr: &net.TCPAddr{IP: ip},
	}

	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxIdleConns:        *concurrency * 2,
		MaxIdleConnsPerHost: *concurrency,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   *timeout,
	}

	stop := make(chan struct{})
	var wg sync.WaitGroup
	var mu sync.Mutex

	var latencies []float64
	success, failed := 0, 0
	errors := make(map[string]int)

	record := func(ms float64, ok bool, reason string) {
		mu.Lock()
		defer mu.Unlock()

		latencies = append(latencies, ms)
		if ok {
			success++
		} else {
			failed++
			errors[reason]++
		}
	}

	fmt.Println("ATLAS HTTP Benchmark")
	fmt.Println("Target:", *target)
	fmt.Println("Source:", *source)
	fmt.Println("Duration:", *duration)
	fmt.Println("Workers:", *concurrency)

	start := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				select {
				case <-stop:
					return
				default:
				}

				requestStart := time.Now()
				req, err := http.NewRequest(http.MethodGet, *target, nil)

				if err != nil {
					record(float64(time.Since(requestStart).Microseconds())/1000, false, "request_error")
					continue
				}

				resp, err := client.Do(req)

				if err != nil {
					record(float64(time.Since(requestStart).Microseconds())/1000, false, "connection_error")
					continue
				}

				_, readErr := io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				ms := float64(time.Since(requestStart).Microseconds()) / 1000
				ok := resp.StatusCode >= 200 && resp.StatusCode < 300 && readErr == nil

				reason := ""
				if readErr != nil {
					reason = "response_read_error"
				} else if !ok {
					reason = fmt.Sprintf("http_status_%d", resp.StatusCode)
				}

				record(ms, ok, reason)
			}
		}()
	}

	timer := time.NewTimer(*duration)
	<-timer.C
	close(stop)
	wg.Wait()
	transport.CloseIdleConnections()

	elapsed := time.Since(start).Seconds()

	mu.Lock()
	defer mu.Unlock()

	sort.Float64s(latencies)

	total := success + failed

	r := Result{
		Target:      *target,
		Duration:    elapsed,
		Concurrency: *concurrency,
		Total:       total,
		Successful:  success,
		Failed:      failed,
		Errors:      errors,
	}

	if total > 0 {
		r.SuccessRate = float64(success) * 100 / float64(total)
		r.FailureRate = float64(failed) * 100 / float64(total)
	}

	if len(latencies) > 0 {
		sum := 0.0
		for _, v := range latencies {
			sum += v
		}

		r.AverageLatency = sum / float64(len(latencies))
		r.P50 = percentile(latencies, 0.50)
		r.P95 = percentile(latencies, 0.95)
		r.P99 = percentile(latencies, 0.99)
		r.MinLatency = latencies[0]
		r.MaxLatency = latencies[len(latencies)-1]
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("\n===== RESULTS =====")
	fmt.Printf("Total requests: %d\n", r.Total)
	fmt.Printf("Successful: %d\n", r.Successful)
	fmt.Printf("Failed: %d\n", r.Failed)
	fmt.Printf("Success rate: %.2f%%\n", r.SuccessRate)
	fmt.Printf("Failure rate: %.2f%%\n", r.FailureRate)
	fmt.Printf("Average latency: %.3f ms\n", r.AverageLatency)
	fmt.Printf("p50: %.3f ms\n", r.P50)
	fmt.Printf("p95: %.3f ms\n", r.P95)
	fmt.Printf("p99: %.3f ms\n", r.P99)

	if *output != "" {
		if err := os.WriteFile(*output, data, 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("Saved:", *output)
	}

	fmt.Println(string(data))
}
