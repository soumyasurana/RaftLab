"use client";

import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  Background,
  Controls,
  Handle,
  MarkerType,
  MiniMap,
  Position,
  ReactFlow,
  type Edge,
  type Node,
  type NodeProps,
} from "@xyflow/react";
import { AnimatePresence, motion } from "framer-motion";
import {
  AlertTriangle,
  ArrowUpRight,
  Brain,
  CheckCircle2,
  CircleDot,
  CloudLightning,
  Database,
  Gauge,
  HeartPulse,
  Layers3,
  Loader2,
  LucideIcon,
  Network,
  PanelLeft,
  PlayCircle,
  RefreshCw,
  ShieldAlert,
  Sparkles,
  Zap,
  X,
} from "lucide-react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { cn } from "@/lib/utils";
import { formatBytes, formatDuration, formatNumber, formatPercent, formatShortDate } from "@/lib/format";
import {
  type ClusterResponse,
  type NodeSnapshot,
  type SnapshotResponse,
  type TimelineEvent,
} from "@/lib/types";
import {
  connectedPeers,
  deriveTimelineEvents,
  durationToMs,
  estimateReplicationLatency,
  formatRole,
  pickLeader,
  replicationLag,
} from "@/lib/view";

type ConnectionMode = "loading" | "websocket" | "polling" | "offline";
type ViewMode = "json" | "table";
type FaultMode = "latency" | "packetLoss" | "partition" | "failure";
type FailureMode = "crashNode" | "restartNode" | "disconnectNode" | "reconnectNode";

type StatusCardProps = {
  icon: LucideIcon;
  label: string;
  value: string;
  detail: string;
  accent?: string;
};

const SECTION_IDS = [
  "overview",
  "topology",
  "node",
  "state",
  "metrics",
  "events",
  "chaos",
  "snapshot",
  "health",
];

function getNodeKey(node: NodeSnapshot, index: number) {
  return node.health?.nodeId ?? node.baseUrl ?? `node-${index + 1}`;
}

function getNodeLabel(node: NodeSnapshot, index: number) {
  return node.health?.nodeId ?? node.baseUrl.match(/node\d+/i)?.[0] ?? `Node ${index + 1}`;
}

function SectionCard({
  title,
  description,
  children,
  action,
  id,
}: {
  title: string;
  description: string;
  children: ReactNode;
  action?: ReactNode;
  id?: string;
}) {
  return (
    <Card id={id} className="scroll-mt-24 overflow-hidden">
      <CardHeader className="flex-row items-start justify-between gap-4">
        <div>
          <CardTitle>{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </div>
        {action}
      </CardHeader>
      <CardContent>{children}</CardContent>
    </Card>
  );
}

function StatusCard({ icon: Icon, label, value, detail, accent }: StatusCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35 }}
      className="rounded-3xl border border-white/8 bg-white/[0.035] p-4 shadow-glow backdrop-blur-xl"
    >
      <div className="flex items-center justify-between gap-4">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-400">
            <span
              className={cn(
                "inline-flex h-8 w-8 items-center justify-center rounded-2xl border border-white/10 bg-white/5",
                accent,
              )}
            >
              <Icon className="h-4 w-4" />
            </span>
            {label}
          </div>
          <div className="text-3xl font-semibold tracking-tight text-white">{value}</div>
          <div className="text-sm text-slate-400">{detail}</div>
        </div>
      </div>
    </motion.div>
  );
}

type ClusterNodeData = {
  node: NodeSnapshot;
  displayName: string;
  leaderId?: string;
  selected: boolean;
};

function roleTone(role: string | undefined) {
  switch (role) {
    case "Leader":
      return "border-emerald-400/40 bg-emerald-500/12 text-emerald-200";
    case "Follower":
      return "border-sky-400/30 bg-sky-500/10 text-sky-200";
    case "Candidate":
      return "border-amber-400/35 bg-amber-500/10 text-amber-100";
    case "Offline":
      return "border-rose-400/35 bg-rose-500/10 text-rose-100";
    default:
      return "border-white/10 bg-white/5 text-slate-200";
  }
}

function healthTone(isHealthy: boolean) {
  return isHealthy
    ? "border-emerald-400/30 bg-emerald-500/10 text-emerald-200"
    : "border-rose-400/30 bg-rose-500/10 text-rose-100";
}

function ClusterNodeView({ data }: NodeProps<Node<ClusterNodeData>>) {
  const role = data.node.health?.role ?? data.node.status?.role ?? "Offline";

  return (
    <div
      className={cn(
        "w-[250px] rounded-3xl border p-4 text-left shadow-glow backdrop-blur-xl transition-transform duration-300",
        roleTone(role),
        data.selected && "scale-[1.03] ring-2 ring-emerald-400/60",
      )}
    >
      <Handle type="target" position={Position.Top} className="!h-2 !w-2 !border-0 !bg-emerald-300" />
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-semibold text-white">{data.displayName}</div>
          <div className="text-[11px] uppercase tracking-[0.16em] text-slate-300/70">
            {data.node.health?.nodeId ?? data.node.baseUrl}
          </div>
          <div className="text-xs uppercase tracking-[0.18em] text-slate-300/80">
            {formatRole(role)}
          </div>
        </div>
        <Badge className={cn("border-white/10 bg-black/10", healthTone(data.node.healthy))}>
          {data.node.healthy ? "Healthy" : "Offline"}
        </Badge>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-2 text-xs text-slate-200/90">
        <InfoPill label="Term" value={String(data.node.health?.currentTerm ?? data.node.status?.currentTerm ?? 0)} />
        <InfoPill label="Commit" value={String(data.node.status?.commitIndex ?? 0)} />
        <InfoPill label="Applied" value={String(data.node.status?.lastApplied ?? 0)} />
        <InfoPill label="Latency" value={`${data.node.latencyMs}ms`} />
      </div>

      <div className="mt-3 flex items-center justify-between text-xs text-slate-300">
        <span>{data.node.health?.leaderId ? `Leader: ${data.node.health.leaderId}` : "Awaiting leader"}</span>
        <span>{formatDuration(data.node.metrics?.uptime)}</span>
      </div>

      <Handle type="source" position={Position.Bottom} className="!h-2 !w-2 !border-0 !bg-emerald-300" />
    </div>
  );
}

function InfoPill({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/8 bg-slate-950/70 px-3 py-2">
      <div className="text-[10px] uppercase tracking-[0.2em] text-slate-500">{label}</div>
      <div className="mt-1 font-medium text-white">{value}</div>
    </div>
  );
}

function nodeColor(node: NodeSnapshot) {
  const role = node.health?.role ?? node.status?.role ?? "Offline";
  switch (role) {
    case "Leader":
      return "#34d399";
    case "Follower":
      return "#38bdf8";
    case "Candidate":
      return "#fbbf24";
    default:
      return node.healthy ? "#94a3b8" : "#fb7185";
  }
}

function useClusterFeed() {
  const [cluster, setCluster] = useState<ClusterResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [mode, setMode] = useState<ConnectionMode>("loading");
  const [lastUpdatedAt, setLastUpdatedAt] = useState<string | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const mountedRef = useRef(true);

  useEffect(() => {
    mountedRef.current = true;
    const wsUrl = process.env.NEXT_PUBLIC_RAFTLAB_WS_URL;

    const applyCluster = (payload: ClusterResponse) => {
      if (!mountedRef.current) {
        return;
      }
      setCluster(payload);
      setError(null);
      setLastUpdatedAt(new Date().toISOString());
    };

    let pollTimer: ReturnType<typeof setInterval> | null = null;

    const stopPolling = () => {
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    };

    const pollOnce = async () => {
      try {
        const response = await fetch("/api/cluster", { cache: "no-store" });
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }

        applyCluster((await response.json()) as ClusterResponse);
        setMode("polling");
      } catch (pollError) {
        if (!mountedRef.current) {
          return;
        }
        setError(pollError instanceof Error ? pollError.message : "Failed to load cluster");
        setMode("offline");
      }
    };

    const startPolling = () => {
      stopPolling();
      setMode("polling");
      void pollOnce();
      pollTimer = setInterval(() => {
        void pollOnce();
      }, 2000);
    };

    if (!wsUrl) {
      startPolling();
      return () => {
        mountedRef.current = false;
        stopPolling();
      };
    }

    let fellBackToPolling = false;

    const fallbackToPolling = () => {
      if (fellBackToPolling) {
        return;
      }
      fellBackToPolling = true;
      wsRef.current = null;
      startPolling();
    };

    try {
      setMode("loading");
      const socket = new WebSocket(wsUrl);
      wsRef.current = socket;

      socket.onopen = () => {
        if (mountedRef.current) {
          setMode("websocket");
        }
      };

      socket.onmessage = (event) => {
        try {
          applyCluster(JSON.parse(event.data) as ClusterResponse);
          setMode("websocket");
        } catch {
          setError("WebSocket payload could not be parsed.");
        }
      };

      socket.onerror = () => {
        if (!mountedRef.current) {
          return;
        }
        socket.close();
        fallbackToPolling();
      };

      socket.onclose = () => {
        if (!mountedRef.current) {
          return;
        }
        fallbackToPolling();
      };
    } catch {
      startPolling();
      return () => {
        mountedRef.current = false;
        stopPolling();
      };
    }

    return () => {
      mountedRef.current = false;
      wsRef.current?.close();
      wsRef.current = null;
      stopPolling();
    };
  }, []);

  return { cluster, error, mode, refresh: async () => {
    setMode("polling");
    const response = await fetch("/api/cluster", { cache: "no-store" });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }
    const payload = (await response.json()) as ClusterResponse;
    setCluster(payload);
    setLastUpdatedAt(new Date().toISOString());
  }, lastUpdatedAt };
}

function useTimeline(cluster: ClusterResponse | null) {
  const [events, setEvents] = useState<TimelineEvent[]>([]);
  const previousRef = useRef<ClusterResponse | undefined>(undefined);

  useEffect(() => {
    if (!cluster) {
      return;
    }

    const nextEvents = deriveTimelineEvents(previousRef.current, cluster);
    if (nextEvents.length > 0) {
      setEvents((current) => [...nextEvents, ...current].slice(0, 50));
    }
    previousRef.current = cluster;
  }, [cluster]);

  return { events };
}

function actionLabel(action: string) {
  switch (action) {
    case "enableChaos":
      return "Enable chaos";
    case "disableChaos":
      return "Disable chaos";
    case "resetChaos":
      return "Reset chaos";
    case "takeSnapshot":
      return "Take snapshot";
    case "setLatency":
      return "Inject latency";
    case "setPacketLoss":
      return "Inject packet loss";
    case "setPartition":
      return "Inject partition";
    default:
      return action;
  }
}

export function DashboardApp() {
  const { cluster, error, mode, refresh, lastUpdatedAt } = useClusterFeed();
  const { events } = useTimeline(cluster);
  const [selectedNodeId, setSelectedNodeId] = useState<string>("");
  const [viewMode, setViewMode] = useState<ViewMode>("table");
  const [search, setSearch] = useState("");
  const [isSidebarOpen, setIsSidebarOpen] = useState(false);
  const [notice, setNotice] = useState<{ tone: "success" | "error"; message: string } | null>(null);
  const [history, setHistory] = useState<ClusterResponse[]>([]);
  const [faultMode, setFaultMode] = useState<FaultMode>("latency");
  const [failureMode, setFailureMode] = useState<FailureMode>("crashNode");
  const [targetNodeId, setTargetNodeId] = useState("");
  const [minLatency, setMinLatency] = useState("25");
  const [maxLatency, setMaxLatency] = useState("180");
  const [packetLoss, setPacketLoss] = useState("12");
  const [partitionGroups, setPartitionGroups] = useState("leader|followers");
  const [isBusy, setIsBusy] = useState(false);

  useEffect(() => {
    if (!cluster) {
      return;
    }

    setHistory((current) => [...current.slice(-14), cluster]);

    const preferredNode =
      cluster.nodes.find((node) => node.health?.nodeId === cluster.summary.leaderId) ?? cluster.nodes[0];

    if (!selectedNodeId && preferredNode) {
      setSelectedNodeId(getNodeKey(preferredNode, cluster.nodes.indexOf(preferredNode)));
    }

    if (!targetNodeId && preferredNode) {
      setTargetNodeId(getNodeKey(preferredNode, cluster.nodes.indexOf(preferredNode)));
    }
  }, [cluster, selectedNodeId, targetNodeId]);

  useEffect(() => {
    if (!cluster?.nodes.length) {
      return;
    }

    const firstKey = getNodeKey(cluster.nodes[0], 0);
    const hasSelectedNode = selectedNodeId
      ? cluster.nodes.some((node, index) => getNodeKey(node, index) === selectedNodeId)
      : true;
    const hasTargetNode = targetNodeId
      ? cluster.nodes.some((node, index) => getNodeKey(node, index) === targetNodeId)
      : true;

    if (selectedNodeId && !hasSelectedNode) {
      setSelectedNodeId(firstKey);
    }

    if (targetNodeId && !hasTargetNode) {
      setTargetNodeId(firstKey);
    }
  }, [cluster, selectedNodeId, targetNodeId]);

  useEffect(() => {
    const timer = setTimeout(() => setNotice(null), 4200);
    return () => clearTimeout(timer);
  }, [notice]);

  const leader = cluster ? pickLeader(cluster.nodes) : undefined;
  const selectedNode =
    cluster?.nodes.find((node, index) => getNodeKey(node, index) === selectedNodeId) ?? leader ?? cluster?.nodes[0];
  const targetNode =
    cluster?.nodes.find((node, index) => getNodeKey(node, index) === targetNodeId) ?? leader ?? cluster?.nodes[0];
  const selectedNodeKey = selectedNode ? getNodeKey(selectedNode, cluster?.nodes.indexOf(selectedNode) ?? 0) : undefined;
  const chaosNode = targetNode ?? selectedNode;

  const filteredStateEntries = useMemo(() => {
    const entries = Object.entries(selectedNode?.state ?? {});
    if (!search.trim()) {
      return entries;
    }

    const query = search.trim().toLowerCase();
    return entries.filter(([key, value]) => key.toLowerCase().includes(query) || value.toLowerCase().includes(query));
  }, [search, selectedNode]);

  const flow = useMemo(() => {
    const flowNodes: Node[] = [];
    const flowEdges: Edge[] = [];
    const nodes = cluster?.nodes ?? [];
    const leaderNode = pickLeader(nodes);
    const leaderKey = leaderNode ? getNodeKey(leaderNode, nodes.indexOf(leaderNode)) : undefined;
    const followers = nodes.filter((node, index) => getNodeKey(node, index) !== leaderKey);

    const leaderX = 380;
    const leaderY = 80;
    const followerCols = Math.min(3, Math.max(2, Math.ceil(Math.sqrt(Math.max(1, followers.length)))));
    const followerSpacingX = 320;
    const followerSpacingY = 230;
    const gridWidth = (followerCols - 1) * followerSpacingX;
    const startX = leaderX - gridWidth / 2;
    const startY = 340;

    nodes.forEach((node, index) => {
      const key = getNodeKey(node, index);
      let x = leaderX;
      let y = leaderY;

      if (key !== leaderKey) {
        const followerIndex = followers.findIndex((candidate, candidateIndex) => getNodeKey(candidate, candidateIndex) === key);
        const row = Math.floor(Math.max(0, followerIndex) / followerCols);
        const col = Math.max(0, followerIndex) % followerCols;
        x = startX + col * followerSpacingX;
        y = startY + row * followerSpacingY;
      }

      flowNodes.push({
        id: key,
        type: "clusterNode",
        position: { x, y },
        data: {
          node,
          displayName: getNodeLabel(node, index),
          leaderId: leaderKey,
          selected: key === selectedNodeKey,
        },
      });
    });

    const leaderId = leaderKey;

    if (leaderId) {
      followers.forEach((node) => {
        const followerKey = getNodeKey(node, nodes.indexOf(node));

        flowEdges.push({
          id: `edge-${leaderId}-${followerKey}`,
          source: leaderId,
          target: followerKey,
          animated: true,
          type: "smoothstep",
          markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color: "#34d399" },
          style: { strokeWidth: 2.4, stroke: "#34d399", opacity: 0.7 },
        });
      });
    }

    return { flowNodes, flowEdges };
  }, [cluster, selectedNodeKey]);

  const chartData = useMemo(() => {
    return history.map((snapshot, index) => {
      const currentLeader = pickLeader(snapshot.nodes) ?? snapshot.nodes[0];
      const previousSnapshot = history[index - 1];
      const previousLeader = previousSnapshot ? pickLeader(previousSnapshot.nodes) ?? previousSnapshot.nodes[0] : undefined;
      const currentMetrics = currentLeader?.metrics;
      const previousMetrics = previousLeader?.metrics;
      const elapsedSeconds = previousSnapshot
        ? Math.max(
            1,
            (new Date(snapshot.observedAt).getTime() - new Date(previousSnapshot.observedAt).getTime()) / 1000,
          )
        : 2;

      const totalRpc = (currentMetrics?.appendEntriesSent ?? 0) + (currentMetrics?.appendEntriesReceived ?? 0) +
        (currentMetrics?.votesGranted ?? 0) + (currentMetrics?.votesRejected ?? 0);

      const previousRpc = (previousMetrics?.appendEntriesSent ?? 0) + (previousMetrics?.appendEntriesReceived ?? 0) +
        (previousMetrics?.votesGranted ?? 0) + (previousMetrics?.votesRejected ?? 0);

      const heartbeatsPerSec = Math.max(
        0,
        ((currentMetrics?.appendEntriesSent ?? 0) - (previousMetrics?.appendEntriesSent ?? 0)) / elapsedSeconds,
      );

      const appendEntriesPerSec = Math.max(
        0,
        ((currentMetrics?.appendEntriesSent ?? 0) - (previousMetrics?.appendEntriesSent ?? 0)) / elapsedSeconds,
      );

      const requestVotePerSec = Math.max(
        0,
        (((currentMetrics?.votesGranted ?? 0) + (currentMetrics?.votesRejected ?? 0)) -
          ((previousMetrics?.votesGranted ?? 0) + (previousMetrics?.votesRejected ?? 0))) / elapsedSeconds,
      );

      const representative = currentLeader ?? snapshot.nodes[0];

      return {
        name: new Date(snapshot.observedAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" }),
        electionCount: (currentMetrics?.electionsWon ?? 0) + (currentMetrics?.electionsLost ?? 0),
        rpcCount: totalRpc,
        rpcDelta: Math.max(0, totalRpc - previousRpc),
        appendEntriesPerSec,
        requestVotePerSec,
        snapshotCount: (currentMetrics?.snapshotsCreated ?? 0) + (currentMetrics?.snapshotsInstalled ?? 0),
        replicationLatency: representative ? estimateReplicationLatency(representative, snapshot.summary.commitIndex) : 0,
        heartbeatsPerSec,
        nodeUptime: durationToMs(currentMetrics?.uptime) / 1000,
      };
    });
  }, [history]);

  const clusterStatusTone = cluster
    ? cluster.summary.status === "healthy"
      ? "border-emerald-400/30 bg-emerald-500/10 text-emerald-200"
      : cluster.summary.status === "degraded"
        ? "border-amber-400/30 bg-amber-500/10 text-amber-100"
        : "border-rose-400/30 bg-rose-500/10 text-rose-100"
    : "border-slate-700 bg-slate-800 text-slate-200";

  const activeFaults = useMemo(() => {
    const faults: Array<{ id: string; label: string; detail: string }> = [];
    cluster?.nodes.forEach((node, index) => {
      const chaos = node.chaos;
      if (!chaos) {
        return;
      }

      const nodeLabel = getNodeLabel(node, index);

      if (chaos.enabled) {
        faults.push({
          id: `${node.health?.nodeId ?? node.baseUrl}-enabled`,
          label: `Chaos enabled on ${nodeLabel}`,
          detail: "The fault injector is active.",
        });
      }

      if (chaos.packetDropProbability > 0) {
        faults.push({
          id: `${node.health?.nodeId ?? node.baseUrl}-drop`,
          label: `${nodeLabel}: packet loss`,
          detail: formatPercent(chaos.packetDropProbability * 100),
        });
      }

      if (chaos.maxDelayMs > 0 || chaos.minDelayMs > 0) {
        faults.push({
          id: `${node.health?.nodeId ?? node.baseUrl}-latency`,
          label: `${nodeLabel}: latency`,
          detail: `${chaos.minDelayMs}-${chaos.maxDelayMs}ms`,
        });
      }

      chaos.partitions.forEach((partition, index) => {
        faults.push({
          id: `${node.health?.nodeId ?? node.baseUrl}-partition-${index}`,
          label: `${nodeLabel}: partition ${index + 1}`,
          detail: partition.groups.map((group) => group.join(", ")).join(" | "),
        });
      });

      Object.entries(chaos.nodes ?? {}).forEach(([nodeId, state]) => {
        if (state.crashed || state.disconnected) {
          faults.push({
            id: `${nodeId}-state`,
            label: nodeId,
            detail: [state.crashed ? "crashed" : "", state.disconnected ? "disconnected" : ""].filter(Boolean).join(", "),
          });
        }
      });
    });
    return faults;
  }, [cluster]);

  async function postAction(
    action:
      | "enableChaos"
      | "disableChaos"
      | "resetChaos"
      | "takeSnapshot"
      | "setLatency"
      | "setPacketLoss"
      | "setPartition"
      | FailureMode,
    payload: Record<string, unknown> = {},
    nodeUrl?: string,
  ) {
    if (!nodeUrl) {
      setNotice({ tone: "error", message: "No target node is available." });
      return;
    }

    try {
      setIsBusy(true);
      const response = await fetch("/api/action", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ action, payload, nodeUrl }),
      });

      const data = (await response.json()) as SnapshotResponse | { success?: boolean; message?: string };
      if (!response.ok || (typeof data === "object" && data && "success" in data && data.success === false)) {
        throw new Error("Action failed");
      }

      setNotice({ tone: "success", message: `${actionLabel(action)} completed.` });
      void refresh();
    } catch (actionError) {
      setNotice({
        tone: "error",
        message: actionError instanceof Error ? actionError.message : "Action failed",
      });
    } finally {
      setIsBusy(false);
    }
  }

  const snapshotInfo = selectedNode?.status?.snapshot ?? {
    available: false,
    lastIncludedIndex: selectedNode?.status?.lastIncludedIndex ?? 0,
    lastIncludedTerm: selectedNode?.status?.lastIncludedTerm ?? 0,
    sizeBytes: 0,
  };

  const storageSummary = {
    walEntries: selectedNode?.status?.logLength ?? 0,
    snapshotSize: selectedNode?.status?.snapshot?.sizeBytes ?? 0,
    stateEntries: Object.keys(selectedNode?.state ?? {}).length,
  };

  return (
    <div className="relative min-h-screen overflow-hidden text-slate-100">
      <div className="pointer-events-none absolute inset-0 grid-mask opacity-25" />
      <div className="pointer-events-none absolute left-[-10rem] top-[-8rem] h-[28rem] w-[28rem] rounded-full bg-cyan-500/10 blur-3xl" />
      <div className="pointer-events-none absolute right-[-8rem] top-[18rem] h-[24rem] w-[24rem] rounded-full bg-emerald-500/10 blur-3xl" />

      <div className="relative mx-auto flex min-h-screen max-w-[1800px] gap-6 px-4 py-4 lg:px-6">
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="fixed left-4 top-4 z-50 rounded-full border-white/10 bg-slate-950/90 px-4 shadow-glow backdrop-blur-xl lg:left-6 lg:top-6"
          onClick={() => setIsSidebarOpen((current) => !current)}
          aria-expanded={isSidebarOpen}
          aria-controls="dashboard-sidebar"
        >
          {isSidebarOpen ? <X className="h-4 w-4" /> : <PanelLeft className="h-4 w-4" />}
          {isSidebarOpen ? "Close" : "Menu"}
        </Button>

        <AnimatePresence>
          {isSidebarOpen ? (
            <>
              <motion.button
                type="button"
                className="fixed inset-0 z-40 cursor-default bg-slate-950/60 backdrop-blur-sm"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                aria-label="Close sidebar"
                onClick={() => setIsSidebarOpen(false)}
              />
              <motion.aside
                id="dashboard-sidebar"
                className="fixed inset-y-0 left-0 z-50 w-[300px] max-w-[calc(100vw-2rem)] p-4"
                initial={{ x: -340, opacity: 0 }}
                animate={{ x: 0, opacity: 1 }}
                exit={{ x: -340, opacity: 0 }}
                transition={{ type: "spring", stiffness: 280, damping: 32 }}
              >
                <div className="glass flex h-full flex-col rounded-[2rem] border border-white/10 p-5 shadow-glow">
                  <div className="flex items-center gap-3">
                    <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-emerald-400/15 text-emerald-300">
                      <Network className="h-5 w-5" />
                    </div>
                    <div>
                      <div className="text-sm font-semibold tracking-[0.08em] text-white">RaftLab</div>
                      <div className="text-xs uppercase tracking-[0.2em] text-slate-400">Distributed systems lab</div>
                    </div>
                  </div>

                  <div className="mt-6 space-y-2">
                    {SECTION_IDS.map((section) => (
                      <a
                        key={section}
                        href={`#${section}`}
                        onClick={() => setIsSidebarOpen(false)}
                        className="flex items-center justify-between rounded-2xl border border-white/5 bg-white/[0.02] px-4 py-3 text-sm text-slate-300 transition hover:border-emerald-400/30 hover:bg-white/[0.05] hover:text-white"
                      >
                        <span>{section.charAt(0).toUpperCase() + section.slice(1)}</span>
                        <ArrowUpRight className="h-4 w-4" />
                      </a>
                    ))}
                  </div>

                  <div className="mt-6 rounded-2xl border border-white/8 bg-slate-950/70 p-4">
                    <div className="flex items-center justify-between text-xs uppercase tracking-[0.18em] text-slate-500">
                      <span>Connection</span>
                      <span>{mode}</span>
                    </div>
                    <div className="mt-3 flex items-center gap-2 text-sm text-slate-200">
                      {mode === "websocket" ? <Sparkles className="h-4 w-4 text-emerald-300" /> : <RefreshCw className="h-4 w-4 text-sky-300" />}
                      <span>{mode === "websocket" ? "WebSocket live" : "Polling every 2s"}</span>
                    </div>
                    <div className="mt-2 text-xs text-slate-500">
                      {lastUpdatedAt ? `Last update ${formatShortDate(lastUpdatedAt)}` : "Waiting for the first cluster snapshot"}
                    </div>
                  </div>
                </div>
              </motion.aside>
            </>
          ) : null}
        </AnimatePresence>

        <main className="min-w-0 flex-1 space-y-6">
          <header id="overview" className="glass rounded-[2rem] border border-white/8 p-6 shadow-glow">
            <div className="flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
              <div className="max-w-3xl space-y-4">
                <div className="flex flex-wrap items-center gap-3">
                  <Badge className={clusterStatusTone}>
                    {cluster ? cluster.summary.status : "loading"}
                  </Badge>
                  <Badge>Raft consensus</Badge>
              <Badge>{cluster?.summary.leaderId ? `Leader ${cluster.summary.leaderId}` : "No leader yet"}</Badge>
                </div>

                <div>
                  <h1 className="text-4xl font-semibold tracking-tight text-white md:text-5xl">
                    RaftLab Cluster Command Center
                  </h1>
                  <p className="mt-3 max-w-2xl text-base leading-7 text-slate-400">
                    Live topology, replicated state, chaos engineering, and snapshot control for the
                    distributed system running underneath.
                  </p>
                </div>
              </div>

              <div className="grid w-full gap-3 sm:grid-cols-2 xl:max-w-[420px]">
                <Button variant="outline" onClick={() => void refresh()} disabled={isBusy}>
                  {isBusy ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                  Refresh
                </Button>
                <Button
                  onClick={() => {
                    const leaderUrl = leader?.baseUrl ?? selectedNode?.baseUrl;
                    void postAction("takeSnapshot", {}, leaderUrl);
                  }}
                  disabled={isBusy}
                >
                  <Database className="h-4 w-4" />
                  Take snapshot
                </Button>
              </div>
            </div>

            <div className="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-6">
              <StatusCard
                icon={ShieldAlert}
                label="Cluster status"
                value={cluster?.summary.status ?? "Loading"}
                detail="Aggregated from all reachable management APIs"
                accent="text-emerald-300"
              />
              <StatusCard
                icon={Brain}
                label="Current leader"
                value={cluster?.summary.leaderId ?? "—"}
                detail={`Term ${cluster?.summary.currentTerm ?? 0}`}
              />
              <StatusCard
                icon={Gauge}
                label="Healthy nodes"
                value={String(cluster?.summary.healthyNodes ?? 0)}
                detail={`${cluster?.summary.followers ?? 0} followers online`}
              />
              <StatusCard
                icon={Layers3}
                label="Commit index"
                value={String(cluster?.summary.commitIndex ?? 0)}
                detail={`Last applied ${cluster?.summary.lastApplied ?? 0}`}
              />
              <StatusCard
                icon={Database}
                label="Snapshots"
                value={String(cluster?.summary.lastSnapshotIndex ?? 0)}
                detail={`Logs ${formatNumber(cluster?.summary.totalLogEntries ?? 0)}`}
              />
              <StatusCard
                icon={HeartPulse}
                label="Cluster uptime"
                value={formatDuration(cluster?.summary.uptime)}
                detail="Derived from the latest healthy leader snapshot"
              />
            </div>
          </header>

          {error ? (
            <Card className="border-rose-400/20 bg-rose-500/10">
              <CardContent className="py-6 text-rose-100">
                Failed to load cluster state: {error}
              </CardContent>
            </Card>
          ) : null}

          <SectionCard
            id="topology"
            title="Live Cluster Topology"
            description="A React Flow view of the cluster with role-aware colors and replication links."
            action={
              <div className="flex items-center gap-2">
                <Badge>Real-time</Badge>
                <Badge>{cluster?.nodes.length ?? 0} nodes</Badge>
              </div>
            }
          >
            <div className="h-[760px] overflow-hidden rounded-[1.75rem] border border-white/8 bg-slate-950/80">
              {cluster ? (
                <ReactFlow
                  nodes={flow.flowNodes}
                  edges={flow.flowEdges}
                  nodeTypes={{ clusterNode: ClusterNodeView }}
                  fitView
                  fitViewOptions={{ padding: 0.32, minZoom: 0.5, maxZoom: 1.2 }}
                  proOptions={{ hideAttribution: true }}
                  nodesDraggable={false}
                  nodesConnectable={false}
                  elementsSelectable={false}
                >
                  <Background color="rgba(148,163,184,0.12)" gap={28} />
                  <MiniMap zoomable pannable nodeStrokeWidth={3} nodeColor={(node) => nodeColor((node.data as ClusterNodeData).node)} />
                  <Controls />
                </ReactFlow>
              ) : (
                <div className="flex h-full items-center justify-center p-8">
                  <div className="space-y-4 text-center">
                    <Skeleton className="mx-auto h-4 w-48" />
                    <Skeleton className="mx-auto h-4 w-72" />
                    <Skeleton className="mx-auto h-64 w-[700px] rounded-[2rem]" />
                  </div>
                </div>
              )}
            </div>
          </SectionCard>

          <SectionCard
            id="node"
            title="Node Details"
            description="Inspect the persistent, volatile, snapshot, and storage state for any node."
            action={
              <div className="flex items-center gap-2">
                <Select value={selectedNodeId} onChange={(event) => setSelectedNodeId(event.target.value)} className="w-[220px]">
                  {cluster?.nodes.map((node, index) => (
                    <option key={getNodeKey(node, index)} value={getNodeKey(node, index)}>
                      {getNodeLabel(node, index)}
                    </option>
                  ))}
                </Select>
              </div>
            }
          >
            {selectedNode ? (
              <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
                <div className="grid gap-4 md:grid-cols-2">
                  <Card className="border-white/8 bg-white/[0.02]">
                    <CardHeader>
                      <CardTitle>Persistent State</CardTitle>
                      <CardDescription>Durable metadata written to disk.</CardDescription>
                    </CardHeader>
                    <CardContent className="grid gap-3">
                      <InfoRow label="Current term" value={String(selectedNode.status?.currentTerm ?? 0)} />
                      <InfoRow label="Voted for" value={selectedNode.status?.votedFor ?? "none"} />
                    </CardContent>
                  </Card>

                  <Card className="border-white/8 bg-white/[0.02]">
                    <CardHeader>
                      <CardTitle>Volatile State</CardTitle>
                      <CardDescription>State that shifts while the node is online.</CardDescription>
                    </CardHeader>
                    <CardContent className="grid gap-3">
                      <InfoRow label="Commit index" value={String(selectedNode.status?.commitIndex ?? 0)} />
                      <InfoRow label="Last applied" value={String(selectedNode.status?.lastApplied ?? 0)} />
                      <InfoRow label="NextIndex" value={String((selectedNode.peers?.[0]?.nextIndex ?? 0))} />
                      <InfoRow label="MatchIndex" value={String((selectedNode.peers?.[0]?.matchIndex ?? 0))} />
                    </CardContent>
                  </Card>

                  <Card className="border-white/8 bg-white/[0.02]">
                    <CardHeader>
                      <CardTitle>Snapshot</CardTitle>
                      <CardDescription>Latest compacted state tracked by the node.</CardDescription>
                    </CardHeader>
                    <CardContent className="grid gap-3">
                      <InfoRow label="LastIncludedIndex" value={String(snapshotInfo.lastIncludedIndex)} />
                      <InfoRow label="LastIncludedTerm" value={String(snapshotInfo.lastIncludedTerm)} />
                      <InfoRow label="Snapshot size" value={formatBytes(snapshotInfo.sizeBytes ?? 0)} />
                    </CardContent>
                  </Card>

                  <Card className="border-white/8 bg-white/[0.02]">
                    <CardHeader>
                      <CardTitle>Storage</CardTitle>
                      <CardDescription>WAL and replicated state machine footprint.</CardDescription>
                    </CardHeader>
                    <CardContent className="grid gap-3">
                      <InfoRow label="WAL entries" value={String(storageSummary.walEntries)} />
                      <InfoRow label="State keys" value={String(storageSummary.stateEntries)} />
                      <InfoRow label="Leader latency" value={`${selectedNode.latencyMs}ms`} />
                    </CardContent>
                  </Card>
                </div>

                <Card className="border-white/8 bg-white/[0.02]">
                  <CardHeader>
                    <CardTitle>Replication detail</CardTitle>
                    <CardDescription>Per-peer replication state for the selected node.</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Peer</TableHead>
                          <TableHead>Connection</TableHead>
                          <TableHead>NextIndex</TableHead>
                          <TableHead>MatchIndex</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(selectedNode.peers ?? []).map((peer) => (
                          <TableRow key={peer.peerId}>
                            <TableCell className="font-medium text-white">{peer.peerId}</TableCell>
                            <TableCell>{peer.connectionState}</TableCell>
                            <TableCell>{peer.nextIndex}</TableCell>
                            <TableCell>{peer.matchIndex}</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              </div>
            ) : (
              <div className="grid gap-4 md:grid-cols-2">
                <Skeleton className="h-44 rounded-[1.75rem]" />
                <Skeleton className="h-44 rounded-[1.75rem]" />
                <Skeleton className="h-44 rounded-[1.75rem]" />
                <Skeleton className="h-44 rounded-[1.75rem]" />
              </div>
            )}
          </SectionCard>

          <SectionCard
            id="state"
            title="Replicated State Machine"
            description="Browse the full key-value store with search, table, and pretty JSON modes."
            action={
              <div className="flex flex-wrap items-center gap-2">
                <Input
                  placeholder="Search keys or values"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  className="w-[220px]"
                />
                <Button variant={viewMode === "table" ? "default" : "outline"} size="sm" onClick={() => setViewMode("table")}>
                  Table
                </Button>
                <Button variant={viewMode === "json" ? "default" : "outline"} size="sm" onClick={() => setViewMode("json")}>
                  JSON
                </Button>
                <Button variant="outline" size="sm" onClick={() => void refresh()}>
                  <RefreshCw className="h-4 w-4" />
                  Refresh
                </Button>
              </div>
            }
          >
            {viewMode === "table" ? (
              <div className="rounded-[1.5rem] border border-white/8 bg-slate-950/70">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Key</TableHead>
                      <TableHead>Value</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredStateEntries.map(([key, value]) => (
                      <TableRow key={key}>
                        <TableCell className="font-medium text-white">{key}</TableCell>
                        <TableCell className="whitespace-pre-wrap break-words font-mono text-xs text-slate-300">{value}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                {filteredStateEntries.length === 0 ? (
                  <div className="px-4 py-8 text-center text-sm text-slate-500">No keys matched the current search.</div>
                ) : null}
              </div>
            ) : (
              <pre className="overflow-auto rounded-[1.5rem] border border-white/8 bg-slate-950/80 p-5 text-xs leading-6 text-slate-200">
                {JSON.stringify(selectedNode?.state ?? {}, null, 2)}
              </pre>
            )}
          </SectionCard>

          <SectionCard
            id="metrics"
            title="Metrics Dashboard"
            description="Chart the cluster&apos;s runtime counters, rates, and observed latency."
            action={<Badge>Recharts</Badge>}
          >
            <div className="grid gap-4 xl:grid-cols-2">
              <MetricChart
                title="Election and RPC volume"
                description="Cumulative election and RPC counts over time."
                data={chartData}
                lines={[
                  { key: "electionCount", label: "Elections", color: "#34d399" },
                  { key: "rpcCount", label: "RPC count", color: "#38bdf8" },
                ]}
              />
              <MetricChart
                title="AppendEntries and votes"
                description="Approximate RPC rates derived from successive samples."
                data={chartData}
                lines={[
                  { key: "appendEntriesPerSec", label: "AppendEntries/sec", color: "#a78bfa" },
                  { key: "requestVotePerSec", label: "RequestVote/sec", color: "#f59e0b" },
                  { key: "heartbeatsPerSec", label: "Heartbeats/sec", color: "#22d3ee" },
                ]}
              />
              <MetricChart
                title="Snapshots and latency"
                description="Snapshot activity and estimated replication latency."
                data={chartData}
                lines={[
                  { key: "snapshotCount", label: "Snapshot count", color: "#34d399" },
                  { key: "replicationLatency", label: "Replication latency", color: "#fb7185" },
                ]}
              />
              <MetricChart
                title="Node uptime"
                description="Observed uptime in seconds for the selected leader sample."
                data={chartData}
                lines={[
                  { key: "nodeUptime", label: "Uptime (s)", color: "#f472b6" },
                ]}
              />
            </div>
          </SectionCard>

          <SectionCard
            id="events"
            title="Event Timeline"
            description="Newest events first, synthesized from live cluster changes."
            action={<Badge>{events.length} events</Badge>}
          >
            <div className="space-y-3">
              <AnimatePresence initial={false}>
                {(events.length > 0
                  ? events
                  : [
                      {
                        id: "empty",
                        type: "idle",
                        title: "Waiting for activity",
                        detail:
                          "The live event stream will populate as the cluster changes.",
                        severity: "info",
                        time: new Date().toISOString(),
                        nodeId: undefined,
                      },
                    ] as TimelineEvent[]).map((event) => (
                  <motion.div
                    key={event.id}
                    initial={{ opacity: 0, y: 12 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: -12 }}
                    className="flex gap-4 rounded-3xl border border-white/8 bg-white/[0.03] p-4"
                  >
                    <div
                      className={cn(
                        "flex h-11 w-11 shrink-0 items-center justify-center rounded-2xl border",
                        event.severity === "critical"
                          ? "border-rose-400/25 bg-rose-500/10 text-rose-200"
                          : event.severity === "warning"
                            ? "border-amber-400/25 bg-amber-500/10 text-amber-100"
                            : event.severity === "success"
                              ? "border-emerald-400/25 bg-emerald-500/10 text-emerald-200"
                              : "border-sky-400/25 bg-sky-500/10 text-sky-200",
                      )}
                    >
                      <CircleDot className="h-4 w-4" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <div className="font-medium text-white">{event.title}</div>
                        <Badge>{event.type}</Badge>
                        {event.nodeId ? <Badge>{event.nodeId}</Badge> : null}
                      </div>
                      <div className="mt-1 text-sm leading-6 text-slate-400">{event.detail}</div>
                    </div>
                    <div className="shrink-0 text-xs text-slate-500">{formatShortDate(event.time)}</div>
                  </motion.div>
                ))}
              </AnimatePresence>
            </div>
          </SectionCard>

          <SectionCard
            id="chaos"
            title="Chaos Control"
            description="Enable, disable, and inject controlled faults from the management API."
            action={<Badge>Active faults {activeFaults.length}</Badge>}
          >
            <div className="grid gap-4 xl:grid-cols-[1fr_1.2fr]">
              <div className="space-y-4">
                <Card className="border-white/8 bg-white/[0.02]">
                  <CardHeader>
                    <CardTitle>Cluster-wide controls</CardTitle>
                    <CardDescription>These operate on the selected management API target.</CardDescription>
                  </CardHeader>
                  <CardContent className="grid gap-3 sm:grid-cols-3">
                    <Button variant="default" onClick={() => void postAction("enableChaos", {}, chaosNode?.baseUrl)} disabled={isBusy}>
                      <Zap className="h-4 w-4" />
                      Enable
                    </Button>
                    <Button variant="outline" onClick={() => void postAction("disableChaos", {}, chaosNode?.baseUrl)} disabled={isBusy}>
                      <PlayCircle className="h-4 w-4" />
                      Disable
                    </Button>
                    <Button variant="destructive" onClick={() => void postAction("resetChaos", {}, chaosNode?.baseUrl)} disabled={isBusy}>
                      <ShieldAlert className="h-4 w-4" />
                      Reset
                    </Button>
                  </CardContent>
                </Card>

                <Card className="border-white/8 bg-white/[0.02]">
                  <CardHeader>
                    <CardTitle>Injection mode</CardTitle>
                    <CardDescription>Choose the fault you want to apply.</CardDescription>
                  </CardHeader>
                  <CardContent className="grid gap-3">
                    <div className="grid grid-cols-2 gap-3">
                      {[
                        { value: "latency", label: "Latency", icon: Gauge },
                        { value: "packetLoss", label: "Packet loss", icon: CloudLightning },
                        { value: "partition", label: "Partition", icon: Network },
                        { value: "failure", label: "Node failure", icon: AlertTriangle },
                      ].map((item) => {
                        const Icon = item.icon;
                        const active = faultMode === item.value;
                        return (
                          <button
                            key={item.value}
                            type="button"
                            onClick={() => setFaultMode(item.value as FaultMode)}
                            className={cn(
                              "rounded-2xl border px-4 py-3 text-left transition",
                              active
                                ? "border-emerald-400/40 bg-emerald-500/10 text-white"
                                : "border-white/8 bg-white/[0.02] text-slate-300 hover:bg-white/[0.05]",
                            )}
                          >
                            <Icon className="h-4 w-4" />
                            <div className="mt-2 text-sm font-medium">{item.label}</div>
                          </button>
                        );
                      })}
                    </div>

                    {faultMode === "latency" ? (
                      <div className="grid gap-3 sm:grid-cols-2">
                        <Input value={minLatency} onChange={(event) => setMinLatency(event.target.value)} placeholder="Min latency ms" />
                        <Input value={maxLatency} onChange={(event) => setMaxLatency(event.target.value)} placeholder="Max latency ms" />
                        <Button
                          className="sm:col-span-2"
                          onClick={() =>
                            void postAction(
                              "setLatency",
                              { minDelayMs: Number(minLatency), maxDelayMs: Number(maxLatency) },
                              chaosNode?.baseUrl,
                            )
                          }
                          disabled={isBusy}
                        >
                          Inject latency
                        </Button>
                      </div>
                    ) : null}

                    {faultMode === "packetLoss" ? (
                      <div className="grid gap-3">
                        <Input value={packetLoss} onChange={(event) => setPacketLoss(event.target.value)} placeholder="Packet loss percent" />
                        <Button
                          onClick={() =>
                            void postAction("setPacketLoss", { probability: Number(packetLoss) / 100 }, chaosNode?.baseUrl)
                          }
                          disabled={isBusy}
                        >
                          Inject packet loss
                        </Button>
                      </div>
                    ) : null}

                    {faultMode === "partition" ? (
                      <div className="grid gap-3">
                        <Input
                          value={partitionGroups}
                          onChange={(event) => setPartitionGroups(event.target.value)}
                          placeholder="node1,node2 | node3,node4"
                        />
                        <Button
                          onClick={() =>
                            void postAction(
                              "setPartition",
                              { groups: partitionGroups.split("|").map((group) => group.split(",").map((value) => value.trim()).filter(Boolean)) },
                              chaosNode?.baseUrl,
                            )
                          }
                          disabled={isBusy}
                        >
                          Inject partition
                        </Button>
                      </div>
                    ) : null}

                    {faultMode === "failure" ? (
                      <div className="grid gap-3">
                        <Select value={failureMode} onChange={(event) => setFailureMode(event.target.value as FailureMode)}>
                          <option value="crashNode">Crash node</option>
                          <option value="restartNode">Restart node</option>
                          <option value="disconnectNode">Disconnect node</option>
                          <option value="reconnectNode">Reconnect node</option>
                        </Select>
                        <Button
                          variant="destructive"
                          onClick={() =>
                            void postAction(failureMode, { nodeId: chaosNode?.health?.nodeId }, chaosNode?.baseUrl)
                          }
                          disabled={isBusy}
                        >
                          Apply node fault
                        </Button>
                      </div>
                    ) : null}
                  </CardContent>
                </Card>
              </div>

              <Card className="border-white/8 bg-white/[0.02]">
                <CardHeader>
                  <CardTitle>Active faults</CardTitle>
                  <CardDescription>What the dashboard currently sees in the chaos controller.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  {(activeFaults.length > 0 ? activeFaults : [{ id: "none", label: "No active faults", detail: "The cluster is currently clean." }]).map((fault) => (
                    <div key={fault.id} className="rounded-2xl border border-white/8 bg-slate-950/70 p-4">
                      <div className="font-medium text-white">{fault.label}</div>
                      <div className="mt-1 text-sm text-slate-400">{fault.detail}</div>
                    </div>
                  ))}
                </CardContent>
              </Card>
            </div>
          </SectionCard>

          <SectionCard
            id="snapshot"
            title="Snapshot Management"
            description="Track the latest snapshot and trigger new compaction events from the leader."
            action={<Badge>Snapshot aware</Badge>}
          >
            <div className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
              <Card className="border-white/8 bg-white/[0.02]">
                <CardHeader>
                  <CardTitle>Current Snapshot</CardTitle>
                  <CardDescription>The selected node&apos;s snapshot metadata.</CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 sm:grid-cols-2">
                  <InfoRow label="Available" value={String(Boolean(snapshotInfo.available))} />
                  <InfoRow label="Index" value={String(snapshotInfo.lastIncludedIndex)} />
                  <InfoRow label="Term" value={String(snapshotInfo.lastIncludedTerm)} />
                  <InfoRow label="Size" value={formatBytes(snapshotInfo.sizeBytes ?? 0)} />
                </CardContent>
              </Card>

              <Card className="border-white/8 bg-white/[0.02]">
                <CardHeader>
                  <CardTitle>Take a snapshot</CardTitle>
                  <CardDescription>Compact the leader log and emit a success notification.</CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <Button
                    className="w-full"
                    onClick={() => void postAction("takeSnapshot", {}, leader?.baseUrl ?? selectedNode?.baseUrl)}
                    disabled={isBusy}
                  >
                    <Database className="h-4 w-4" />
                    Take Snapshot
                  </Button>
                  <div className="text-sm text-slate-400">
                    This updates the WAL, the snapshot store, and the live status card once the cluster refreshes.
                  </div>
                </CardContent>
              </Card>
            </div>
          </SectionCard>

          <SectionCard
            id="health"
            title="Cluster Health"
            description="A node-by-node health table with latency and replication lag."
            action={<Badge>Node health</Badge>}
          >
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Node</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>Latency</TableHead>
                  <TableHead>Connected peers</TableHead>
                  <TableHead>Replication lag</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {cluster?.nodes.map((node, index) => (
                  <TableRow key={getNodeKey(node, index)}>
                    <TableCell className="font-medium text-white">{getNodeLabel(node, index)}</TableCell>
                    <TableCell>
                      <Badge className={cn("border-white/10", healthTone(node.healthy))}>
                        {node.healthy ? "Healthy" : "Offline"}
                      </Badge>
                    </TableCell>
                    <TableCell>{formatRole(node.health?.role ?? node.status?.role)}</TableCell>
                    <TableCell>{node.latencyMs}ms</TableCell>
                    <TableCell>{connectedPeers(node)}</TableCell>
                    <TableCell>{replicationLag(node, cluster?.summary.commitIndex ?? 0)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </SectionCard>
        </main>
      </div>

      <AnimatePresence>
        {notice ? (
          <motion.div
            initial={{ opacity: 0, y: 18 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 18 }}
            className={cn(
              "fixed bottom-5 right-5 z-50 max-w-sm rounded-2xl border px-4 py-3 shadow-glow backdrop-blur-xl",
              notice.tone === "success"
                ? "border-emerald-400/30 bg-emerald-500/15 text-emerald-100"
                : "border-rose-400/30 bg-rose-500/15 text-rose-100",
            )}
          >
            <div className="flex items-start gap-3">
              {notice.tone === "success" ? <CheckCircle2 className="mt-0.5 h-4 w-4" /> : <AlertTriangle className="mt-0.5 h-4 w-4" />}
              <div className="text-sm">{notice.message}</div>
            </div>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-2xl border border-white/8 bg-slate-950/70 px-4 py-3">
      <div className="text-[11px] uppercase tracking-[0.18em] text-slate-500">{label}</div>
      <div className="mt-1 font-medium text-white">{value}</div>
    </div>
  );
}

function MetricChart({
  title,
  description,
  data,
  lines,
}: {
  title: string;
  description: string;
  data: Array<Record<string, number | string>>;
  lines: Array<{ key: string; label: string; color: string }>;
}) {
  return (
    <Card className="border-white/8 bg-white/[0.02]">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
        <CardDescription>{description}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="h-[280px]">
          <ResponsiveContainer width="100%" height="100%">
            {lines.length > 1 ? (
              <LineChart data={data}>
                <CartesianGrid stroke="rgba(148,163,184,0.1)" strokeDasharray="4 4" />
                <XAxis dataKey="name" tick={{ fill: "#94a3b8", fontSize: 12 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fill: "#94a3b8", fontSize: 12 }} axisLine={false} tickLine={false} />
                <Tooltip
                  contentStyle={{
                    background: "#08101d",
                    border: "1px solid rgba(255,255,255,0.08)",
                    borderRadius: "16px",
                    color: "#fff",
                  }}
                />
                {lines.map((line) => (
                  <Line
                    key={line.key}
                    type="monotone"
                    dataKey={line.key}
                    name={line.label}
                    stroke={line.color}
                    strokeWidth={2.2}
                    dot={false}
                  />
                ))}
              </LineChart>
            ) : (
              <AreaChart data={data}>
                <CartesianGrid stroke="rgba(148,163,184,0.1)" strokeDasharray="4 4" />
                <XAxis dataKey="name" tick={{ fill: "#94a3b8", fontSize: 12 }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fill: "#94a3b8", fontSize: 12 }} axisLine={false} tickLine={false} />
                <Tooltip
                  contentStyle={{
                    background: "#08101d",
                    border: "1px solid rgba(255,255,255,0.08)",
                    borderRadius: "16px",
                    color: "#fff",
                  }}
                />
                {lines.map((line) => (
                  <Area
                    key={line.key}
                    type="monotone"
                    dataKey={line.key}
                    name={line.label}
                    stroke={line.color}
                    fill={`${line.color}55`}
                    fillOpacity={0.3}
                    strokeWidth={2.2}
                  />
                ))}
              </AreaChart>
            )}
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}
