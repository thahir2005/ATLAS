# Experiment 08: Independent Monitoring Under Application Latency

## Objective

Verify that slow application health checks do not block BGP routing observations.

## Environment

- Three simulated regions: Mumbai, Singapore, Frankfurt
- Linux network namespaces inside Colima
- FRRouting BGP
- Go-based ATLAS recovery intelligence controller
- Application and routing observations collected concurrently

## Fault Injection

Applied artificial 800 ms network delay to Mumbai's loopback interface using Linux traffic control (`tc netem`).

The delay was temporary and removed after the experiment.

## Observed Results

### Application Health

| Region | HTTP status | Measured latency |
|---|---:|---:|
| Mumbai | 200 | 3217.373 ms |
| Singapore | 200 | 9.767 ms |
| Frankfurt | 200 | 10.018 ms |

All three applications remained healthy.

### BGP Observation Timeline

| Event | Timestamp (UTC) |
|---|---|
| Monitoring cycle started | 17:31:08.444132932 |
| Frankfurt BGP snapshot | 17:31:08.444907405 |
| Mumbai BGP snapshot | 17:31:08.450809618 |
| Singapore BGP snapshot | 17:31:08.451228187 |
| Monitoring cycle completed | 17:31:11.665559809 |

All three BGP observations occurred within approximately 7.1 ms of cycle start.

### Network Assessment

- Mumbai: HEALTHY
- Singapore: HEALTHY
- Frankfurt: HEALTHY
- All observed BGP peers established.
- Best paths and kernel routes installed.

## Result

PASS.

BGP observations were collected independently while the Mumbai application health check experienced approximately 3.2 seconds of latency.

The controller waited for both application and routing observations before publishing the combined assessment.

This validates the independent monitoring design under the tested latency condition.

## Cleanup

Verified that Mumbai's loopback traffic-control configuration returned to:

`qdisc noqueue 0: root refcnt 2`

## Limitations

- This was a single controlled latency experiment.
- The 800 ms netem configuration produced a measured HTTP latency of approximately 3.2 seconds; this is the observed result, not a guaranteed conversion factor.
- All regions still run inside one Colima VM on one physical Mac.
- This experiment validates monitoring concurrency, not physical multi-region resilience.
