# ATLAS Benchmark 01 — Healthy Network Baseline

## Configuration

- Target: Singapore Regional API
- Source: Mumbai loopback (10.10.1.1)
- Destination: 10.10.2.1:8080
- Duration: 30 seconds
- Concurrent workers: 5
- Request timeout: 2 seconds

## Results

| Metric | Result |
|---|---:|
| Total requests | 2,714,898 |
| Successful requests | 2,714,898 |
| Failed requests | 0 |
| Success rate | 100% |
| Average latency | 0.0546 ms |
| p50 latency | 0.036 ms |
| p95 latency | 0.107 ms |
| p99 latency | 0.333 ms |
| Minimum latency | 0.013 ms |
| Maximum latency | 27.89 ms |
| Throughput | Approximately 90,493 requests/sec |

## Conclusion

The Singapore regional API handled the 30-second benchmark
without failed requests under healthy network conditions.

The results establish a local simulation baseline for comparison
with subsequent network-failure experiments.

## Limitations

All regions execute inside one Colima VM on one physical Mac.
The measurements represent virtual networking and local
application performance, not real geographic network latency.
