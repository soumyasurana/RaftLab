#!/usr/bin/env bash
set -euo pipefail

tag="$1"
env_b64="$2"
staging="$3"

[[ "$tag" =~ ^[0-9a-f]{40}$ ]]
[[ "$staging" =~ ^/tmp/raftlab-release\.[A-Za-z0-9]{8}$ ]]

base="/opt/raftlab"
project_file="$base/compose-project"
if [[ ! -s "$project_file" || $(cat "$project_file") != deployments ]]; then
  echo "ERROR: Expected Compose project 'deployments' in $project_file." >&2
  exit 1
fi

# A retry of the same commit must leave the live release's bind mounts and env intact.
mkdir -p "$base/releases"
release=$(mktemp -d "$base/releases/$tag.XXXXXXXX")
mkdir -p "$release/configs/docker"
install -m 644 "$staging"/configs/docker/*.yaml "$release/configs/docker/"
install -m 644 "$staging"/{docker-compose.production.yml,Caddyfile,production_checks.py} "$release/"
printf '%s' "$env_b64" | base64 -d > "$release/.env"
chmod 600 "$release/.env"

compose_args=(
  --project-directory "$release"
  --env-file "$release/.env"
  -f "$release/docker-compose.production.yml"
)
compose() {
  docker compose -p deployments "${compose_args[@]}" "$@"
}
checks() {
  python3 "$release/production_checks.py" "$@"
}

echo "Validating the release before stopping the existing stack..."
compose config --format json | checks validate

echo "Pulling images before stopping the existing stack..."
compose pull

echo "Removing old project containers and resetting the five Raft data volumes..."
checks reset "${compose_args[@]}"

echo "Starting the fresh Raft cluster, dashboard, and Caddy..."
if ! compose up --wait --wait-timeout 360 --pull never --remove-orphans ||
   ! checks ready "${compose_args[@]}"; then
  echo "ERROR: Release $tag failed startup or Raft readiness. See container state and logs below." >&2
  compose ps --all || true
  ids=$(compose ps --all --quiet) || ids=
  if [[ -n "$ids" ]]; then
    mapfile -t containers <<< "$ids"
    docker inspect --type container --format '{{.Name}} {{json .State}}' "${containers[@]}" || true
  fi
  compose logs --no-color --tail 40 || true
  exit 1
fi

ln -sfn "$release" "$base/current.next"
mv -Tf "$base/current.next" "$base/current"
compose ps
echo "Deployment verified for commit $tag at $release."
