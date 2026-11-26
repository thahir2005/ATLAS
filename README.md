# ATLAS
## Autonomous Multi-Region Disaster Evacuation Engine

ATLAS is a hands-on infrastructure engineering project that simulates
multi-region disaster detection, traffic evacuation, database failover,
and controlled regional recovery.

### Simulated regions
- Mumbai
- Singapore
- Frankfurt

### Core technologies
- Linux networking and network namespaces
- FRRouting and BGP
- etcd consensus and leases
- PostgreSQL streaming replication
- Go recovery controller
- Prometheus observability
- Chaos engineering

### Project principles
- Explicit failure detection
- Fenced database leadership
- Monotonically increasing leadership epochs
- Measurable recovery objectives
- Reproducible failure experiments

### Lab limitation
The initial implementation runs on one physical Mac using a Linux VM.
It simulates regional failures but does not provide independent
geographic or physical failure domains.
