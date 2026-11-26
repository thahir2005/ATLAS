# Experiment 02: HTTP Benchmark During BGP Link Failure

## Objective
Measure HTTP service availability and request latency while the direct
Mumbai-Singapore network link is deliberately disabled and restored.

## Environment
- Mumbai region: `10.10.1.1`
- Singapore region: `10.10.2.1`
- Frankfurt region: `10.10.3.1`
- HTTP endpoint: `http://10.10.2.1:8080/region`
- Benchmark duration: 30 seconds
- Concurrent workers: 5
- Request timeout: 2 seconds
- Source address: `10.10.1.1`

## Failure Injection
1. Verified the Singapore health endpoint was healthy.
2. Started the 30-second HTTP benchmark.
3. Waited 5 seconds after benchmark start.
4. Disabled Mumbai's direct Singapore interface (`ms0`).
5. Kept the link disabled for approximately 5 seconds.
6. Restored the Mumbai-Singapore link.
7. Waited for the benchmark to finish.
8. Verified both link endpoints were UP and Singapore's health endpoint was healthy.

## Results

| Metric | Healthy baseline | Link failure test |
|---|---:|---:|
| Total requests | 2,714,898 | 2,656,971 |
| Successful requests | 2,714,898 | 2,656,971 |
| Failed requests | 0 | 0 |
| Success rate | 100.00% | 100.00% |
| Throughput | 90,492.92 req/s | 88,564.80 req/s |
| Average latency | 0.0546 ms | 0.0558 ms |
| p50 latency | 0.036 ms | 0.037 ms |
| p95 latency | 0.107 ms | 0.108 ms |
| p99 latency | 0.333 ms | 0.344 ms |
| Maximum latency | 27.89 ms | 205.75 ms |

## Observations
- All 2,656,971 HTTP requests succeeded during the 30-second test.
- No failed HTTP requests were recorded during the test that included the
  five-second direct-link outage.
- Throughput was approximately 2.13% lower than the healthy baseline.
- p99 latency increased from 0.333 ms to 0.344 ms.
- Maximum observed latency increased to 205.75 ms. This indicates that
  a temporary latency spike occurred even though no request failed.
- Both regional link interfaces were UP after restoration.
- The Singapore health endpoint responded successfully after the test.

## Interpretation
The benchmark demonstrates that the Singapore HTTP service remained
reachable throughout the tested link-failure scenario. This is consistent
with traffic being carried through the alternate Frankfurt path after
BGP convergence.

A separate BGP experiment measured approximately 138 ms for route
convergence during a Mumbai-Singapore link failure. That measurement is
not the same as HTTP recovery time and is reported separately.

The benchmark did not continuously capture route-table changes or
timestamp each request's path. Therefore, it does not independently prove
the exact instant at which traffic switched to Frankfurt.

## Limitations
- All three regions run inside one Colima Linux VM on one physical Mac.
- This is a network and service-failover simulation, not independent
  geographic infrastructure.
- The benchmark does not represent production internet latency or load.
- A zero request-failure count does not mean there was zero latency impact.

## Conclusion
ATLAS sustained a 100% HTTP request success rate during this controlled
five-second regional-link outage test. The observed maximum latency
increased, so the result demonstrates service continuity in this
simulation, not interruption-free failover.
