#!/usr/bin/env bash
# Run on a dedicated CI Docker daemon with the two :ci images already built.
set -euo pipefail

export RAFTLAB_IMAGE=raftlab:ci DASHBOARD_IMAGE=raftlab-dashboard:ci
export CADDY_DOMAIN=http://localhost IMAGE_TAG=ci
fixture=$(mktemp -d)
compose_args=(--env-file /dev/null -f deployments/docker-compose.production.yml)
compose=(docker compose -p deployments "${compose_args[@]}")
checks=(python3 deployments/production_checks.py)

cleanup() {
  local status=$?
  if (( status != 0 )); then
    "${compose[@]}" ps --all || true
    "${compose[@]}" logs --no-color --tail 40 || true
  fi
  "${compose[@]}" down --remove-orphans || true
  rm -rf -- "$fixture"
}
trap cleanup EXIT

"${compose[@]}" config --quiet
"${compose[@]}" config --format json | "${checks[@]}" validate
for var in RAFTLAB_IMAGE DASHBOARD_IMAGE CADDY_DOMAIN IMAGE_TAG; do
  if env -u "$var" "${compose[@]}" config --quiet >"$fixture/error" 2>&1 ||
     ! grep -F "$var is required" "$fixture/error"; then
    cat "$fixture/error"
    echo "::error::Compose must reject missing $var"
    exit 1
  fi
done

"${compose[@]}" pull caddy
"${compose[@]}" create
old_containers=$("${compose[@]}" ps --all --quiet | sort)
test "$(printf '%s\n' "$old_containers" | wc -l)" -eq 7

volume_shell() {
  local volume=$1
  shift
  docker run --rm --entrypoint sh -v "deployments_${volume}:/test" "$RAFTLAB_IMAGE" "$@"
}

for volume in node{1,2,3,4,5}-data caddy-data caddy-config; do
  volume_shell "$volume" -c 'printf "%s" "$1" > /test/ci-sentinel' sh "$volume"
done

sed 's,http://127\.0\.0\.1:8080/health,[http://127.0.0.1:8080/health](http://127.0.0.1:8080/health),' \
  deployments/docker-compose.production.yml > "$fixture/malformed.yml"
if "${checks[@]}" reset --env-file /dev/null -f "$fixture/malformed.yml" >"$fixture/error" 2>&1 ||
   ! grep -F 'remove Markdown link syntax from healthcheck.test' "$fixture/error"; then
  cat "$fixture/error"
  echo "::error::Deployment reset must reject a Markdown healthcheck"
  exit 1
fi
test "$("${compose[@]}" ps --all --quiet | sort)" = "$old_containers"

"${checks[@]}" reset "${compose_args[@]}"
for volume in node{1,2,3,4,5}-data; do
  if docker volume inspect "deployments_$volume" >/dev/null 2>&1; then
    echo "::error::Raft volume $volume survived the fresh-cluster reset"
    exit 1
  fi
done

"${compose[@]}" up --wait --wait-timeout 360 --pull never --remove-orphans
"${checks[@]}" ready "${compose_args[@]}"
for volume in node{1,2,3,4,5}-data; do
  volume_shell "$volume" -c 'test ! -e /test/ci-sentinel'
done
for volume in caddy-data caddy-config; do
  test "$(volume_shell "$volume" -c 'cat /test/ci-sentinel')" = "$volume"
done
curl --fail --silent --show-error --max-time 10 http://localhost/ >/dev/null
