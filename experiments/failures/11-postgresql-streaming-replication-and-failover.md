# Experiment 11: PostgreSQL Streaming Replication and Regional Failover

## Objective

Implement PostgreSQL physical streaming replication across three simulated
regions and perform a controlled primary failover from Mumbai to Singapore.

## Architecture

- Singapore: New PostgreSQL primary
- Frankfurt: PostgreSQL standby
- Mumbai: PostgreSQL standby
- PostgreSQL version: 16.15
- Replication mode: Asynchronous physical streaming replication
- Replication transport: Regional Linux network namespaces

All three database instances run inside one Colima VM on one physical Mac.
They are simulated regions, not independent geographic failure domains.

## Initial topology

Mumbai was the primary.

Singapore and Frankfurt were streaming standbys.

Both replicas used physical replication slots. Before failover, the primary
reported both connections as streaming and asynchronous.

## Pre-failover checkpoint

The primary flush LSN was:

0/50003A8

Both standby replay LSNs matched:

- Singapore: 0/50003A8
- Frankfurt: 0/50003A8

The checkpoint transaction was visible on all three databases.

## Failure and promotion

Failure start:
2026-11-27T00:01:26.009505533+05:30

Mumbai PostgreSQL was stopped using pg_ctl with fast shutdown.

The process was confirmed stopped and port 5432 was confirmed closed.

Singapore was promoted using pg_ctl promote.

Promotion completed:
2026-11-27T00:02:01.941792187+05:30

The first successful post-promotion write was recorded at:
2026-11-27 18:32:01.904001+00

The measured interval from Mumbai shutdown to successful write on Singapore
was approximately 35.9 seconds.

## Frankfurt reconfiguration

Frankfurt was reconfigured to use Singapore as its upstream.

A new physical replication slot was created on Singapore.

Frankfurt reconnected and reported:

- Recovery mode: true
- WAL receiver status: streaming
- Sender: 10.10.2.1
- Slot: atlas_frankfurt_slot_v2

## Mumbai re-seeding

The former Mumbai primary was not restarted as a writable database.

Its original data directory was preserved at:

/var/lib/postgresql/atlas/mumbai-former-primary

A fresh physical backup was taken from Singapore using pg_basebackup.

Mumbai was configured with standby.signal and a new replication slot.

After correcting the new data directory permissions to 0700, Mumbai started
successfully as a standby.

Mumbai reported:

- Recovery mode: true
- WAL receiver status: streaming
- Sender: 10.10.2.1
- Slot: atlas_mumbai_slot_v2

## Final verification

Singapore:

- pg_is_in_recovery(): false
- Writable primary
- Flush LSN: 0/7000060

Frankfurt:

- pg_is_in_recovery(): true
- WAL receiver streaming
- Replay LSN: 0/7000060

Mumbai:

- pg_is_in_recovery(): true
- WAL receiver streaming
- Replay LSN: 0/7000060

Singapore reported both replicas as streaming and asynchronous.

All three databases contained the same four replication_probe records:

1. Mumbai-to-Singapore replication successful
2. Three-region replication verification
3. ATLAS_FAILOVER_CHECKPOINT_2026
4. SINGAPORE_PROMOTED_TO_PRIMARY

The post-promotion transaction was verified on both standbys.

## Results

- Controlled primary shutdown succeeded.
- Singapore promotion succeeded.
- Singapore accepted writes after promotion.
- Frankfurt successfully changed upstream to Singapore.
- Mumbai was safely re-seeded from Singapore.
- Both replicas resumed streaming.
- Both replay LSNs matched the new primary's flush LSN at final verification.
- The observed failover interval to a successful write was approximately
  35.9 seconds.
- Both replicas were confirmed streaming by 00:04:53 IST.

The exact standby reconnection timestamp was not captured.

## RPO and RTO interpretation

The pre-failover checkpoint showed both standby replay LSNs equal to the
old primary's flush LSN. This demonstrates that the recorded checkpoint
was replicated before shutdown.

It does not establish a general zero-RPO guarantee.

Replication is asynchronous. Transactions not replayed before a failure
could be lost.

The measured 35.9-second interval is specific to this controlled experiment
and is not a production RTO guarantee.

## Safety limitations

- etcd fencing currently protects etcd KV writes only.
- PostgreSQL does not yet enforce the etcd fencing token at its own
  write boundary.
- This experiment prevented simultaneous writable primaries by explicitly
  stopping the old primary before promotion.
- Production-grade split-brain protection requires resource-side fencing
  and independent failure domains.
- Lab credentials are not suitable for production.

## Conclusion

ATLAS successfully demonstrated PostgreSQL streaming replication,
controlled regional primary promotion, standby reconfiguration, safe
re-seeding of the former primary, and post-failover data verification
across three simulated regions.
