# Experiment 01: Static Route Failure and Recovery

## Objective
Observe regional connectivity when the Mumbai-Singapore
network link is disconnected and restored.

## Environment
- Mumbai: 10.10.1.1
- Singapore: 10.10.2.1
- Frankfurt: 10.10.3.1
- Runtime: Colima Ubuntu 24.04 ARM64
- Networking: Linux network namespaces and veth pairs

## Failure Injection
Command:
ip -n atlas-mumbai link set ms0 down

## Observations
1. Mumbai-Singapore connectivity failed.
2. Mumbai-Frankfurt connectivity remained operational.
3. Singapore-Frankfurt connectivity remained operational.
4. Bringing ms0 back up did not restore the missing
   manually configured route.

## Recovery
The interface was brought up on both endpoints.
The regional routes were restored using ip route replace.

## Verification
- Mumbai to Singapore: 4/4 packets received.
- Singapore to Mumbai: 4/4 packets received.
- Packet loss after recovery: 0%.

## Engineering Findings
Interface state and routing state are separate.
Restoring an interface does not guarantee restoration
of manually configured routes.

## Limitation
This experiment used static routing and did not test
dynamic routing or automatic failover.
