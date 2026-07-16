# Kubernetes Deployment

This directory contains a production-style Kubernetes deployment for the RaftLab cluster.

## Architecture

- `StatefulSet` with 5 replicas for stable identities: `raft-0` through `raft-4`
- Headless `Service` for peer discovery via Kubernetes DNS
- ConfigMap-backed per-node Raft configuration
- Separate persistent volumes for:
  - WAL
  - metadata
  - snapshots
- StatefulSet `volumeClaimTemplates` for the per-pod storage layout
- HTTP management API probes for readiness and liveness
- NetworkPolicy to keep traffic limited to the Raft pods and cluster DNS

The cluster uses the default namespace so the peer DNS names match the example:

`raft-0.raft.default.svc.cluster.local`

## Prerequisites

- Kubernetes cluster running locally with Kind, Minikube, Docker Desktop Kubernetes, or k3d
- `kubectl`
- A locally built image named `raftlab:local`

Build the image from the repo root:

```bash
docker build -t raftlab:local -f deployments/Dockerfile .
```

If you use Kind or k3d, load the image into the cluster before applying the manifests.

## Deploy

Create or confirm the namespace and apply the kustomization:

```bash
kubectl apply -k deployments/kubernetes
```

Dry-run validation:

```bash
kubectl apply --dry-run=client -k deployments/kubernetes
```

## Inspect

Check the pods:

```bash
kubectl get pods
```

Watch the rollout:

```bash
kubectl get pods -w
```

View logs:

```bash
kubectl logs raft-0
kubectl logs -f raft-0
```

Port-forward the management API:

```bash
kubectl port-forward pod/raft-0 8080:8080
```

Query the API:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/status
curl http://localhost:8080/peers
curl http://localhost:8080/state
```

Open an exec shell in a pod:

```bash
kubectl exec -it raft-0 -- sh
```

## Test Checklist

- Cluster starts and all 5 pods become ready
- One leader is elected
- Leader failover works after deleting the leader pod
- PVC data survives pod restarts
- WAL, metadata, and snapshot files survive a pod restart
- Management API responds through `/health`
- WebSocket endpoint is only verifiable if that feature exists in the current build

## Delete

Remove the deployment:

```bash
kubectl delete -k deployments/kubernetes
```

Delete the namespace only if you intentionally changed it from `default`:

```bash
kubectl delete namespace <your-namespace>
```

## Notes

- The app already handles `SIGTERM` by stopping heartbeats, shutting down the gRPC server, shutting down the Fiber API, and flushing WAL state.
- `publishNotReadyAddresses: true` is enabled so peers can resolve each other during bootstrap.
- The StatefulSet declares the `volumeClaimTemplates` directly, so each pod gets dedicated WAL, metadata, and snapshot PVCs.
