# Experiment 07: Native Go BGP Event Detection

## Objective

Validate that the ATLAS Go BGP observer detects regional peer failures,
BGP best-path changes, Linux kernel route changes, and recovery.

## Environment

- Source region: Mumbai
- Destination region: Singapore
- Alternate route: Frankfurt
- BGP ASNs: Mumbai 65001, Singapore 65002, Frankfurt 65003
- Observer: atlas-bgp-go
- Polling interval: 200ms
- Failure injection: Mumbai-Singapore interface ms0 disabled
- Recovery: Mumbai-Singapore interface ms0 restored

All three regions run as Linux network namespaces inside one Colima VM
on one physical Mac. This is a network-failure simulation, not independent
geographic infrastructure.

## Observed timeline (UTC)

| Time | Event | Observation |
|---|---|---|
| 16:58:29.071 | INITIAL_STATE | Direct route via ms0 |
| 16:58:32.115 | PEER_STATE_CHANGE | Singapore peer Established -> Idle |
| 16:58:32.115 | BGP_BESTPATH_CHANGE | Best path switched to Frankfurt |
| 16:58:32.115 | KERNEL_ROUTE_CHANGE | Kernel route switched to fm0 |
| 16:58:33.131 | PEER_STATE_CHANGE | Peer Idle -> Active |
| 16:58:37.118 | PEER_STATE_CHANGE | Peer Active -> Idle |
| 16:58:41.137 | PEER_STATE_CHANGE | Peer Idle -> Established |
| 16:58:42.334 | BGP_BESTPATH_CHANGE | Best path returned to Singapore |
| 16:58:42.334 | KERNEL_ROUTE_CHANGE | Kernel route returned to ms0 |

## Route transitions

### Initial route

- Next hop: 172.20.12.2
- Interface: ms0
- Source: 172.20.12.1
- AS path: 65002

### Failover route

- Next hop: 172.20.31.1
- Interface: fm0
- Source: 172.20.31.2
- AS path: 65003 65002

### Recovered route

- Next hop: 172.20.12.2
- Interface: ms0
- Source: 172.20.12.1
- AS path: 65002

## Measurements and interpretation

- The first peer, best-path, and kernel-route changes were observed
  in the same polling cycle.
- The observer polling interval was configured to 200ms.
- The direct BGP peer was observed Established again at 16:58:41.137.
- The direct best path and kernel route were observed restored at
  16:58:42.334.
- The difference between peer recovery and route restoration was
  approximately 1.197 seconds.
- The link restoration was requested at approximately 16:58:37 UTC.
- The direct route was observed restored approximately 5.334 seconds
  after that request.

The shared timestamp of the initial failover events does not prove
zero-millisecond convergence. The observer detected all three changes
within one polling cycle, so finer-grained timing requires additional
instrumentation.

## Result

PASS.

The native Go observer detected:
- BGP peer state transitions.
- Alternate BGP best-path selection.
- Linux kernel route failover.
- Direct peer recovery.
- Best-path and kernel-route restoration.

The experiment validates read-only BGP event detection. It does not
yet demonstrate autonomous recovery, independent geographic failure
domains, or end-to-end HTTP recovery.
