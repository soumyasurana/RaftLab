import { type ClusterResponse, type NodeSnapshot, type TimelineEvent } from "@/lib/types";

export function durationToMs(value: string | number | undefined | null) {
  if (value === undefined || value === null) {
    return 0;
  }

  if (typeof value === "number") {
    return value / 1_000_000;
  }

  if (/^\d+$/.test(value)) {
    return Number(value) / 1_000_000;
  }

  const match = value.match(
    /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)(?:\.(\d+))?s)?(?:(\d+)ms)?(?:(\d+)us)?(?:(\d+)ns)?$/,
  );

  if (!match) {
    return 0;
  }

  const [, h, m, s, sFrac, ms, us, ns] = match;
  return (
    (Number(h ?? 0) * 60 * 60 * 1000) +
    (Number(m ?? 0) * 60 * 1000) +
    (Number(s ?? 0) * 1000) +
    (sFrac ? Number(`0.${sFrac}`) * 1000 : 0) +
    Number(ms ?? 0) +
    Number(us ?? 0) / 1000 +
    Number(ns ?? 0) / 1_000_000
  );
}

export function formatRole(role: string | undefined) {
  if (!role) {
    return "Unknown";
  }

  return role.slice(0, 1).toUpperCase() + role.slice(1).toLowerCase();
}

export function estimateReplicationLatency(node: NodeSnapshot, leaderCommit: number) {
  const commitIndex = node.status?.commitIndex ?? 0;
  const lagEntries = Math.max(0, leaderCommit - commitIndex);
  return Math.max(node.latencyMs, lagEntries * 45);
}

export function pickLeader(nodes: NodeSnapshot[]) {
  return nodes.find((node) => node.health?.role === "Leader") ?? nodes.find((node) => node.status?.role === "Leader");
}

function newestEvent(
  type: string,
  title: string,
  detail: string,
  severity: TimelineEvent["severity"],
  nodeId?: string,
): TimelineEvent {
  return {
    id: `${type}-${nodeId ?? "cluster"}-${Date.now()}-${Math.random()
      .toString(36)
      .slice(2, 8)}`,
    type,
    title,
    detail,
    severity,
    nodeId,
    time: new Date().toISOString(),
  };
}

export function deriveTimelineEvents(
  previous: ClusterResponse | undefined,
  current: ClusterResponse,
): TimelineEvent[] {
  const events: TimelineEvent[] = [];
  const previousLeader = previous ? pickLeader(previous.nodes) : undefined;
  const currentLeader = pickLeader(current.nodes);

  if (currentLeader?.health?.nodeId && currentLeader.health.nodeId !== previousLeader?.health?.nodeId) {
    events.push(
      newestEvent(
        previousLeader ? "leader-change" : "leader-elected",
        previousLeader ? "Leader change" : "Leader elected",
        `Node ${currentLeader.health.nodeId} is now leader for term ${current.summary.currentTerm}.`,
        "success",
        currentLeader.health.nodeId,
      ),
    );
  }

  current.nodes.forEach((node) => {
    const prevNode = previous?.nodes.find((candidate) => candidate.health?.nodeId === node.health?.nodeId);
    const nodeId = node.health?.nodeId ?? "unknown";

    if (prevNode?.healthy && !node.healthy) {
      events.push(
        newestEvent("node-offline", "Leader crashed", `Node ${nodeId} became unreachable.`, "critical", nodeId),
      );
    }

    if (!prevNode?.healthy && node.healthy) {
      events.push(
        newestEvent("node-recovered", "Node restarted", `Node ${nodeId} rejoined the cluster.`, "success", nodeId),
      );
    }

    const prevRole = prevNode?.health?.role ?? prevNode?.status?.role;
    const nextRole = node.health?.role ?? node.status?.role;

    if (prevRole === "Follower" && nextRole === "Candidate") {
      events.push(
        newestEvent(
          "follower-candidate",
          "Follower became candidate",
          `Node ${nodeId} started a new election in term ${node.health?.currentTerm ?? 0}.`,
          "warning",
          nodeId,
        ),
      );
    }

    if ((prevNode?.status?.snapshot?.lastIncludedIndex ?? 0) < (node.status?.snapshot?.lastIncludedIndex ?? 0)) {
      events.push(
        newestEvent(
          "snapshot-created",
          "Snapshot created",
          `Snapshot advanced to index ${node.status?.snapshot?.lastIncludedIndex ?? 0}.`,
          "success",
          nodeId,
        ),
      );
    }

    if ((prevNode?.metrics?.snapshotsInstalled ?? 0) < (node.metrics?.snapshotsInstalled ?? 0)) {
      events.push(
        newestEvent(
          "snapshot-installed",
          "Snapshot installed",
          `Node ${nodeId} installed a newer snapshot from the leader.`,
          "info",
          nodeId,
        ),
      );
    }

    if (
      durationToMs(prevNode?.health?.uptime) > 0 &&
      durationToMs(node.health?.uptime) > 0 &&
      durationToMs(node.health?.uptime) + 5_000 < durationToMs(prevNode?.health?.uptime)
    ) {
      events.push(
        newestEvent("node-restarted", "Node restarted", `Node ${nodeId} uptime reset after a restart.`, "warning", nodeId),
      );
    }

    if ((prevNode?.metrics?.appendEntriesSent ?? 0) < (node.metrics?.appendEntriesSent ?? 0)) {
      events.push(
        newestEvent(
          "append-entries",
          "AppendEntries",
          `Leader sent ${node.metrics?.appendEntriesSent ?? 0} AppendEntries RPCs.`,
          "info",
          nodeId,
        ),
      );
    }

    if ((prevNode?.metrics?.rpcFailures ?? 0) < (node.metrics?.rpcFailures ?? 0)) {
      events.push(
        newestEvent(
          "rpc-failure",
          "Heartbeat timeout",
          `RPC failures increased to ${node.metrics?.rpcFailures ?? 0}.`,
          "warning",
          nodeId,
        ),
      );
    }
  });

  return events.sort((left, right) => right.time.localeCompare(left.time));
}

export function connectedPeers(node: NodeSnapshot) {
  return node.peers?.filter((peer) => peer.connectionState === "connected").length ?? 0;
}

export function replicationLag(node: NodeSnapshot, leaderCommit: number) {
  return Math.max(0, leaderCommit - (node.status?.commitIndex ?? 0));
}
