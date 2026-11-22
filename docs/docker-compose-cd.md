# Compute Engine Docker Compose deployment

RaftLab production deployment uses one Ubuntu Compute Engine VM. Docker Compose runs five private Raft containers, a private dashboard container, and Caddy as the only public HTTP/HTTPS entry point. Raft data is stored in named Docker volumes. Images are immutable commit-SHA tags in Artifact Registry.

## Create GCP resources

Replace the values below before running them. The expected project is `raftlab-prod`.

```bash
export PROJECT_ID=raftlab-prod REGION=asia-south1 ZONE=asia-south1-a
export REPOSITORY=raftlab INSTANCE=raftlab-prod-vm
gcloud services enable artifactregistry.googleapis.com compute.googleapis.com iap.googleapis.com --project "$PROJECT_ID"
gcloud artifacts repositories create "$REPOSITORY" --project "$PROJECT_ID" --location "$REGION" --repository-format=docker
gcloud compute instances create "$INSTANCE" --project "$PROJECT_ID" --zone "$ZONE" --machine-type=e2-standard-2 --image-family=ubuntu-2404-lts-amd64 --image-project=ubuntu-os-cloud --boot-disk-size=30GB --tags=raftlab-web
gcloud compute firewall-rules create raftlab-web --project "$PROJECT_ID" --allow=tcp:80,tcp:443 --target-tags=raftlab-web
```

Do not open ports 8080, 3000, or 50051 in the firewall.

## VM bootstrap

Connect through IAP, install Docker Engine and Compose V2, then grant the deployment Linux user access to Docker:

```bash
gcloud compute ssh DEPLOY_USER@raftlab-prod-vm --project raftlab-prod --zone asia-south1-a --tunnel-through-iap
sudo apt-get update && sudo apt-get install -y docker.io docker-compose-v2
sudo usermod -aG docker DEPLOY_USER
sudo mkdir -p /opt/raftlab/releases
```

## GitHub OIDC

Create the OIDC trust (replace `GITHUB_OWNER/GITHUB_REPOSITORY` only if this repository moves):

```bash
export PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format='value(projectNumber)')
export DEPLOYER=github-raftlab-deployer
gcloud iam service-accounts create "$DEPLOYER" --project "$PROJECT_ID"
export DEPLOYER_EMAIL="${DEPLOYER}@${PROJECT_ID}.iam.gserviceaccount.com"
for ROLE in roles/artifactregistry.writer roles/compute.osAdminLogin roles/iap.tunnelResourceAccessor; do
  gcloud projects add-iam-policy-binding "$PROJECT_ID" --member="serviceAccount:${DEPLOYER_EMAIL}" --role="$ROLE"
done
gcloud iam workload-identity-pools create github --project "$PROJECT_ID" --location=global --display-name=github
gcloud iam workload-identity-pools providers create-oidc github --project "$PROJECT_ID" --location=global --workload-identity-pool=github --issuer-uri=https://token.actions.githubusercontent.com --attribute-mapping='google.subject=assertion.sub,attribute.repository=assertion.repository'
export WIF="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/providers/github"
gcloud iam service-accounts add-iam-policy-binding "$DEPLOYER_EMAIL" --project "$PROJECT_ID" --role=roles/iam.workloadIdentityUser --member="principalSet://iam.googleapis.com/projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/github/attribute.repository/GITHUB_OWNER/GITHUB_REPOSITORY"
```

The VM service account also needs `roles/artifactregistry.reader` so Docker can pull images. The VM deployment identity needs no downloaded service-account key.

Create these GitHub repository variables: `GCP_PROJECT_ID`, `GCP_REGION`, `GAR_REPOSITORY`, `GCP_WORKLOAD_IDENTITY_PROVIDER`, `GCP_SERVICE_ACCOUNT`, `GCE_INSTANCE`, `GCE_ZONE`, `GCE_DEPLOY_USER`, and `CADDY_DOMAIN`. Create a protected `production` environment.

`CADDY_DOMAIN` must be a DNS name pointing at the VM external IP; Caddy obtains HTTPS certificates automatically. No GitHub secret is required when using IAP and Workload Identity Federation.

## Deploy and rollback

A successful CI run on `main` starts **Deploy to Compute Engine**. The workflow validates the rendered Compose file and pulls images before stopping the `deployments` project. Each release recreates all five Raft data volumes and retains `deployments_caddy-data` and `deployments_caddy-config`. Compose starts the dashboard after all five nodes are healthy, then starts Caddy after the dashboard is healthy.

The workflow advances `/opt/raftlab/current` after two consecutive checks confirm that all seven containers are running with zero restarts and all five nodes report the same leader and term. The deployment intentionally discards Raft state. To roll back the application version, revert the application changes on `main` while retaining the updated deployment workflow.

To verify the active release on the VM:

```bash
sudo sh -c 'cd /opt/raftlab/current && docker compose -p deployments --env-file .env -f docker-compose.production.yml ps --all'
sudo sh -c 'cd /opt/raftlab/current && python3 production_checks.py ready --env-file .env -f docker-compose.production.yml'
```

From a client outside the VM, verify the public Caddy endpoint with `curl --fail --silent --show-error https://raftlab.blocklogsecurity.com/ -o /dev/null`.

Use `sudo docker compose -p deployments --env-file .env -f docker-compose.production.yml logs -f` from the release directory for troubleshooting. Consult the Actions log for the path and diagnostics of a failed deployment.
