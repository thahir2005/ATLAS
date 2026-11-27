# Experiment 19: HashiCorp Vault and Kubernetes Secrets Management

## Objective
Deploy HashiCorp Vault in Kubernetes and demonstrate secure secret storage, Kubernetes authentication, and least-privilege secret access for ATLAS workloads.

## Environment
- Kubernetes: Kind cluster (atlas-k8s)
- Vault: 2.0.4
- Deployment method: HashiCorp Helm chart
- Namespace: vault
- ATLAS namespace: atlas
- Secret engine: KV v2
- Authentication: Kubernetes auth method

## Architecture
1. Vault runs in the dedicated `vault` namespace.
2. KV v2 is enabled at the `atlas/` path.
3. A test database credential is stored at `atlas/database`.
4. Vault uses Kubernetes service-account JWT authentication.
5. The Vault service account receives the Kubernetes `system:auth-delegator` role.
6. A Vault policy named `atlas-read` grants read access only to `atlas/data/database`.
7. A Kubernetes auth role named `atlas-api` binds the `default` service account in namespace `atlas`.
8. The role requires the JWT audience `vault` and issues tokens with a one-hour TTL.

## Implementation

### 1. Vault deployment
Installed Vault using the HashiCorp Helm chart in development mode.

### 2. Secret engine
Enabled KV v2 at `atlas/`.

### 3. Secret storage
Stored a lab-only database credential:
- Username: atlas_app
- Password: dummy lab value
- Environment: development

Verified successful secret retrieval without printing the password.

### 4. Kubernetes authentication
- Enabled Vault Kubernetes authentication.
- Configured the Kubernetes API endpoint and CA certificate.
- Created the `vault-token-review` ClusterRoleBinding.
- Configured service-account JWT validation.

### 5. Access policy
Created `atlas-read` with read-only access to:
`atlas/data/database`

Created role `atlas-api` with:
- Service account: default
- Namespace: atlas
- Audience: vault
- Policy: atlas-read
- Token TTL: 1 hour

## Validation Results

| Test | Result |
|---|---|
| Vault pod readiness | PASS |
| Vault initialized and unsealed | PASS |
| KV v2 enabled | PASS |
| Secret write and retrieval | PASS |
| Kubernetes authentication | PASS |
| Database secret read permission | PASS |
| Unrelated secret access denied | PASS |
| Vault-specific JWT audience | PASS |
| GitOps policy enforcement | PASS |
| ATLAS regional pods healthy | PASS |

## Security Findings
- Kubernetes authentication successfully issued a Vault client token.
- The ATLAS service account could read the permitted database secret.
- Access to an unrelated secret path returned `deny`.
- The Kubernetes JWT audience was explicitly restricted to `vault`.
- Vault's development root token was used only for this local experiment.

## Limitations
- Vault was deployed in development mode.
- Storage type was in-memory; data will not survive a Vault restart.
- The root token was used for administration.
- The Vault Agent Injector was disabled.
- Existing ATLAS API deployments were not modified to consume Vault secrets.
- No production-grade HA, persistent storage, auto-unseal, or backup configuration was implemented.
- The Kubernetes authentication and secret retrieval workflow was tested independently of the running application.

This experiment validates the Vault-to-Kubernetes authentication and authorization workflow. It does not establish production-ready secrets management.

## Conclusion
Successfully deployed HashiCorp Vault, configured KV v2 secret storage, integrated Kubernetes service-account authentication, and enforced least-privilege access for an ATLAS service account. Secret retrieval and access-denial tests passed.
