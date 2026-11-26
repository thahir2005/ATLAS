# Experiment 03 — BGP Link Failover

## Objective

Validate that the ATLAS three-region network can maintain connectivity when
the direct Mumbai–Singapore network link fails.

## Topology

- Mumbai: AS 65001, loopback 10.10.1.1/32
- Singapore: AS 65002, loopback 10.10.2.1/32
- Frankfurt: AS 65003, loopback 10.10.3.1/32
- Routing: eBGP through FRRouting
- Topology: Full mesh with point-to-point links

## Failure injection

The Mumbai-side interface of the direct Mumbai–Singapore link was disabled:

    ip -n atlas-mumbai link set ms0 down

A continuous ICMP test ran from the Mumbai loopback (10.10.1.1) to the
Singapore loopback (10.10.2.1).

## Observed failover

After the direct link failed, BGP selected the alternative path through
Frankfurt.

Mumbai forward route:

    10.10.2.1 via 172.20.31.1 dev fm0

Singapore return route:

    10.10.1.1 via 172.20.23.2 dev sf0

Both directions converged in approximately 137.81 ms.

## Results

| Metric | Observed result |
| --- | ---: |
| Both-direction route convergence | 137.81 ms |
| ICMP packets transmitted | 112 |
| ICMP packets received | 111 |
| ICMP packet loss | 0.89% |
| Average ICMP RTT | 0.174 ms |
| Final connectivity after restoration | 5/5 packets received |

The ICMP test ran for approximately 12 seconds. One packet was lost during
the transition; traffic continued over the alternate path.

The 137.81 ms measurement is the observed time until both route lookups
showed the Frankfurt path. It is not an independently measured
application-level recovery time.

## Debugging observation

An earlier probe used the automatically selected transit-interface source
address. That test experienced 91.07% packet loss because the return path
was not suitable for that source address.

Repeating the experiment with the Mumbai regional loopback as the explicit
ping source provided a valid end-to-end test and confirmed bidirectional
failover.

This demonstrates why routing-table convergence alone is insufficient:
the source address and return route must also be validated.

## Recovery

Both ends of the direct link were brought back up. BGP re-established the
direct peering, and the preferred direct route returned.

A final five-packet connectivity test succeeded with zero packet loss.

## Conclusion

The three-region BGP topology successfully rerouted traffic through
Frankfurt after the direct Mumbai–Singapore link failed.

The experiment demonstrates dynamic route convergence and alternate-path
connectivity in the simulated environment.

## Limitations

All three regions run as Linux network namespaces inside one Colima VM on
one physical Mac. This experiment validates routing behavior, not
independent geographic, host, power, or data-center failure resilience.

The ICMP measurements are laboratory observations and should not be
interpreted as production service-level guarantees.
