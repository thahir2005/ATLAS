# Experiment 05: Autonomous Application Failure Detection

## Objective

Verify that the ATLAS Go control plane can detect a regional
application failure and automatically detect its recovery.

## Environment

- Three regional Linux network namespaces
- Mumbai, Singapore and Frankfurt
- Regional Go HTTP API on port 8080
- ATLAS Go health-monitoring controller
- Monitoring interval: 1 second
- HTTP timeout: 1 second
- Controller executed inside atlas-mumbai
- Read-only monitoring mode

## Failure injection

Only the Singapore regional API process was stopped.

The following components were not intentionally modified:
- FRRouting processes
- BGP configurations
- Linux network interfaces
- Regional network topology
- Mumbai and Frankfurt API processes

## Observed timeline (UTC)

| Timestamp | Observation |
|---|---|
| 16:42:21.281 | All three regions initialized as healthy |
| 16:42:24.292 | Singapore last observed healthy before failure detection |
| 16:42:25.296 | Singapore observed unhealthy |
| 16:42:26.302 | Singapore remained unhealthy |
| 16:42:27.308 | Singapore remained unhealthy |
| 16:42:28.310 | Singapore remained unhealthy |
| 16:42:29.316 | Singapore observed healthy again |

The exact API termination and restart timestamps were not
recorded. Therefore, exact stop-to-detection and
restart-to-recovery durations cannot be calculated.

## Failure event

Event: REGION_STATE_CHANGED

Region: Singapore

Previous status: healthy

Current status: unhealthy

Reason: connection refused

At the first unhealthy observation:
- Healthy regions: 2
- Total regions: 3
- Overall controller status: degraded

Mumbai and Frankfurt remained healthy.

## Recovery event

Event: REGION_STATE_CHANGED

Region: Singapore

Previous status: unhealthy

Current status: healthy

The controller returned to an overall healthy state with
3 of 3 regional services healthy.

The final Singapore HTTP health check returned HTTP 200.

## Results

- Initial state detection: successful
- Singapore application failure detection: successful
- Persistent unhealthy state reporting: successful
- Singapore recovery detection: successful
- Graceful controller shutdown: successful
- Final Singapore health check: successful

## Limitations

- A single failed health check immediately changes the state
  to unhealthy.
- No consecutive-failure threshold or hysteresis is implemented.
- Exact failure injection and restart timestamps were not recorded.
- This experiment tests application health, not BGP failover.
- The controller runs inside the Mumbai namespace.
- All regions are simulated inside one Colima VM on one physical Mac.

## Conclusion

ATLAS successfully detected the Singapore application failure,
reported the regional system as degraded, and detected recovery
after the service restarted.

This validates the first application-level failure detection
workflow in the ATLAS Go control plane. Automated traffic
failover has not yet been implemented.
