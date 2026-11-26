# Experiment 02: BGP Dynamic Routing Baseline

## Objective
Verify dynamic route exchange and cross-region connectivity
using eBGP instead of manually installed static routes.

## Topology
- Mumbai: AS65001
- Singapore: AS65002
- Frankfurt: AS65003
- Full-mesh eBGP peering

## Procedure
1. Start the three regional FRR instances.
2. Establish all six eBGP neighbor relationships.
3. Advertise each regional loopback /32 prefix.
4. Verify BGP routing tables.
5. Remove the manually installed static host routes.
6. Verify BGP routes in Linux kernel routing tables.
7. Test connectivity between all regional pairs.

## Results
- All six BGP peer relationships established.
- Each router learned two remote service prefixes.
- Kernel routes showed proto bgp.
- All six connectivity tests passed.
- Packet loss: 0%.

## Conclusion
Dynamic routing is operational across all three simulated
regions. The network is ready for link-failure and
route-convergence experiments.

## Limitation
The experiment runs in one Linux VM on one physical Mac.
It does not measure real geographic network latency.
