package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"atlas.local/control-plane/controller/internal/database"
)

func main() {
	region := flag.String("region", "", "Region name")
	dsn := flag.String("dsn", os.Getenv("ATLAS_PG_DSN"), "PostgreSQL connection string")
	timeout := flag.Duration("timeout", 5*time.Second, "Database observation timeout")
	flag.Parse()

	if *region == "" {
		fmt.Fprintln(os.Stderr, "ERROR: --region is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	observer, err := database.Open(ctx, *dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "DATABASE_CONNECTION_ERROR:", err)
		os.Exit(1)
	}
	defer observer.Close()

	snapshot, err := observer.Snapshot(ctx, *region)
	if err != nil {
		fmt.Fprintln(os.Stderr, "DATABASE_OBSERVATION_ERROR:", err)
		os.Exit(1)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(snapshot); err != nil {
		fmt.Fprintln(os.Stderr, "JSON_ENCODING_ERROR:", err)
		os.Exit(1)
	}
}
