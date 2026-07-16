import {
  type ActionResponse,
  type ChaosStatusResponse,
  type ClusterResponse,
  type ClusterSummary,
  type HealthResponse,
  type MetricsResponse,
  type NodeSnapshot,
  type PeerResponse,
  type SnapshotResponse,
  type StatusResponse,
} from "@/lib/types";

const DEFAULT_TIMEOUT_MS = 1800;

function readNodeUrls() {
  const raw = process.env.RAFTLAB_NODE_URLS?.trim();

  if (!raw) {
    return [];
  }

  return raw
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean)
    .map((value) => value.replace(/\/+$/, ""));
}

function toNumber(value: string | number | undefined, fallback = 0) {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }

  if (typeof value === "string") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }

  return fallback;
}

async function fetchJson<T>(
  url: string,
  timeoutMs = DEFAULT_TIMEOUT_MS,
  init: RequestInit = {},
): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(url, {
      cache: "no-store",
      signal: controller.signal,
      headers: {
        Accept: "application/json",
        ...(init.body ? { "Content-Type": "application/json" } : {}),
        ...(init.headers ?? {}),
      },
      ...init,
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    return (await response.json()) as T;
  } finally {
    clearTimeout(timer);
  }
}

async function tryFetchJson<T>(url: string, timeoutMs = DEFAULT_TIMEOUT_MS) {
  try {
    return await fetchJson<T>(url, timeoutMs);
  } catch {
    return undefined;
  }
}

function normalizeNodeState(state: ChaosStatusResponse | undefined) {
  if (!state) {
    return undefined;
  }

  return {
    ...state,
    nodes: state.nodes ?? {},
    partitions: state.partitions ?? [],
  };
}

export async function fetchNodeSnapshot(baseUrl: string): Promise<NodeSnapshot> {
  const started = performance.now();

  const [health, status, peers, state, metrics, chaos] = await Promise.all([
    tryFetchJson<HealthResponse>(`${baseUrl}/health`),
    tryFetchJson<StatusResponse>(`${baseUrl}/status`),
    tryFetchJson<PeerResponse[]>(`${baseUrl}/peers`),
    tryFetchJson<Record<string, string>>(`${baseUrl}/state`),
    tryFetchJson<MetricsResponse>(`${baseUrl}/metrics`),
    tryFetchJson<ChaosStatusResponse>(`${baseUrl}/chaos`),
  ]);

  const healthy = Boolean(health && status);
  const latencyMs = Math.max(0, Math.round(performance.now() - started));

  return {
    baseUrl,
    healthy,
    latencyMs,
    health,
    status,
    peers,
    state,
    metrics,
    chaos: normalizeNodeState(chaos),
    error: healthy ? undefined : "management API unavailable",
  };
}

function summarize(nodes: NodeSnapshot[]): ClusterSummary {
  const healthyNodes = nodes.filter((node) => node.healthy && node.health && node.status);
  const leader = healthyNodes.find((node) => node.health?.role === "Leader") ?? healthyNodes[0];
  const followers = healthyNodes.filter((node) => node.health?.role === "Follower").length;
  const commitIndex = Math.max(...healthyNodes.map((node) => node.status?.commitIndex ?? 0), 0);
  const lastApplied = Math.max(...healthyNodes.map((node) => node.status?.lastApplied ?? 0), 0);
  const lastSnapshotIndex = Math.max(
    ...healthyNodes.map((node) => node.status?.snapshot?.lastIncludedIndex ?? 0),
    0,
  );
  const totalLogEntries = healthyNodes.reduce((sum, node) => sum + (node.status?.logLength ?? 0), 0);

  const uptimeValues = healthyNodes
    .map((node) => toNumber(node.health?.uptime, 0))
    .filter((value) => value > 0);
  const uptime = uptimeValues.length > 0
    ? Math.max(...uptimeValues)
    : 0;

  let status: ClusterSummary["status"] = "unstable";
  if (healthyNodes.length === nodes.length && leader) {
    status = "healthy";
  } else if (healthyNodes.length > 0) {
    status = "degraded";
  }

  return {
    status,
    leaderId: leader?.health?.nodeId,
    currentTerm: leader?.health?.currentTerm ?? Math.max(...healthyNodes.map((node) => node.health?.currentTerm ?? 0), 0),
    uptime: uptime ? String(uptime) : "0",
    healthyNodes: healthyNodes.length,
    followers,
    commitIndex,
    lastApplied,
    lastSnapshotIndex,
    totalLogEntries,
  };
}

export async function loadClusterState() {
  const nodeUrls = readNodeUrls();
  const nodes = await Promise.all(nodeUrls.map((url) => fetchNodeSnapshot(url)));

  return {
    observedAt: new Date().toISOString(),
    nodes,
    summary: summarize(nodes),
  } satisfies ClusterResponse;
}

export async function issueNodeAction(
  baseUrl: string,
  action:
    | "enableChaos"
    | "disableChaos"
    | "resetChaos"
    | "takeSnapshot"
    | "setLatency"
    | "setPacketLoss"
    | "setPartition"
    | "crashNode"
    | "restartNode"
    | "disconnectNode"
    | "reconnectNode",
  payload: Record<string, unknown> = {},
) {
  switch (action) {
    case "enableChaos":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/enable`, 2500, { method: "POST" });
    case "disableChaos":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/disable`, 2500, { method: "POST" });
    case "resetChaos":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/reset`, 2500, { method: "POST" });
    case "takeSnapshot":
      return fetchJson<SnapshotResponse>(`${baseUrl}/snapshot`, 3500, { method: "POST" });
    case "setLatency":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/latency`, 2500, {
        method: "POST",
        body: JSON.stringify(payload),
      });
    case "setPacketLoss":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/packet-loss`, 2500, {
        method: "POST",
        body: JSON.stringify(payload),
      });
    case "setPartition":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/partition`, 2500, {
        method: "POST",
        body: JSON.stringify(payload),
      });
    case "crashNode":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/node-failure`, 2500, {
        method: "POST",
        body: JSON.stringify(payload),
      });
    case "restartNode":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/node-restart`, 2500, {
        method: "POST",
        body: JSON.stringify(payload),
      });
    case "disconnectNode":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/node-disconnect`, 2500, {
        method: "POST",
        body: JSON.stringify(payload),
      });
    case "reconnectNode":
      return fetchJson<ActionResponse>(`${baseUrl}/chaos/node-reconnect`, 2500, {
        method: "POST",
        body: JSON.stringify(payload),
      });
    default:
      throw new Error(`unsupported action: ${action}`);
  }
}

export function getConfiguredNodeUrls() {
  return readNodeUrls();
}

export function toDisplayUptime(value: string | number | undefined) {
  const numeric = toNumber(value, 0);

  if (numeric <= 0) {
    return "0s";
  }

  const seconds = numeric / 1_000_000_000;
  const minutes = seconds / 60;
  const hours = minutes / 60;

  if (hours >= 1) {
    return `${hours.toFixed(1)}h`;
  }

  if (minutes >= 1) {
    return `${minutes.toFixed(1)}m`;
  }

  return `${seconds.toFixed(1)}s`;
}

export function estimateReplicationLatency(node: NodeSnapshot, leaderCommit: number) {
  const matchIndex = node.status?.commitIndex ?? 0;
  const lagEntries = Math.max(0, leaderCommit - matchIndex);
  return Math.max(node.latencyMs, lagEntries * 45);
}
