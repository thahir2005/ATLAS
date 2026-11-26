# Experiment 14: ATLAS Recovery Decision Validation

## Objective
Validate ATLAS recovery decisions using application, BGP and database health observations.

## Baseline
All three regions reported:
- Application: HEALTHY
- BGP: 2/2 peers established
- Database: HEALTHY
- Recovery decision: CONTINUE_MONITORING

## Application Failure Simulation
The Mumbai regional API process was suspended using SIGSTOP.

Observed decision before failure detection:
- CONTINUE_MONITORING
- HEALTHY
- Database status: HEALTHY

Observed decision after detection:
- Action: INVESTIGATE_APPLICATION
- Classification: APPLICATION_DEGRADED
- Database status: HEALTHY

This confirms ATLAS distinguished application degradation from database unavailability.

## Recovery
The Mumbai API process was resumed using SIGCONT.

The health endpoint returned:
- Service: ATLAS Regional API
- Region: Mumbai
- Status: healthy

The recovery decision returned to:
- CONTINUE_MONITORING
- HEALTHY
- Database status: HEALTHY

Singapore and Frankfurt remained healthy throughout the experiment.

## Final Result

| Region | Application | BGP peers | Database | Decision |
|---|---|---|---|---|
| Mumbai | Healthy | 2/2 | Healthy | CONTINUE_MONITORING |
| Singapore | Healthy | 2/2 | Healthy | CONTINUE_MONITORING |
| Frankfurt | Healthy | 2/2 | Healthy | CONTINUE_MONITORING |

PASS — ATLAS detected the simulated Mumbai application degradation,
selected INVESTIGATE_APPLICATION, and returned to CONTINUE_MONITORING
after recovery.

## Limitations
- This experiment used SIGSTOP/SIGCONT, not a physical host failure.
- It validated application degradation and recovery decisions.
- It did not validate automatic failover.
- All simulated regions share one Colima VM and physical host.
- Automatic failover remains disabled.

## Conclusion
ATLAS successfully combined application and database health observations
to produce a recovery decision and recognize recovery.
