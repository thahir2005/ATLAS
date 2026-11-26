# Experiment 03 — Request-Level BGP Failover Telemetry

## Objective
Measure individual HTTP request latency before, during, and after a
controlled Mumbai-Singapore link failure.

## Test configuration
- Source region: Mumbai
- Target region: Singapore
- Target endpoint: /region
- Benchmark duration: 30 seconds
- Concurrent workers: 5
- Request timeout: 2 seconds
- Injected outage: approximately 5.008 seconds
- Event format: JSONL, one record per HTTP request

## Failure timestamps (UTC)
- Link disabled: 2026-11-26T16:21:39.414890003Z
- Link restored: 2026-11-26T16:21:44.422556169Z

## Aggregate results
- Total requests: 2,350,214
- Successful requests: 2,350,214
- Failed requests: 0
- Success rate: 100%
- Average latency: 0.062 ms
- p50: 0.042 ms
- p95: 0.116 ms
- p99: 0.371 ms
- Maximum latency: 208.912 ms

## Latency by request-start phase

| Metric | Before failure | Link down | After restoration |
|---|---:|---:|---:|
| Requests | 398,364 | 368,248 | 1,583,602 |
| p50 (ms) | 0.041 | 0.043 | 0.042 |
| p95 (ms) | 0.121 | 0.122 | 0.113 |
| p99 (ms) | 0.405 | 0.398 | 0.351 |
| Maximum (ms) | 4.984 | 208.912 | 30.225 |
| Requests >= 1 ms | 1,558 | 1,718 | 7,519 |
| Requests >= 10 ms | 0 | 5 | 11 |
| Requests >= 100 ms | 0 | 5 | 0 |

Requests are assigned to phases by their start timestamp. A request can
finish in a later phase than the one in which it started.

## Key observations
- No HTTP request failures were recorded.
- Five requests exceeded 100 ms during the link-down phase.
- All five began approximately 2.8 ms after the link-disable timestamp.
- The maximum request latency was 208.912 ms.
- The maximum latency after restoration was 30.225 ms.
- The request timeline shows a latency disturbance near failure injection,
  despite the 100% request success rate.

## Interpretation
The observations are consistent with a temporary network-path transition
during BGP failover. Successful HTTP responses indicate that the requests
eventually completed.

The 208.912 ms request latency is an application-level observation. It is
not a direct measurement of BGP convergence or proof of the exact time
when the alternate route became active.

The earlier BGP experiment independently measured approximately 137.81 ms
for bidirectional route convergence. A future experiment will correlate
timestamped route observations with request events.

## Limitations
- All regions run inside one Colima VM on one physical Mac.
- This is a controlled network simulation, not independent geographic
  infrastructure.
- Event recording adds disk I/O and synchronization overhead.
- The benchmark does not represent production traffic or internet latency.
- The route was checked after restoration, not continuously during this run.

## Conclusion
ATLAS maintained 100% HTTP request success during the tested link outage.
However, request-level telemetry revealed latency spikes during failure
injection and restoration.

This demonstrates why availability percentages alone are insufficient
for evaluating failover behavior. Latency and route convergence must also
be measured.
