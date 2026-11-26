package main

import (
	"atlas.local/control-plane/controller/internal/database"
	"atlas.local/control-plane/controller/internal/health"
)

type databaseProbe struct {
	Region  string
	DSN     string
	EnvName string
}

type databaseResult struct {
	Probe      databaseProbe
	Snapshot   database.Snapshot
	Configured bool
	Error      string
}

type observationBatch struct {
	Applications []health.Result
	Routing      []routingResult
	Databases    []databaseResult
}

// collectConcurrently starts application, routing and database observations
// independently, then returns all results to the aggregator.
func collectConcurrently(
	appCheck func() []health.Result,
	probes []regionProbe,
	routeCheck func(regionProbe) routingResult,
	databaseProbes []databaseProbe,
	databaseCheck func(databaseProbe) databaseResult,
) observationBatch {
	applications := make(chan []health.Result, 1)
	routingResults := make(chan routingResult, len(probes))
	databaseResults := make(chan databaseResult, len(databaseProbes))

	go func() {
		applications <- appCheck()
	}()

	for _, probe := range probes {
		go func(p regionProbe) {
			routingResults <- routeCheck(p)
		}(probe)
	}

	for _, probe := range databaseProbes {
		go func(p databaseProbe) {
			databaseResults <- databaseCheck(p)
		}(probe)
	}

	batch := observationBatch{
		Applications: <-applications,
		Routing:      make([]routingResult, 0, len(probes)),
		Databases:    make([]databaseResult, 0, len(databaseProbes)),
	}

	for range probes {
		batch.Routing = append(batch.Routing, <-routingResults)
	}

	for range databaseProbes {
		batch.Databases = append(batch.Databases, <-databaseResults)
	}

	return batch
}
