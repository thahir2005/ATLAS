# Experiment 07: Live Network Failure and Recovery Intelligence

## Objective
Validate whether ATLAS can observe a regional network failure,
identify an alternate BGP route, classify network degradation,
and confirm recovery after link restoration.

## Environment
- Mumbai: AS65001
- Singapore: AS65002
- Frankfurt: AS65003
- Routing: FRRouting eBGP
- Controller: Native Go ATLAS intelligence controller
- Monitoring interval: 1 second
- Application health timeout: 3 seconds
- Health failure threshold: 3 consecutive failures
- Health recovery threshold: 2 consecutive successes

## Failure Injection
The Mumbai-Singapore veth link was taken down by disabling
Mumbai's ms0 interface.

Failure injection: 2026-11-26 17:13:16.084 UTC
Link restored: 2026-11-26 17:13:22.109 UTC
Observed link-down duration: approximately 6.025 seconds.

## Baseline
All three regional application health endpoints returned HTTP 200.
Mumbai's route to Singapore used the direct path:

10.10.2.1 via 172.20.12.2 dev ms0

Both Mumbai BGP peers were Established.

## Failure Observations
During the failure:
- Mumbai's direct BGP peer 172.20.12.2 entered Active.
- Singapore's corresponding peer 172.20.12.1 entered Active.
- Mumbai retained its Frankfurt peer 172.20.31.1 as Established.
- The best path to Singapore changed to AS path 65003 65002.
- The kernel route changed to:

10.10.2.1 via 172.20.31.1 dev fm0

ATLAS classified the Mumbai and Singapore network observations
as NETWORK_DEGRADED.

The Singapore HTTP health request timed out. This indicates
loss of reachability during the disruption, not proof that the
Singapore application process stopped.

## Recovery Observations
After restoring Mumbai ms0:
- The direct BGP peer returned to Established.
- The best path returned to peer 172.20.12.2.
- The kernel route returned to interface ms0.
- Singapore health checks succeeded again.
- The health tracker emitted REGION_RECOVERY_SUSPECTED after
  the first successful check.
- It confirmed healthy after the second consecutive success.

Important observed timestamps:
- 17:13:19.157 UTC: alternate route observed through Frankfurt.
- 17:13:22.109 UTC: direct link restored.
- 17:13:25.349 UTC: direct path observed again.
- 17:13:25.439 UTC: first successful Singapore health check.
- 17:13:26.153 UTC: Singapore recovery confirmed.

## Results
- BGP alternate path: demonstrated.
- Kernel route change: demonstrated.
- Network degradation classification: demonstrated.
- Direct path restoration: demonstrated.
- Application health recovery hysteresis: demonstrated.
- Final state: all three regions healthy.

## Measurement Limitations
The first post-failure BGP observation occurred after the
monitoring cycle was delayed by the 3-second HTTP timeout.
Therefore, the timestamps above are observation timestamps,
not precise BGP convergence measurements.

This experiment does not establish a precise RTO or RPO.
A dedicated high-frequency routing observer and request-level
traffic benchmark are required for those measurements.

## Infrastructure Limitation
All three regional namespaces run inside one Colima Linux VM
on one physical Mac. This is a controlled regional simulation,
not three geographically independent failure domains.

## Raw Evidence
See 07-live-network-failure.jsonl.
