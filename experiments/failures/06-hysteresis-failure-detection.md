# Experiment 06: Failure Detection Hysteresis

## Objective

Validate configurable consecutive-failure and recovery thresholds
using a real regional application failure.

## Configuration

- Regions: Mumbai, Singapore, Frankfurt
- Monitoring interval: 1 second
- HTTP timeout: 1 second
- Failure threshold: 3 consecutive failures
- Recovery threshold: 2 consecutive successes
- Controller location: atlas-mumbai
- Failure injected: Singapore regional API process stopped
- Recovery: Singapore regional API restarted

## Failure detection observations

| Cycle | Consecutive failures | Singapore confirmed state | Overall status |
|---|---:|---|---|
| 1-3 | 0 | Healthy | Healthy |
| 4 | 1 | Healthy; failure suspected | Healthy |
| 5 | 2 | Healthy | Healthy |
| 6 | 3 | Unhealthy | Degraded |

At 2026-11-26T16:50:35.800413102Z, the controller emitted
REGION_FAILURE_SUSPECTED after the first failed check.

At 2026-11-26T16:50:37.809184060Z, the controller emitted
REGION_STATE_CHANGED after three consecutive failures.

The failure reason was connection refused on the Singapore
health endpoint.

Mumbai and Frankfurt remained healthy.

## Recovery observations

| Cycle | Consecutive successes | Singapore confirmed state | Overall status |
|---|---:|---|---|
| 9 | 1 | Unhealthy; recovery suspected | Degraded |
| 10 | 2 | Healthy | Healthy |
| 11-12 | — | Healthy | Healthy |

At 2026-11-26T16:50:40.822382705Z, the controller emitted
REGION_RECOVERY_SUSPECTED after the first successful check.

At 2026-11-26T16:50:41.828319796Z, the controller emitted
REGION_STATE_CHANGED after two consecutive successful checks.

The final Singapore health check returned HTTP 200.

## Results

- Initial regional state detection: successful
- First failed check did not mark Singapore unhealthy: successful
- Second failed check did not mark Singapore unhealthy: successful
- Third failed check marked Singapore unhealthy: successful
- First recovery success did not mark Singapore healthy: successful
- Second recovery success restored Singapore: successful
- Overall degraded and healthy states: correctly reported
- Final regional health check: successful
- Total monitoring cycles: 12

## Limitations

- Exact API termination and restart timestamps were not recorded.
- This experiment validates application-level health detection.
- It does not implement automated traffic failover.
- The controller runs inside the Mumbai namespace.
- All three simulated regions run inside one Colima VM on one
  physical Mac, so they do not provide physical or geographic
  failure independence.

## Conclusion

ATLAS successfully applied a three-failure threshold and a
two-success recovery threshold during a controlled Singapore
application failure.

The controller avoided declaring a regional failure after
transient failed checks and avoided declaring recovery after
only one successful check.

This validates the hysteresis state machine against a real
regional service failure.
