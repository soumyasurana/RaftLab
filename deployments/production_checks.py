#!/usr/bin/env python3
"""Validate and operate on the production Compose project."""

import json
import re
import subprocess
import sys
import time

PROJECT = "deployments"
NODES = tuple(f"node{i}" for i in range(1, 6))
SERVICES = {*NODES, "dashboard", "caddy"}
RAFT_VOLUMES = tuple(f"{node}-data" for node in NODES)
CADDY_VOLUMES = ("caddy-data", "caddy-config")
HEALTH_URL = "http://127.0.0.1:8080/health"
HEALTH_CHECKS = {
    node: ["CMD", "wget", "-q", "-O", "/dev/null", HEALTH_URL] for node in NODES
}
HEALTH_CHECKS["dashboard"] = [
    "CMD", "node", "-e",
    "fetch('http://127.0.0.1:3000/').then(r => process.exit(r.ok ? 0 : 1)).catch(() => process.exit(1))",
]
MARKDOWN_URL = re.compile(r"\[[^\]\n]*\]\(\s*<?https?://", re.IGNORECASE)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def validate(config):
    services = config["services"]
    volumes = config["volumes"]
    require(config["name"] == PROJECT, f"Compose project must be {PROJECT}")
    require(services.keys() == SERVICES, f"Expected services {sorted(SERVICES)}, got {sorted(services)}")

    for name, service in services.items():
        check = service.get("healthcheck", {})
        require(not MARKDOWN_URL.search(json.dumps(check)),
                f"{name}: remove Markdown link syntax from healthcheck.test")
        if name in HEALTH_CHECKS:
            require(not check.get("disable") and check.get("test") == HEALTH_CHECKS[name],
                    f"{name}: healthcheck.test must be {HEALTH_CHECKS[name]!r}")

    for name, dependencies in (("dashboard", set(NODES)), ("caddy", {"dashboard"})):
        actual = services[name].get("depends_on", {})
        require(actual.keys() == dependencies and all(
            spec.get("condition") == "service_healthy" and spec.get("required", True)
            for spec in actual.values()
        ), f"{name}: must require healthy dependencies {sorted(dependencies)}")
    require(services["dashboard"].get("environment", {}).get("HOSTNAME") == "0.0.0.0",
            "dashboard: HOSTNAME must bind the HTTP server for loopback and Caddy")

    require(volumes.keys() == set(RAFT_VOLUMES + CADDY_VOLUMES),
            "Expected five Raft data volumes and two Caddy state volumes")
    expected_mounts = {
        node: [(volume, f"/app/data/{node}")] for node, volume in zip(NODES, RAFT_VOLUMES)
    }
    expected_mounts["caddy"] = [("caddy-data", "/data"), ("caddy-config", "/config")]
    for name, service in services.items():
        mounts = sorted((mount.get("source", ""), mount["target"])
                        for mount in service.get("volumes", []) if mount["type"] == "volume")
        require(mounts == sorted(expected_mounts.get(name, [])), f"{name}: unexpected volume mounts {mounts}")
    for name, volume in volumes.items():
        require(volume.get("name") == f"{PROJECT}_{name}", f"{name}: unexpected Docker volume name")
        if name in RAFT_VOLUMES:
            require(not volume.get("external"), f"{name}: Raft data must use a Compose-managed volume")


def reset(compose):
    names = {f"{PROJECT}_{volume}": volume for volume in RAFT_VOLUMES}
    existing = set(subprocess.check_output(["docker", "volume", "ls", "-q"], text=True).splitlines())
    disposable = sorted(names.keys() & existing)
    for name in disposable:
        labels = json.loads(subprocess.check_output(
            ["docker", "volume", "inspect", "--format", "{{json .Labels}}", name], text=True
        )) or {}
        require(labels.get("com.docker.compose.project") == PROJECT
                and labels.get("com.docker.compose.volume") == names[name],
                f"Refusing to remove {name}: volume labels do not match the Raft Compose volume")

    subprocess.run([*compose, "down", "--remove-orphans"], check=True)
    if disposable:
        subprocess.run(["docker", "volume", "rm", *disposable], check=True)


def cluster_snapshot(compose, deadline):
    def output(command):
        remaining = deadline - time.monotonic()
        require(remaining > 0, "Raft readiness deadline reached")
        return subprocess.check_output(command, text=True, timeout=min(5, remaining))

    ids = output([*compose, "ps", "--all", "--quiet"]).splitlines()
    require(len(ids) == len(SERVICES), f"Expected {len(SERVICES)} containers, found {len(ids)}")
    containers = json.loads(output(["docker", "inspect", "--type", "container", *ids]))
    by_service = {container["Config"]["Labels"]["com.docker.compose.service"]: container
                  for container in containers}
    require(by_service.keys() == SERVICES, f"Unexpected running service set: {sorted(by_service)}")
    for name, container in by_service.items():
        state = container["State"]
        require(state["Status"] == "running" and container["RestartCount"] == 0,
                f"{name}: state={state['Status']}, restart count={container['RestartCount']}")
        health = state.get("Health", {}).get("Status")
        if name in HEALTH_CHECKS:
            require(container["Config"].get("Healthcheck", {}).get("Test") == HEALTH_CHECKS[name],
                    f"{name}: live healthcheck.test differs from the validated command")
        if name in HEALTH_CHECKS or health is not None:
            require(health == "healthy", f"{name}: health={health}")

    health = {}
    for node in NODES:
        health[node] = json.loads(output(
            ["docker", "exec", by_service[node]["Id"], "wget", "-T", "3", "-qO-", HEALTH_URL]
        ))
        require(health[node].get("nodeId") == node, f"{node}: /health returned an unexpected nodeId")
    summary = ", ".join(f"{node}={report.get('role')} term={report.get('currentTerm')} "
                        f"leader={report.get('leaderId')}" for node, report in health.items())
    leaders = [node for node, report in health.items() if report.get("role") == "Leader"]
    require(len(leaders) == 1, f"Expected exactly one Raft leader; {summary}")
    leader = leaders[0]
    term = health[leader].get("currentTerm")
    require(isinstance(term, int) and term > 0 and all(
        report.get("role") == ("Leader" if node == leader else "Follower")
        and report.get("leaderId") == leader and report.get("currentTerm") == term
        for node, report in health.items()
    ), f"Raft nodes have not converged on a leader and term; {summary}")
    return leader, term, tuple(sorted(ids))


def ready(compose):
    deadline = time.monotonic() + 90
    previous = None
    while time.monotonic() < deadline:
        try:
            snapshot = cluster_snapshot(compose, deadline)
            if snapshot == previous:
                leader, term, _ = snapshot
                print(f"All seven services running with zero restarts; Raft leader {leader}, term {term}.")
                return
            previous = snapshot
            print(f"Observed Raft leader {snapshot[0]} in term {snapshot[1]}; checking stability.", flush=True)
        except (ValueError, KeyError, subprocess.SubprocessError) as error:
            previous = None
            print(f"Waiting for deployment readiness: {error}", flush=True)
        time.sleep(max(0, min(3, deadline - time.monotonic())))
    raise ValueError("Deployment did not pass consecutive readiness checks within 90 seconds")


def main():
    action, *args = sys.argv[1:]
    compose = ["docker", "compose", "-p", PROJECT, *args]
    if action == "validate":
        validate(json.load(sys.stdin))
        print("Production Compose configuration validated.")
    elif action == "reset":
        # Validate again at the entry point that can remove containers and volumes.
        validate(json.loads(subprocess.check_output([*compose, "config", "--format", "json"], text=True)))
        reset(compose)
    elif action == "ready":
        ready(compose)
    else:
        raise ValueError(f"Unknown production check: {action}")


if __name__ == "__main__":
    try:
        main()
    except (ValueError, KeyError, OSError, subprocess.SubprocessError) as error:
        print(f"ERROR: {error}", file=sys.stderr)
        sys.exit(1)
