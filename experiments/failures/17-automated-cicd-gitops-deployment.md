# Experiment 17: Automated CI/CD and GitOps Deployment

## Objective
Implement and verify an automated CI/CD pipeline that builds, publishes, and deploys regional API updates through GitHub Actions, GHCR, Kubernetes, and Argo CD.

## Technology Stack
- GitHub Actions
- Go testing and race detector
- Docker Buildx
- GitHub Container Registry (GHCR)
- Kubernetes (Kind)
- Argo CD

## Procedure
1. Added a `/version` endpoint to the regional API.
2. Pushed the application change to GitHub.
3. Executed Go tests and race detection through GitHub Actions.
4. Built and published a multi-architecture Docker image.
5. Updated Kubernetes image references in Git through the publishing workflow.
6. Configured image tags to use full Git commit SHA values.
7. Verified the published image could be pulled from GHCR.
8. Allowed Argo CD to synchronize the updated Kubernetes manifests.
9. Verified all three regional deployments and tested the Mumbai `/version` endpoint.

## Issue Encountered
The initial publishing workflow generated short SHA image tags, while the GitOps manifest referenced full SHA tags. Kubernetes therefore reported ErrImagePull and ImagePullBackOff.

## Resolution
Configured Docker metadata to generate full SHA tags using `type=sha,format=long`. Triggered the publishing workflow again and verified the exact image was available in GHCR.

## Results
- GitHub Actions container publishing: Successful.
- GHCR image pull: Successful.
- Argo CD synchronization: Synced.
- Argo CD health: Healthy.
- Frankfurt deployment: 1/1 Ready.
- Mumbai deployment: 1/1 Ready.
- Singapore deployment: 1/1 Ready.
- Mumbai `/version` endpoint returned version 1.0.0 and region Mumbai.

## Conclusion
The automated CI/CD and GitOps deployment pipeline successfully built, published, and deployed the updated regional API image across all three Kubernetes deployments.

## Limitations
The Kubernetes environment uses a single-node local Kind cluster. The three regional deployments do not represent independent physical regions or independent failure domains.
