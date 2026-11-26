# ADR 0002: Dynamic Regional Routing with eBGP

## Status
Accepted

## Context
ATLAS requires dynamic routing between three simulated regions:
Mumbai, Singapore, and Frankfurt.

Static routes are insufficient for automatic path selection
when a regional network link fails.

## Decision
Use FRRouting (FRR) with external BGP (eBGP).

Each region operates in an independent Linux network namespace
and has its own FRR pathspace.

| Region | Autonomous System | Router ID |
|---|---:|---|
| Mumbai | 65001 | 10.10.1.1 |
| Singapore | 65002 | 10.10.2.1 |
| Frankfurt | 65003 | 10.10.3.1 |

The three routers use a full-mesh topology.

Each router advertises its own /32 loopback service prefix.

## Implementation
- FRRouting 8.4.4
- Zebra for kernel route installation
- BGP for dynamic route exchange
- Three regional FRR instances
- Six eBGP neighbor relationships

## Verification
- All six BGP peer relationships established.
- Each region learned the other two regional prefixes.
- BGP routes installed in Linux kernel routing tables.
- Six cross-region ping tests passed.
- All tests reported 0% packet loss.

## Limitations
All three regions run inside one Colima Linux VM on one
physical Mac. This is a network simulation, not independent
geographic or physical infrastructure.
