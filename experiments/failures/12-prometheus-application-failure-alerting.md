# Experiment 12: Prometheus Application Failure Detection and Alerting

## Objective

Validate end-to-end application failure detection using the ATLAS monitoring
controller, Prometheus metrics, and Prometheus alert rules.

## Environment

- Monitoring controller: `atlas-intel`
- Metrics endpoint: `172.30.90.2:9090`
- Prometheus: `127.0.0.1:9090` in the Colima root namespace
- Scrape interval: 5 seconds
- Alert evaluation interval: 5 seconds
- Alert pending duration: 15 seconds
- Test region: Mumbai
- Application port: 8080

The three simulated regions run inside one Colima Linux VM on one physical
machine. They are not independent physical failure domains.

## Experiment procedure

1. Verified Mumbai, Singapore, and Frankfurt health endpoints from the
   `atlas-mumbai` network namespace.
2. Verified all three `atlas_application_up` metrics reported `1`.
3. Verified the Prometheus `atlas-controller` and `prometheus` scrape targets
   were UP.
4. Identified the Mumbai API process.
5. Suspended only that process using `SIGSTOP`.
6. Waited 40 seconds for monitoring and alert evaluation.
7. Queried the controller metric and Prometheus active alerts.
8. Resumed the same process using `SIGCONT`.
9. Verified the Mumbai health endpoint and all three application metrics.

## Commands used

Suspend the Mumbai demo API:

`kill -STOP <MUMBAI_API_PID>`

Resume it:

`kill -CONT <MUMBAI_API_PID>`

Check application metrics:

`ip netns exec atlas-mumbai curl -fsS http://127.0.0.1:9090/metrics`

Check active Prometheus alerts:

`curl -fsS http://127.0.0.1:9090/api/v1/alerts`

## Observed results

| Observation | Result |
|---|---|
| Mumbai metric during suspension | `0` |
| Alert name | `AtlasApplicationDown` |
| Alert region | Mumbai |
| Alert state | `firing` |
| Alert summary | Application is down in Mumbai |
| Mumbai health after resume | `healthy` |
| Mumbai metric after resume | `1` |
| Singapore metric after test | `1` |
| Frankfurt metric after test | `1` |

The tested alert expression was:

`atlas_application_up == 0`

The alert rule uses a 15-second pending duration.

## Result

**PASS** — Prometheus detected the suspended Mumbai application through
the ATLAS controller's application health metric. The
`AtlasApplicationDown` alert entered the `firing` state.

The Mumbai API was resumed successfully. Its health endpoint returned
`healthy`, and all three regional application metrics returned to `1`.

## Limitations

- The test used `SIGSTOP`, which suspends a process without terminating it.
  It validates monitoring of an unresponsive application process, not
  process-crash recovery.
- The 40-second observation window is not an exact alert-detection latency
  measurement.
- This experiment validates alert detection, not automatic application
  restart, traffic failover, or database failover.
- All regions share one physical host and one Colima VM.

## Evidence

Observed terminal output confirmed:

    atlas_application_up{region="Mumbai"} 0

    Alert: AtlasApplicationDown
    Region: Mumbai
    State: firing
    Summary: Application is down in Mumbai

After restoration:

    Mumbai: healthy
    Mumbai: 1
    Singapore: 1
    Frankfurt: 1
