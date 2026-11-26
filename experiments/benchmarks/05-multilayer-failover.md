# Experiment 05: Multi-Layer BGP and HTTP Failover

## Objective

Correlate regional BGP neighbor state, BGP best-path selection,
Linux kernel forwarding routes, and HTTP request-level telemetry
during a controlled regional network link failure.

## Environment

- Three Linux network namespaces: Mumbai, Singapore, Frankfurt
- FRRouting eBGP between all three regions
- Regional HTTP API listening on port 8080
- HTTP benchmark duration: 30 seconds
- HTTP concurrency: 5 workers
- HTTP timeout: 2 seconds
- Link outage: approximately 5 seconds
- BGP observer polling target: 100 ms
- Kernel route observer polling target: 20 ms
- All timestamps: UTC

All three namespaces run inside one Colima VM on one physical Mac.
This experiment demonstrates simulated regional failover, not
independent geographic or physical failure domains.

## Failure injection timeline

| Event | Timestamp UTC |
|---|---|
| Link failure requested | 16:30:37.079084336 |
| Link disabled | 16:30:37.083791181 |
| Kernel route lost | 16:30:37.085300948 |
| Singapore BGP peer Idle observed | 16:30:37.113750 |
| Frankfurt kernel route observed | 16:30:37.178212493 |
| Frankfurt BGP best path observed | 16:30:37.251372 |

The observed kernel no-route interval was:

16:30:37.178212493 - 16:30:37.085300948
= approximately 92.91 ms.

The Frankfurt route was observed approximately 94.42 ms after
the link was disabled.

The BGP observer detected the alternate best path approximately
167.58 ms after link disablement.

These are observer detection times, not exact internal protocol
transition times.

## Recovery timeline

| Event | Timestamp UTC |
|---|---|
| Link restoration requested | 16:30:42.085062293 |
| Link restored | 16:30:42.092980508 |
| Singapore BGP peer Established | 16:30:43.207774 |
| Direct kernel route observed | 16:30:44.321649194 |
| Direct BGP best path observed | 16:30:44.389055 |

The preferred direct path returned after BGP session recovery.
Link restoration did not immediately restore the preferred route.

## HTTP benchmark results

| Metric | Result |
|---|---:|
| Duration | 30.0015 seconds |
| Concurrency | 5 |
| Total requests | 1,733,670 |
| Successful requests | 1,733,670 |
| Failed requests | 0 |
| Success rate | 100% |
| Average latency | 0.0848 ms |
| p50 latency | 0.059 ms |
| p95 latency | 0.167 ms |
| p99 latency | 0.713 ms |
| Maximum latency | 202.822 ms |

## Request-level phase analysis

Requests are classified by their start timestamp relative to
the failure and restoration request timestamps.

| Metric | Before failure | During outage | After restoration |
|---|---:|---:|---:|
| Requests | 278,948 | 268,216 | 1,186,506 |
| p50 | 0.060 ms | 0.059 ms | 0.058 ms |
| p95 | 0.181 ms | 0.174 ms | 0.162 ms |
| p99 | 0.779 ms | 0.750 ms | 0.689 ms |
| Maximum | 202.822 ms | 198.232 ms | 16.191 ms |
| Requests >= 10 ms | 3 | 17 | 2 |
| Requests >= 100 ms | 3 | 2 | 0 |
| Failed requests | 0 | 0 | 0 |

Five of the slowest requests started within milliseconds of
the failure injection. Their durations were approximately
198–203 ms. All returned HTTP 200.

The request overlap calculation identified 268,221 requests
whose execution intervals overlapped the five-second link-down
window. This does not mean all those requests were interrupted:
most were successfully served over the alternate path.

## Interpretation

1. The direct Mumbai-Singapore BGP neighbor transitioned from
   Established to Idle after link failure.

2. The kernel temporarily reported the Singapore service
   destination as unreachable.

3. The alternate Frankfurt route became available and HTTP
   requests continued to succeed.

4. The direct BGP session re-established after link restoration.

5. The preferred direct route returned later than the physical
   link restoration event.

6. The five slowest requests clustered around failure injection.
   They completed successfully, indicating latency impact
   without observed HTTP request failures in this run.

## Measurement limitations

- BGP and kernel observers poll independently.
- Polling intervals are targets; command execution adds overhead.
- Observer timestamps represent detection times, not exact
  internal state transition times.
- Request phase classification uses request start timestamps.
- The overlap calculation uses the full five-second link-down
  interval, not just the 92.91 ms no-route interval.
- A successful HTTP response does not prove zero user-perceived
  interruption or zero latency impact.
- All regions run inside one Colima VM on one physical host.

## Artifacts

- multilayer-summary.json
- multilayer-control.csv
- multilayer-bgp.jsonl
- multilayer-routes.csv
- multilayer-events.jsonl.gz (local compressed raw telemetry)

## Conclusion

The experiment successfully demonstrated simulated multi-layer
regional failover. BGP detected the failed peer, selected an
alternate path, and the Linux kernel recovered forwarding through
Frankfurt. The HTTP benchmark recorded zero failed requests across
1,733,670 requests, while request-level telemetry exposed latency
spikes around the failure transition.

This is evidence of successful failover in the tested local
simulation, not proof of production-grade multi-region resilience.
