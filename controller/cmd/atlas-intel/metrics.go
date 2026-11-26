package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"atlas.local/control-plane/controller/internal/intelligence"
)

type metricsHandler struct {
	mu   sync.RWMutex
	body string
}

func metricHeader(b *strings.Builder, name, help, metricType string) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s %s\n", name, metricType)
}

func (m *metricsHandler) Update(r report) {
	var b strings.Builder

	metricHeader(&b, "atlas_application_up", "Whether the regional application health check succeeded.", "gauge")
	metricHeader(&b, "atlas_application_latency_seconds", "Regional application health check latency in seconds.", "gauge")
	metricHeader(&b, "atlas_bgp_peers_established", "Number of established BGP peers observed in a region.", "gauge")
	metricHeader(&b, "atlas_bgp_peers_total", "Number of BGP peers observed in a region.", "gauge")
	metricHeader(&b, "atlas_bgp_peer_up", "Whether an individual BGP peer is established.", "gauge")
	metricHeader(&b, "atlas_database_up", "Database availability: 1 available, 0 unavailable, -1 unknown.", "gauge")
	metricHeader(&b, "atlas_database_replication_lag_bytes", "Maximum observed PostgreSQL replication lag in bytes on a primary.", "gauge")
	metricHeader(&b, "atlas_database_status", "One-hot PostgreSQL assessment status by region.", "gauge")
	metricHeader(&b, "atlas_recovery_decision_info", "Current advisory recovery decision by region.", "gauge")

	for _, app := range r.ApplicationResults {
		up := 0
		if app.Healthy {
			up = 1
		}

		fmt.Fprintf(&b, "atlas_application_up{region=%q} %d\n", app.Region, up)
		fmt.Fprintf(&b, "atlas_application_latency_seconds{region=%q} %g\n", app.Region, app.LatencyMS/1000)
	}

	for _, region := range r.Regions {
		name := region.Assessment.Region

		total := len(region.Routing.Peers)
		established := 0

		for address, peer := range region.Routing.Peers {
			up := 0
			if strings.EqualFold(peer.State, "Established") {
				up = 1
				established++
			}

			fmt.Fprintf(
				&b,
				"atlas_bgp_peer_up{region=%q,peer=%q} %d\n",
				name,
				address,
				up,
			)
		}

		fmt.Fprintf(&b, "atlas_bgp_peers_established{region=%q} %d\n", name, established)
		fmt.Fprintf(&b, "atlas_bgp_peers_total{region=%q} %d\n", name, total)

		dbUp := -1
		switch region.Database.Status {
		case intelligence.DatabaseHealthy, intelligence.DatabaseReplicationDegraded:
			dbUp = 1
		case intelligence.DatabaseUnavailable:
			dbUp = 0
		}

		fmt.Fprintf(&b, "atlas_database_up{region=%q} %d\n", name, dbUp)
		fmt.Fprintf(
			&b,
			"atlas_database_status{region=%q,status=%q} 1\n",
			name,
			region.Database.Status,
		)

		if strings.EqualFold(region.Database.Role, "PRIMARY") {
			fmt.Fprintf(
				&b,
				"atlas_database_replication_lag_bytes{region=%q} %d\n",
				name,
				region.Database.MaxLagBytes,
			)
		}

		fmt.Fprintf(
			&b,
			"atlas_recovery_decision_info{region=%q,action=%q,classification=%q,database_status=%q} 1\n",
			name,
			region.RecoveryDecision.Action,
			region.RecoveryDecision.Classification,
			region.RecoveryDecision.DatabaseStatus,
		)
	}

	m.mu.Lock()
	m.body = b.String()
	m.mu.Unlock()
}

func (m *metricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/metrics" {
		http.NotFound(w, r)
		return
	}

	m.mu.RLock()
	body := m.body
	m.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	_, _ = w.Write([]byte(body))
}
