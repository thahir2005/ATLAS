# ATLAS
## Autonomous Multi-Region Disaster Evacuation Engine

ATLAS is a distributed-infrastructure engineering project that explores how a multi-region system can detect failures, maintain leadership coordination, monitor database health, and make controlled recovery decisions.

It simulates three regions — Mumbai, Singapore, and Frankfurt — using Linux network namespaces, BGP routing, etcd consensus, PostgreSQL replication, Go services, and a Prometheus/Grafana observability stack.

> Project status: Working infrastructure simulation with tested failure-detection, database-replication, monitoring, and recovery-decision workflows.

## Architecture

                    ATLAS INFRASTRUCTURE

    +-------------+       eBGP       +-------------+
    |   MUMBAI    |<---------------->|  SINGAPORE  |
    |  ASN 65001  |                  |  ASN 65002  |
    +-------------+                  +-------------+
           ^                                |
           |                                |
           | eBGP                           | eBGP
           |                                |
           |        +-------------+         |
           +------->|  FRANKFURT  |<--------+
                    |  ASN 65003  |
                    +-------------+

                 ATLAS CONTROL PLANE

    +----------------------+ 
    | Health Observation   |
    +----------+-----------+
               |
               v
    +----------------------+
    | Go Intelligence      |
    | Controller           |
    +----------+-----------+
               |
               v
    +----------------------+
    | Recovery Decision    |
    | Engine               |
    +----------+-----------+
               |
               v
    +----------------------+
    | etcd Consensus       |
    | Leases and Fencing   |
    +----------------------+

                 DATABASE LAYER

    +----------------------+
    | Singapore PostgreSQL |
    | Primary              |
    +----------+-----------+
               |
         Async streaming
          replication
          /         \
         v           v
    +----------+  +----------+
    | Mumbai   |  | Frankfurt|
    | Standby  |  | Standby  |
    +----------+  +----------+

                 OBSERVABILITY

    +----------------------+       +----------------------+
    | Prometheus           |------>| Grafana              |
    | Metrics and Alerts   |       | Dashboard            |
    +----------------------+       +----------------------+

## Technology Stack

| Area | Technologies |
|---|---|
| Operating environment | macOS, Colima, Ubuntu |
| Networking | Linux network namespaces, veth, IP routing |
| Dynamic routing | FRRouting, eBGP, BGP |
| Consensus | etcd, leases, leadership coordination |
| Database | PostgreSQL 16, physical streaming replication |
| Backend | Go |
| Database observation | pgx, PostgreSQL monitoring |
| Metrics | Prometheus |
| Dashboards | Grafana |
| Testing | Go unit tests, race detector, failure experiments |

## Regional Topology

| Region | Loopback address | ASN | Database role |
|---|---|---:|---|
| Mumbai | 10.10.1.1 | 65001 | Standby |
| Singapore | 10.10.2.1 | 65002 | Primary |
| Frankfurt | 10.10.3.1 | 65003 | Standby |

The three regions are connected using a full-mesh eBGP topology.

## Key Engineering Features

### 1. Regional Networking

- Linux network namespace-based regional isolation.
- Full-mesh point-to-point connectivity.
- FRRouting and eBGP dynamic routing.
- Regional route observation and failure experiments.

### 2. Consensus and Leadership Coordination

- Three-member etcd cluster.
- Lease-based leadership coordination.
- Monotonically increasing fencing tokens.
- Stale leadership write rejection in the etcd demonstration.

### 3. Database Replication and Recovery

- PostgreSQL physical streaming replication.
- One primary and two standby instances.
- Replication health observation.
- Controlled database promotion and reparenting experiments.
- Database health and replication-state classification.

### 4. Go Intelligence Controller

- Regional application health observation.
- BGP peer monitoring.
- PostgreSQL health observation.
- Concurrent monitoring operations.
- Recovery decision generation.
- Explicit separation of application and database health.

### 5. Observability

- Prometheus metrics collection.
- Application, BGP, and database monitoring.
- Critical and warning alert rules.
- Grafana dashboard provisioning.
- Controlled application and database failure alerting.

## Recovery Decision Model

ATLAS evaluates observed infrastructure health and produces recovery decisions.

Examples demonstrated in the project:

| Observed condition | Recovery decision |
|---|---|
| All monitored components healthy | CONTINUE_MONITORING |
| Application degraded | INVESTIGATE_APPLICATION |
| Database unavailable | Database investigation |

The controller is designed to support human-reviewed recovery decisions. Automatic failover remains disabled in the current monitoring experiments.

## Failure Experiments

The project includes 14 documented failure experiments.

| Experiment | Description |
|---:|---|
| 01 | Static route failure |
| 02 | BGP dynamic routing |
| 03 | BGP link failover |
| 04 | HTTP service failover |
| 05 | Application health detection |
| 06 | Hysteresis-based failure detection |
| 07 | Go BGP event detection |
| 08 | Independent monitoring under latency |
| 09 | etcd consensus and leader failover |
| 10 | etcd leases and fencing |
| 11 | PostgreSQL streaming replication and failover |
| 12 | Prometheus application failure alerting |
| 13 | PostgreSQL database failure alerting |
| 14 | Recovery decision validation |

Experiment reports and supporting evidence are stored in `experiments/`.

## Observability Dashboard

Grafana dashboard:

**ATLAS - Multi-Region Observability**

Local dashboard URL:

http://localhost:3000/d/atlas-overview/atlas-multi-region-observability

The dashboard includes:

- Application health by region.
- Application response latency.
- BGP peer status.
- Established BGP peer counts.
- Database availability.
- Primary database replication lag.
- Recovery decisions.
- Prometheus scrape health.

Prometheus and Grafana configuration files are maintained in `observability/`.

## Validation

Run Go unit tests:

    go test ./...

Run the race detector:

    go test -race ./...

Validate the observability deployment script:

    bash -n scripts/deploy-observability.sh

Deploy the observability configuration in the configured Colima environment:

    ./scripts/deploy-observability.sh

## Repository Structure

    atlas/
    ├── architecture/
    ├── infra/
    │   ├── network/
    │   ├── regions/
    │   ├── consensus/
    │   └── database/
    ├── controller/
    │   ├── cmd/
    │   └── internal/
    ├── services/
    ├── experiments/
    │   ├── failures/
    │   └── benchmarks/
    ├── observability/
    │   ├── prometheus/
    │   └── grafana/
    ├── docs/
    │   ├── adr/
    │   └── runbooks/
    ├── scripts/
    ├── build/
    ├── go.mod
    └── README.md

## Important Limitations

This is an infrastructure simulation and engineering portfolio project, not a production-ready geographically distributed deployment.

- All three simulated regions currently run inside one Colima VM on one physical Mac.
- They do not represent independent physical or geographic failure domains.
- PostgreSQL replication is asynchronous; zero data loss is not guaranteed.
- The etcd fencing demonstration protects etcd writes; it does not automatically fence PostgreSQL or arbitrary external resources.
- Controlled failure experiments demonstrate detection and recovery behavior, not production-grade disaster recovery guarantees.
- Automatic database failover is not enabled in the current observability experiments.

## Project Goals

The long-term goal is to evolve ATLAS into a more realistic distributed recovery platform with independently deployed regions, stronger resource-side fencing, measurable recovery objectives, and comprehensive failure testing.

---

**ATLAS — Observe failures. Coordinate recovery. Protect system integrity.**
