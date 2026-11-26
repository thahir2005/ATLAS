package main

import "atlas.local/control-plane/controller/internal/health"

type observationBatch struct {
	Applications []health.Result
	Routing      []routingResult
}

// collectConcurrently starts application and routing observations
// independently, then returns both sets of results to the aggregator.
func collectConcurrently(
	appCheck func() []health.Result,
	probes []regionProbe,
	routeCheck func(regionProbe) routingResult,
) observationBatch {
	applications := make(chan []health.Result, 1)
	routingResults := make(chan routingResult, len(probes))

	go func() {
		applications <- appCheck()
	}()

	for _, probe := range probes {
		go func(p regionProbe) {
			routingResults <- routeCheck(p)
		}(probe)
	}

	batch := observationBatch{
		Applications: <-applications,
		Routing:      make([]routingResult, 0, len(probes)),
	}

	for range probes {
		batch.Routing = append(batch.Routing, <-routingResults)
	}

	return batch
}
