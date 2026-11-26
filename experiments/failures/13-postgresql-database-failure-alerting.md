# Experiment 13: PostgreSQL Database Failure Detection and Alerting

## Objective
Verify that ATLAS detects PostgreSQL database unavailability, exposes the
correct Prometheus metrics, triggers a critical alert, and recognizes recovery.

## Environment
- Platform: macOS with Colima Ubuntu VM
- Regions: Mumbai, Singapore, Frankfurt
- Monitoring: Prometheus and Grafana
- Controller: atlas-intel
- Database: PostgreSQL 16

Note: All three simulated regions run inside one Colima VM on one physical
machine. They are not independent physical or geographic failure domains.

## Initial Database Topology

| Region | Role | Initial state |
|---|---|---|
| Mumbai | Standby | Streaming |
| Singapore | Primary | Active |
| Frankfurt | Standby | Streaming |

Both replicas were connected to Singapore using asynchronous replication.

## Failure Injection

The Singapore PostgreSQL primary process (PID 24639) was temporarily
suspended using SIGSTOP.

Only the Singapore PostgreSQL process was affected. No database files
were deleted, and no replica was promoted.

## Detection Evidence

ATLAS reported:

    atlas_database_up{region="Singapore"} 0
    atlas_database_status{region="Singapore",status="DATABASE_UNAVAILABLE"} 1

Prometheus reported the alert:

    alertname: AtlasDatabaseUnavailable
    region: Singapore
    severity: critical
    state: firing

The controller detected the failure on its first observation check.
This is not a precise failure-detection latency measurement.

## Recovery

The PostgreSQL process was resumed using SIGCONT.

ATLAS subsequently reported:

    atlas_database_up{region="Singapore"} 1

Database roles after recovery:

| Region | Role after recovery |
|---|---|
| Mumbai | Standby |
| Singapore | Primary |
| Frankfurt | Standby |

Replication status after recovery:

| Replica | Replication state | Sync mode |
|---|---|---|
| Frankfurt | streaming | async |
| Mumbai | streaming | async |

The AtlasDatabaseUnavailable alert was no longer listed after recovery.

## Result

PASS — ATLAS detected the simulated Singapore database outage, exposed
DATABASE_UNAVAILABLE metrics, triggered a critical Prometheus alert, and
recognized recovery after the database process resumed.

## Scope and Limitations

- This experiment tested database failure detection and alerting.
- It did not test automatic database promotion or automatic failover.
- No zero-data-loss guarantee is established.
- PostgreSQL replicas use asynchronous replication.
- The simulated regions share one physical host and Colima VM.
- The test used SIGSTOP/SIGCONT, not a real host or storage failure.

## Conclusion

The experiment validates the database observability path from PostgreSQL
health observation through ATLAS metrics to Prometheus alerting and recovery
recognition.
