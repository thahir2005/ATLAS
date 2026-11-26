# Experiment 04 — Correlated BGP and HTTP Failover

## Objective

Correlate regional link failure, Linux kernel route changes, and individual
HTTP request latency using timestamps from the same Colima Linux VM.

## Topology

- Source: Mumbai (10.10.1.1)
- Destination: Singapore (10.10.2.1)
- Alternate path: Frankfurt
- Routing: FRRouting eBGP
- HTTP endpoint: Singapore regional API
- Benchmark duration: 30 seconds
- Concurrent workers: 5
- Route observer polling target: 20 ms

## Control and route timestamps (UTC)

| Event | Timestamp |
|---|---|
| Link failure requested | 16:26:34.162493391 |
| Route became unreachable | 16:26:34.167261445 |
| Frankfurt route selected | 16:26:34.261330974 |
| Link restoration requested | 16:26:39.171303575 |
| Direct route selected | 16:26:40.409650667 |

## Routing measurements

- Failure request to observed route loss: approximately 4.77 ms.
- Observed no-route interval: approximately 94.07 ms.
- Failure request to Frankfurt route selection: approximately 98.84 ms.
- Restoration request to direct route selection: approximately 1.238 seconds.

These are observations from this experiment. The route observer uses
periodic polling and command execution, so its timestamps are not
hardware-level link or routing event timestamps.

## HTTP benchmark results

| Metric | Result |
|---|---:|
| Total requests | 1,895,865 |
| Successful requests | 1,895,865 |
| Failed requests | 0 |
| Success rate | 100% |
| Average latency | 0.077 ms |
| p50 latency | 0.049 ms |
| p95 latency | 0.159 ms |
| p99 latency | 0.621 ms |
| Maximum latency | 206.116 ms |

## Request and route correlation

- No request started during the observer's no-route interval.
- Five requests had execution intervals overlapping the no-route interval.
- All five eventually succeeded.
- The five slowest requests started approximately 0.6–1.8 ms after
  the link failure was requested.
- Their latencies were approximately 205–206 ms.
- The maximum observed latency was 206.116 ms.

The five concurrent workers were occupied by in-flight requests during
the transition. This is consistent with why no new request start was
recorded during the observed no-route interval.

## Interpretation

The experiment demonstrates that HTTP success rate alone does not
describe the full impact of a network failure.

All requests succeeded, but several in-flight requests experienced
significant latency while the network transitioned to the alternate path.

The route observer independently recorded the kernel route becoming
unreachable and then selecting Frankfurt.

The timestamps support a correlation between link failure, route
transition, and elevated request latency. They do not prove the precise
network-level cause of every individual request delay.

## Measurement limitations

- The route observer polls every 20 ms; actual scheduling and command
  execution add measurement overhead.
- Request timestamps record request start, not response completion.
- Route selection is not identical to application-level recovery.
- The benchmark uses five concurrent workers.
- All regions run inside one Colima VM on one physical Mac.
- This is a controlled simulation, not independent geographic
  infrastructure or a production SLA measurement.

## Conclusion

ATLAS maintained 100% HTTP request success during the controlled
Mumbai-Singapore link failure.

The correlated measurements recorded an approximately 94.07 ms
observed no-route interval and HTTP latency spikes up to 206.116 ms.

This experiment demonstrates the value of correlating network routing
telemetry with application-level request measurements.
