# Experiment 09 — etcd Consensus and Leader Failover

## Objective

Deploy a three-member etcd cluster across the ATLAS regional network
namespaces and verify Raft leader election, quorum availability,
replicated writes, leader failure, and member recovery.

## Environment

- etcd version: 3.4.30
- Runtime: Colima Linux VM
- Architecture: ARM64
- Regional namespaces:
  - atlas-mumbai
  - atlas-singapore
  - atlas-frankfurt
- Client port: 2379
- Peer port: 2380
- Raft cluster token: atlas-consensus-v1

All three members run inside one Colima VM on one physical Mac.
This experiment demonstrates logical regional separation, not
independent physical-region fault tolerance.

## Initial cluster

| Member | Client IP | Peer IP | Initial role |
|---|---|---|---|
| Mumbai | 10.10.1.1:2379 | 10.10.1.1:2380 | Leader |
| Singapore | 10.10.2.1:2379 | 10.10.2.1:2380 | Follower |
| Frankfurt | 10.10.3.1:2379 | 10.10.3.1:2380 | Follower |

All three members started successfully.

Initial Raft election:
- Leader: Mumbai
- Raft term: 2
- All three members reported the same applied index.

## Test 1 — Cluster health and membership

Command:
`etcdctl endpoint health`

Result:
- Mumbai: healthy
- Singapore: healthy
- Frankfurt: healthy
- Three started voting members were listed.

## Test 2 — Replicated write

A key was written through Mumbai:

`/atlas/consensus/test = first-quorum-write`

The value was successfully read through:
- Singapore
- Frankfurt

Result: PASS

## Test 3 — Leader failure

The Mumbai etcd systemd service was stopped.

The regional namespace, BGP routing process, and HTTP service
were left running.

After the failure:
- Mumbai etcd: inactive
- Singapore: healthy
- Frankfurt: healthy
- New leader: Frankfurt
- Raft term increased from 2 to 3.
- Two-member quorum remained available.

## Test 4 — Write during leader failure

A new key was written through Singapore:

`/atlas/consensus/failover-test = leader-failover-success`

The value was successfully read through Frankfurt while Mumbai
was stopped.

Result: PASS

## Test 5 — Member recovery

The existing Mumbai etcd service was restarted without deleting
or recreating its data directory.

After recovery:
- All three endpoints reported healthy.
- All three members appeared in cluster membership.
- Frankfurt remained leader at Raft term 3.
- All three members reported Raft index 12.
- All three members reported applied index 12.
- Mumbai successfully read the failover-test key.

Mumbai logs confirmed peer connections were re-established and
the member recognized Frankfurt as leader.

Result: PASS

## Quorum analysis

For three voting members:

Quorum = floor(3 / 2) + 1 = 2

Therefore:
- Three available members: quorum available.
- Two available members: quorum available.
- One available member: quorum unavailable.

The experiment demonstrated that the cluster continued committing
writes after losing its original leader because two voting members
remained available.

## Security and deployment limitations

- Client and peer communication currently use plain HTTP.
- TLS, authentication, and authorization are not configured.
- All members share one physical VM and host.
- This is a controlled lab simulation, not production multi-region
  disaster recovery.

## Conclusion

The ATLAS etcd cluster successfully demonstrated initial Raft
leader election, three-member quorum, replicated writes, leader
failure, replacement leader election, continued writes with two
members, and recovery of the stopped member.

The next milestone is distributed leases, leadership ownership,
and fencing to prevent stale controllers from performing unsafe
writes during failover.
