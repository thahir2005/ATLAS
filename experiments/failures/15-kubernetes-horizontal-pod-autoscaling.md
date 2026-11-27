# Experiment 15 — Kubernetes Horizontal Pod Autoscaling

## Objective

Demonstrate Kubernetes CPU-based Horizontal Pod Autoscaling (HPA) for the ATLAS Mumbai regional API, including automatic scale-up and scale-down.

## Environment

- Kubernetes: Kind v1.33.4
- Cluster: atlas-k8s
- Namespace: atlas
- Workload: mumbai-api Deployment
- Metrics: Kubernetes Metrics Server
- Container image: ghcr.io/thahir2005/atlas-region-api:sha-1b058c5

## Configuration

- Minimum replicas: 1
- Maximum replicas: 3
- CPU utilization target: 50% of requested CPU
- CPU request: 50m
- CPU limit: 250m
- Memory request: 32Mi
- Memory limit: 128Mi

## Procedure

1. Installed Metrics Server and enabled kubelet metric collection for the local Kind lab.
2. Verified node and pod CPU and memory metrics using kubectl top.
3. Configured resource requests and limits for all three regional API Deployments.
4. Created an HPA targeting the Mumbai API Deployment.
5. Generated concurrent HTTP requests against the Mumbai health endpoint using a temporary load-generator pod.
6. Observed HPA CPU utilization and replica changes.
7. Deleted the temporary load-generator pod.
8. Observed the HPA scale the Mumbai Deployment back down.

## Observations

| Stage | Observed CPU | Mumbai replicas |
|---|---:|---:|
| Initial idle state | 2% | 1 |
| Load increased | 114% | 1 |
| Peak observed | 320% | 3 |
| Sustained load | Approximately 124–140% | 3 |
| After load removal | 2% | 1 |

The HPA reached its configured maximum of three replicas. Its ScalingLimited condition indicated that the desired replica count exceeded the configured maximum.

After load removal, the HPA reported:

- Desired replicas: 1
- Current CPU utilization: 2%
- Event: New size: 1; reason: All metrics below target

The temporary load-generator pod was deleted.

## Final Result

The experiment successfully demonstrated:

- Metrics Server integration.
- CPU-based HPA metric collection.
- Automatic scale-up from one to three replicas.
- Enforcement of the maximum replica count.
- Automatic scale-down from three to one replica.

Final cluster state:

- Mumbai API: 1/1
- Singapore API: 1/1
- Frankfurt API: 1/1
- Mumbai HPA: 1–3 replicas, currently 1
- Temporary load generator: removed

## Limitations

This experiment used a single-node local Kind cluster running inside Colima. The three regional APIs are simulated workloads on one local Kubernetes node, not independent physical regions.

The test demonstrates application pod autoscaling. It does not demonstrate node autoscaling, geographic failover, or independent regional infrastructure scaling.

The Metrics Server kubelet TLS verification workaround used in this local lab is not a production security configuration.

## Conclusion

ATLAS successfully demonstrated Kubernetes Horizontal Pod Autoscaling using live CPU metrics. The Mumbai API scaled up under load and returned to its minimum replica count after the load was removed.
