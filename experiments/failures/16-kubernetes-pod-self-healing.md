# Experiment 16: Kubernetes Pod Self-Healing

## Objective
Verify that Kubernetes automatically replaces a deleted application pod and restores the desired replica count.

## Environment
- Kubernetes: Kind
- Cluster: atlas-k8s
- Namespace: atlas
- Deployment: frankfurt-api
- Desired replicas: 1

## Procedure
1. Checked the Frankfurt API pod and deployment status.
2. Deleted the running Frankfurt API pod manually.
3. Watched Kubernetes create a replacement pod.
4. Verified the replacement became Ready.
5. Checked deployment availability and Argo CD health.

## Observations
- Original pod: frankfurt-api-8c8c8d979-9sg2t
- Replacement pod: frankfurt-api-8c8c8d979-x4twj
- Replacement pod reached 1/1 Ready in approximately 7 seconds.
- Deployment returned to 1/1 Available.
- Argo CD remained Synced and Healthy.

## Result
Kubernetes successfully replaced the deleted Frankfurt API pod and restored the desired replica count automatically.

## Limitations
This experiment validates pod-level recovery on a single-node Kind cluster. It does not validate node failure, independent regional failure, or cross-region failover.
