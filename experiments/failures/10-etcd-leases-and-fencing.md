# Experiment 10 — etcd Leases, Leadership and Fencing

## Objective

Implement lease-backed leadership, monotonically increasing fencing
tokens, and atomic protection against stale controller writes.

## Architecture

- Three etcd members representing Mumbai, Singapore and Frankfurt.
- Go consensus manager using etcd client v3.5.17.
- Lease-backed leadership sessions with a 10-second TTL.
- etcd concurrency mutex for exclusive leadership.
- Mutex acquisition revision used as the fencing token.
- Fencing token registered once during leadership acquisition.
- Protected writes atomically verify mutex ownership and that the
  registered fencing token still matches the controller's token.

## Experiment A — Repeated writes by one leader

Mumbai acquired leadership with token 15.

Results:
- First write accepted.
- Second write accepted using the same token.
- Final resource value: `mumbai-second-write`.
- Final fencing token: `00000000000000000015`.

This verifies that a legitimate leader can perform multiple writes
without incorrectly advancing its fencing token on every write.

## Experiment B — Lease revocation and stale-writer rejection

Date: 26 November 2026

| Timestamp (local) | Event |
|---|---|
| 23:43:37.208 | Mumbai acquired leadership, token 20 |
| 23:43:37.208 | Mumbai first write accepted |
| 23:43:41.697 | Singapore began acquiring leadership |
| 23:43:41.702 | Singapore acquired leadership, token 24 |
| 23:43:41.703 | Singapore first write accepted |
| 23:44:07.225 | Mumbai attempted its delayed second write |
| 23:44:07.225 | Mumbai stale write rejected |
| 23:44:26.703 | Singapore completed its leadership hold |

Mumbai's lease was explicitly revoked using its hexadecimal lease ID.
Singapore subsequently acquired leadership with a higher fencing token.

Mumbai's process remained alive and attempted its delayed write.
The write was rejected because it no longer owned the mutex and its
fencing token was no longer current.

## Final verification

- Mumbai fencing token: 20
- Singapore fencing token: 24
- Mumbai stale write: rejected
- Final fencing token in etcd: `00000000000000000024`
- Final protected resource value: `singapore-new-leader`

The final value remained unchanged after the stale-write attempt.

## Implementation safety

The protected etcd transaction checks both:
1. The controller still owns its lease-backed mutex.
2. The stored fencing token equals the controller's token.

Leadership acquisition registers a strictly newer token. Repeated
writes by that leader reuse the registered token.

This protection applies to writes made through this etcd-backed
protected-write implementation. External resources such as PostgreSQL
must independently enforce fencing tokens before they can claim the
same protection.

## Limitations

All three etcd members and regional namespaces run inside one Colima
VM on one physical Mac. This is a regional infrastructure simulation,
not a deployment across independent geographic failure domains.

The experiment demonstrates lease revocation, leader acquisition,
monotonic fencing tokens, and stale-writer rejection. It does not
demonstrate protection of external databases.

