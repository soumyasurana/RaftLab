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

export const dashboardActions = [
  "enableChaos", "disableChaos", "resetChaos", "takeSnapshot", "setLatency",
  "setPacketLoss", "setPartition", "crashNode", "restartNode", "disconnectNode", "reconnectNode",
] as const;

export type DashboardAction = (typeof dashboardActions)[number];

function readNodeUrls() {
  const raw = process.env.RAFTLAB_NODE_URLS?.trim();

  if (!raw) {
    return [];
  }

  return [...new Set(raw
    .split(",")
    .map((value) => value.trim().replace(/\/+$/, ""))
    .filter(Boolean))];
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
    const headers = new Headers(init.headers);
    headers.set("Accept", "application/json");
    if (init.body) {
      headers.set("Content-Type", "application/json");
    }
    const response = await fetch(url, {
      ...init,
      cache: "no-store",
      signal: controller.signal,
      headers,
    });

    if (!response.ok) {
      const body = await response.json().catch(() => null);
      throw new Error(typeof body?.error === "string" ? body.error : `HTTP ${response.status}`);
    }

    return (await response.json()) as T;
  } finally {
    clearTimeout(timer);
  }
}

async function tryFetchJson<T>(url: string, timeoutMs = DEFAULT_TIMEOUT_MS) {
  try {
    return (await fetchJson<T>(url, timeoutMs)) ?? undefined;
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
  const healthRequest = (async () => {
    const started = performance.now();
    const health = await tryFetchJson<HealthResponse>(`${baseUrl}/health`);
    return { health, latencyMs: Math.round(performance.now() - started) };
  })();

  const [{ health, latencyMs }, status, peers, state, metrics, chaos] = await Promise.all([
    healthRequest,
    tryFetchJson<StatusResponse>(`${baseUrl}/status`),
    tryFetchJson<PeerResponse[]>(`${baseUrl}/peers`),
    tryFetchJson<Record<string, string>>(`${baseUrl}/state`),
    tryFetchJson<MetricsResponse>(`${baseUrl}/metrics`),
    tryFetchJson<ChaosStatusResponse>(`${baseUrl}/chaos`),
  ]);

  const chaosState = normalizeNodeState(chaos);
  const simulatedCrash = Boolean(health && chaosState?.enabled && chaosState.nodes[health.nodeId]?.crashed);
  const healthy = Boolean(health && status) && !simulatedCrash;

  return {
    baseUrl,
    nodeId: health?.nodeId,
    healthy,
    latencyMs,
    health,
    status,
    peers,
    state,
    metrics,
    chaos: chaosState,
    error: healthy ? undefined : simulatedCrash ? "node is crashed by chaos control" : "management API unavailable",
  };
}

function summarize(nodes: NodeSnapshot[]): ClusterSummary {
  const healthyNodes = nodes.filter((node) => node.healthy && node.health && node.status);
  const leaders = healthyNodes.filter((node) => node.health?.role === "Leader");
  const leader = leaders.length === 1 ? leaders[0] : undefined;
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
  if (!leader) {
    status = "unstable";
  } else if (healthyNodes.length === nodes.length) {
    status = "healthy";
  } else if (healthyNodes.length > nodes.length / 2) {
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
  action: DashboardAction,
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

export function isDashboardAction(value: unknown): value is DashboardAction {
  return typeof value === "string" && dashboardActions.includes(value as DashboardAction);
}

export function isClusterChaosAction(action: DashboardAction) {
  return action !== "takeSnapshot";
}

// Every node owns its own transport-level chaos controller. Applying a control
// action to every configured node keeps the simulated network view consistent.
export async function issueClusterChaosAction(action: DashboardAction, payload: Record<string, unknown> = {}) {
  const nodeUrls = getConfiguredNodeUrls();
  if (nodeUrls.length === 0) {
    throw new Error("No management API URLs configured.");
  }

  const results = await Promise.allSettled(
    nodeUrls.map((nodeUrl) => issueNodeAction(nodeUrl, action, payload)),
  );
  const failures = results.filter((result) => result.status === "rejected" || !result.value.success);

  if (failures.length > 0) {
    const messages = failures.map((result) => result.status === "rejected"
      ? (result.reason instanceof Error ? result.reason.message : "Management API action failed.")
      : (("message" in result.value && result.value.message) || "Management API action failed."));
    return {
      success: false,
      message: `Applied on ${nodeUrls.length - failures.length}/${nodeUrls.length} nodes. ${[...new Set(messages)].join(" ")}`,
    } satisfies ActionResponse;
  }

  return { success: true } satisfies ActionResponse;
}

export function getConfiguredNodeUrls() {
  return readNodeUrls();
}
