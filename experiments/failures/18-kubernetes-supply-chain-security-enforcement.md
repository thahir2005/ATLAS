# Experiment 18 — Kubernetes Supply Chain Security Enforcement

## Objective
Enforce container image authenticity in Kubernetes using Kyverno ImageValidatingPolicy and Cosign keyless signatures.

## Environment
- Kubernetes: Kind v1.33.4
- Kyverno: v1.20.0
- Cosign: v3.1.6
- Container registry: GitHub Container Registry (GHCR)
- Namespace: atlas
- Trusted repository: thahir2005/ATLAS

## Implementation
1. Installed Kyverno using Helm.
2. Created an ImageValidatingPolicy at `infra/kubernetes/atlas-image-policy.yaml`.
3. Configured the policy to trust the GitHub Actions OIDC issuer:
   `https://token.actions.githubusercontent.com`
4. Restricted the signing identity to:
   `https://github.com/thahir2005/ATLAS/.github/workflows/publish-image.yml@refs/heads/feature/independent-monitoring`
5. Initially used Audit mode to inspect existing workloads.
6. Identified older unsigned image references in historical ReplicaSets.
7. Confirmed all three active regional Pods used the newly signed image.
8. Changed the policy to Deny with failurePolicy Fail.

## Test Results

### Test 1 — Unsigned image rejection
Image:
`ghcr.io/thahir2005/atlas-region-api:sha-957d9a8f1411971cc32adde4be054c4d409283b9`

Command used a Kubernetes server-side dry run to test admission.

Result: REJECTED.

Kyverno reported:
`failed to verify cosign signatures: no signatures found`

### Test 2 — Signed image acceptance
Image:
`ghcr.io/thahir2005/atlas-region-api:sha-015cdc1520e5708bb8f8b35384b16c099590148d`

Command used a Kubernetes server-side dry run.

Result: ACCEPTED.

Output:
`deployment.apps/atlas-signed-test created (server dry run)`

No Deployment was actually created by this dry-run test.

## Existing Workload Verification
All three active Pods continued running:
- Mumbai
- Singapore
- Frankfurt

All used the signed image tag associated with commit `015cdc1520e5708bb8f8b35384b16c099590148d`.

## Outcome
Both admission tests passed:
- Unsigned image rejected.
- Trusted signed image accepted.

Kyverno successfully enforced the configured Cosign identity during Kubernetes admission.

## Limitations
- Testing was performed on a local, single-node Kind cluster.
- The three regional workloads share one local development environment.
- Admission tests used server-side dry runs, not actual unsigned workload creation.
- The policy trusts a specific GitHub Actions workflow and branch identity.
- The policy currently covers the configured ATLAS image repository.
- Production deployment requires independent cluster failure domains, availability planning, registry access controls, and ongoing policy maintenance.

## Conclusion
This experiment demonstrated Kubernetes admission enforcement of container image signatures using Kyverno and Cosign keyless signing. Unsigned images were rejected, while images signed by the trusted ATLAS GitHub Actions workflow were accepted.
