# ATLAS Observability

## Overview

ATLAS uses Prometheus and Grafana to monitor the simulated Mumbai,
Singapore, and Frankfurt regions.

The monitoring stack collects application health, application latency,
BGP peer status, database availability, replication lag, and recovery
decision metrics.

## Architecture

- `atlas-intel` collects application, BGP, and database observations.
- The controller exposes Prometheus metrics on port `9090`.
- Prometheus scrapes the controller and evaluates alert rules.
- Grafana reads Prometheus data and displays the ATLAS dashboard.

The monitoring controller runs inside the `atlas-mumbai` network namespace.
Prometheus and Grafana run in the Colima VM root namespace.

A dedicated monitoring link connects them:

- Root namespace: `172.30.90.1/30`
- Mumbai namespace: `172.30.90.2/30`

All three simulated regions run on one physical machine inside one Colima VM.
They are not independent physical failure domains.

## Repository layout

```text
observability/
├── README.md
├── prometheus/
│   ├── prometheus.yml
│   └── atlas-alerts.yml
└── grafana/
    ├── dashboards/
    │   └── atlas-overview.json
    └── provisioning/
        ├── datasources/
        │   └── atlas-prometheus.yml
        └── dashboards/
            └── atlas.yml
```

## Prometheus configuration

- Global scrape interval: 5 seconds
- Evaluation interval: 5 seconds
- Prometheus self-monitoring target
- ATLAS controller target: `172.30.90.2:9090`

Alert rules are stored in `prometheus/atlas-alerts.yml`.

| Alert | Purpose | Severity |
|---|---|---|
| AtlasApplicationDown | Detects an unavailable regional application | Critical |
| AtlasBGPPeerDown | Detects a failed BGP peer | Warning |
| AtlasDatabaseUnavailable | Detects unavailable database status | Critical |
| AtlasControllerScrapeFailed | Detects controller scrape failure | Critical |

Each alert currently has a 15-second pending duration.

## Grafana

Grafana uses the `ATLAS-Prometheus` data source.

Dashboard: **ATLAS - Multi-Region Observability**

It includes application health, latency, BGP status, database availability,
replication lag, recovery decisions, and Prometheus scrape health.

Dashboard URL:

http://localhost:3000/d/atlas-overview/atlas-multi-region-observability

## Configuration validation

Validate dashboard JSON:

`python3 -m json.tool observability/grafana/dashboards/atlas-overview.json`

Validate Prometheus configuration:

`promtool check config /etc/prometheus/prometheus.yml`

Validate alert rules:

`promtool check rules /etc/prometheus/atlas-alerts.yml`

Check Prometheus targets:

`curl -fsS http://127.0.0.1:9090/api/v1/targets`

Check active alerts:

`curl -fsS http://127.0.0.1:9090/api/v1/alerts`

## Deployment notes

These files are repository copies of the currently working configuration.
They are not a complete automated installation procedure.

A fresh environment still needs:

- Prometheus and Grafana installed.
- The monitoring network link and namespace connectivity configured.
- The ATLAS controller built and started.
- Prometheus and Grafana provisioning files installed at their configured paths.
- Services restarted or reloaded as appropriate.

The monitoring network link is currently configured at runtime and must be
recreated after a Colima VM restart.

The Grafana data source UID is retained from the existing Grafana instance.
A fresh Grafana installation may require consistent datasource provisioning
and dashboard datasource references.

## Experiment evidence

See:

`experiments/failures/12-prometheus-application-failure-alerting.md`

This experiment validated detection of a suspended Mumbai application and
the firing of the `AtlasApplicationDown` alert.
