package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"atlas.local/control-plane/controller/internal/routing"
)

func main() {
	namespace := flag.String("namespace", "atlas-mumbai", "FRR namespace")
	prefix := flag.String("prefix", "10.10.2.1/32", "BGP prefix to inspect")
	target := flag.String("target", "10.10.2.1", "Kernel route destination")
	interval := flag.Duration("interval", 1*time.Second, "Polling interval")
	timeout := flag.Duration("timeout", 5*time.Second, "Command timeout")
	once := flag.Bool("once", false, "Take one snapshot and exit")
	flag.Parse()

	if *timeout <= 0 || *interval <= 0 {
		fmt.Fprintln(os.Stderr, "timeout and interval must be positive")
		os.Exit(1)
	}

	observer := routing.Observer{
		Runner: routing.ExecRunner{},
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	encoder := json.NewEncoder(os.Stdout)
	var previous *routing.Snapshot

	fmt.Fprintf(os.Stderr,
		"ATLAS BGP Monitor: namespace=%s prefix=%s target=%s interval=%s\n",
		*namespace, *prefix, *target, *interval,
	)

	run := func() bool {
		checkCtx, cancel := context.WithTimeout(ctx, *timeout)
		defer cancel()

		current, err := observer.Snapshot(
			checkCtx,
			*namespace,
			*prefix,
			*target,
		)

		if err != nil {
			event := routing.Event{
				Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
				Event:     "OBSERVER_ERROR",
				Details: map[string]string{
					"error": err.Error(),
				},
			}
			_ = encoder.Encode(event)
			return false
		}

		events := routing.CompareSnapshots(previous, current)

		for _, event := range events {
			if err := encoder.Encode(event); err != nil {
				fmt.Fprintln(os.Stderr, "event encoding error:", err)
				return false
			}
		}

		previous = &current
		return true
	}

	run()

	if *once {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "ATLAS BGP monitor stopped gracefully")
			return
		case <-ticker.C:
			run()
		}
	}
}
