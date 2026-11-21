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
import { AnimatePresence, MotionConfig, motion, useReducedMotion } from "framer-motion";
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
  formatRole,
  pickLeader,
  replicationLag,
} from "@/lib/view";

type ConnectionMode = "loading" | "polling" | "offline";
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

const SECTION_LABELS: Record<string, string> = {
  overview: "Overview",
  topology: "Live Topology",
  node: "Node Details",
  state: "Replicated State",
  metrics: "Metrics Dashboard",
  events: "Event Timeline",
  chaos: "Chaos Control",
  snapshot: "Snapshot Management",
  health: "Cluster Health",
};

function getNodeLabel(node: NodeSnapshot) {
  return node.nodeId ?? node.baseUrl.replace(/^https?:\/\//, "");
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
      <CardHeader className="items-start gap-4 sm:flex-row sm:justify-between">
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
  const role = data.node.healthy ? data.node.health?.role : "Offline";

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
          <div className="break-all text-[11px] tracking-wide text-slate-300/80">
            {data.node.baseUrl.replace(/^https?:\/\//, "")}
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
        <InfoPill label="API response" value={data.node.health ? `${data.node.latencyMs} ms` : "—"} />
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

const nodeTypes = { clusterNode: ClusterNodeView };

function nodeColor(node: NodeSnapshot) {
  const role = node.healthy ? node.health?.role : "Offline";
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
  const [isRefreshing, setIsRefreshing] = useState(false);
  const refreshRef = useRef(() => {});

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>;
    let activeRequest: AbortController | undefined;

    const poll = async () => {
      clearTimeout(timer);
      activeRequest?.abort();
      const controller = new AbortController();
      activeRequest = controller;
      const signal = AbortSignal.any([controller.signal, AbortSignal.timeout(8000)]);
      setIsRefreshing(true);

      try {
        const response = await fetch("/api/cluster", { cache: "no-store", signal });
        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }
        const payload = (await response.json()) as ClusterResponse;
        if (controller.signal.aborted) {
          return;
        }

        setCluster((previous) => {
          const knownIds = new Map(previous?.nodes.map((node) => [node.baseUrl, node.nodeId]));
          return {
            ...payload,
            nodes: payload.nodes.map((node) => ({ ...node, nodeId: node.nodeId ?? knownIds.get(node.baseUrl) })),
          };
        });
        setMode("polling");
        setError(null);
      } catch (pollError) {
        if (!controller.signal.aborted) {
          setError(pollError instanceof Error ? pollError.message : "Failed to load cluster");
          setMode("offline");
        }
      } finally {
        if (!controller.signal.aborted) {
          setIsRefreshing(false);
          timer = setTimeout(poll, 2000);
        }
      }
    };

    refreshRef.current = () => void poll();
    void poll();
    return () => {
      clearTimeout(timer);
      activeRequest?.abort();
      refreshRef.current = () => {};
    };
  }, []);

  return { cluster, error, mode, isRefreshing, refresh: () => refreshRef.current(), lastUpdatedAt: cluster?.observedAt };
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
    case "crashNode":
      return "Crash node";
    case "restartNode":
      return "Restart node";
    case "disconnectNode":
      return "Disconnect node";
    case "reconnectNode":
      return "Reconnect node";
    default:
      return action;
  }
}

export function DashboardApp() {
  const { cluster, error, mode, isRefreshing, refresh, lastUpdatedAt } = useClusterFeed();
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
  const [partitionGroups, setPartitionGroups] = useState("");
  const [isBusy, setIsBusy] = useState(false);
  const reduceMotion = useReducedMotion();
  const menuButton = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!isSidebarOpen) {
      return;
    }
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setIsSidebarOpen(false);
        menuButton.current?.focus();
      }
    }
    document.addEventListener("keydown", closeOnEscape);
    return () => document.removeEventListener("keydown", closeOnEscape);
  }, [isSidebarOpen]);

  useEffect(() => {
    if (cluster) {
      setHistory((current) => [...current.slice(-14), cluster]);
    }
  }, [cluster]);

  useEffect(() => {
    if (!notice) {
      return;
    }

    const timer = setTimeout(() => setNotice(null), 4200);
    return () => clearTimeout(timer);
  }, [notice]);

  const leader = cluster ? pickLeader(cluster.nodes) : undefined;
  const defaultNode = leader ?? cluster?.nodes[0];
  const selectedNode = cluster?.nodes.find((node) => node.baseUrl === selectedNodeId) ?? defaultNode;
  const targetNode = cluster?.nodes.find((node) => node.baseUrl === targetNodeId && node.nodeId) ??
    leader ?? cluster?.nodes.find((node) => node.nodeId);
  const selectedNodeKey = selectedNode?.baseUrl;
  const canControl = Boolean(cluster?.nodes.length) && mode === "polling";
  const canSnapshot = canControl && Boolean(leader);
  const chaosEnabledCount = cluster?.nodes.filter((node) => node.chaos?.enabled).length ?? 0;
  const knownNodeIds = cluster?.nodes.flatMap((node) => node.nodeId ? [node.nodeId] : []) ?? [];

  const validLatency = minLatency.trim() !== "" && maxLatency.trim() !== "" &&
    Number.isInteger(Number(minLatency)) && Number.isInteger(Number(maxLatency)) &&
    Number(minLatency) >= 0 && Number(maxLatency) >= Number(minLatency);

  const validPacketLoss = packetLoss.trim() !== "" && Number(packetLoss) >= 0 && Number(packetLoss) <= 100;

  const groups = partitionGroups.split("|").map((group) => group.split(",").map((value) => value.trim()).filter(Boolean));
  const partitionIds = groups.flat();
  const validPartition = groups.length >= 2 && groups.every((group) => group.length > 0) &&
    new Set(partitionIds).size === partitionIds.length && partitionIds.every((id) => knownNodeIds.includes(id));

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
    const leaderKey = leaderNode ? leaderNode.baseUrl : undefined;
    const followers = nodes.filter((node) => node.baseUrl !== leaderKey);

    const leaderX = 380;
    const leaderY = 80;
    const followerCols = Math.min(3, followers.length);
    const followerSpacingX = 320;
    const gridWidth = (followerCols - 1) * followerSpacingX;
    const startX = leaderX - gridWidth / 2;
    const startY = leaderKey ? 365 : 80;

    nodes.forEach((node) => {
      const key = node.baseUrl;
      let x = leaderX;
      let y = leaderY;

      if (key !== leaderKey) {
        const followerIndex = followers.findIndex(
          (candidate) => candidate.baseUrl === key,
        );
        x = startX + (followerIndex % followerCols) * followerSpacingX;
        y = startY + Math.floor(followerIndex / followerCols) * 270;
      }

      flowNodes.push({
        id: key,
        type: "clusterNode",
        position: { x, y },
        data: {
          node,
          displayName: getNodeLabel(node),
          selected: key === selectedNodeKey,
        },
      });
    });

    const leaderId = leaderKey;

    if (leaderId) {
      followers.forEach((node) => {
        const followerKey = node.baseUrl;

        const connected = node.healthy && leaderNode?.peers?.some(
          (peer) => peer.peerId === node.nodeId && peer.connectionState === "connected",
        );
        const color = connected ? "#34d399" : "#64748b";
        flowEdges.push({
          id: `edge-${leaderId}-${followerKey}`,
          source: leaderId,
          target: followerKey,
          animated: connected && !reduceMotion,
          type: "straight",
          markerEnd: { type: MarkerType.ArrowClosed, width: 14, height: 14, color },
          style: { strokeWidth: 2.4, stroke: color, opacity: 0.7, strokeDasharray: connected ? undefined : "6 5" },
        });
      });
    }

    return { flowNodes, flowEdges };
  }, [cluster, selectedNodeKey, reduceMotion]);

  const chartData = useMemo(() => history.map((snapshot, index) => {
    const previous = history[index - 1];
    const elapsedSeconds = previous
      ? (Date.parse(snapshot.observedAt) - Date.parse(previous.observedAt)) / 1000
      : 0;
    const totals = { electionCount: 0, rpcCount: 0, snapshotCount: 0 };
    let reportedNodes = 0;
    const rates = { appendEntriesSent: 0, appendEntriesReceived: 0, votes: 0 };
    let comparedNodes = 0;

    for (const node of snapshot.nodes) {
      const metrics = node.metrics;
      if (!node.healthy || !metrics) {
        continue;
      }
      reportedNodes++;
      const votes = metrics.votesGranted + metrics.votesRejected;
      totals.electionCount += metrics.electionsWon + metrics.electionsLost;
      totals.rpcCount += metrics.appendEntriesSent + votes;
      totals.snapshotCount += metrics.snapshotsCreated;

      const previousNode = previous?.nodes.find((candidate) => candidate.baseUrl === node.baseUrl);
      const before = previousNode?.metrics;
      if (!previousNode?.healthy || !before || elapsedSeconds <= 0 ||
          durationToMs(metrics.uptime) < durationToMs(before.uptime)) {
        continue;
      }

      comparedNodes++;
      rates.appendEntriesSent += Math.max(0, metrics.appendEntriesSent - before.appendEntriesSent);
      rates.appendEntriesReceived += Math.max(0, metrics.appendEntriesReceived - before.appendEntriesReceived);
      rates.votes += Math.max(0, votes - before.votesGranted - before.votesRejected);
    }

    return {
      name: new Date(snapshot.observedAt).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" }),
      electionCount: reportedNodes ? totals.electionCount : null,
      rpcCount: reportedNodes ? totals.rpcCount : null,
      snapshotCount: reportedNodes ? totals.snapshotCount : null,
      appendEntriesSent: comparedNodes ? rates.appendEntriesSent / elapsedSeconds : null,
      appendEntriesReceived: comparedNodes ? rates.appendEntriesReceived / elapsedSeconds : null,
      votesPerSec: comparedNodes ? rates.votes / elapsedSeconds : null,
      nodeUptime: snapshot.summary.leaderId ? durationToMs(pickLeader(snapshot.nodes)?.health?.uptime) / 1000 : null,
    };
  }), [history]);

  const clusterStatusTone = cluster
    ? cluster.summary.status === "healthy"
      ? "border-emerald-400/30 bg-emerald-500/10 text-emerald-200"
      : cluster.summary.status === "degraded"
        ? "border-amber-400/30 bg-amber-500/10 text-amber-100"
        : "border-rose-400/30 bg-rose-500/10 text-rose-100"
    : "border-slate-700 bg-slate-800 text-slate-200";

  const activeFaults = useMemo(() => {
    const faults: Array<{ id: string; label: string; detail: string }> = [];
    cluster?.nodes.forEach((node) => {
      const chaos = node.chaos;
      if (!chaos?.enabled) {
        return;
      }

      const nodeLabel = getNodeLabel(node);

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
            id: `${node.baseUrl}-${nodeId}-state`,
            label: `${nodeLabel} reports ${nodeId}`,
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
    if (action === "takeSnapshot" && !nodeUrl) {
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
        throw new Error("message" in data && data.message ? data.message : "Action failed.");
      }

      setNotice({ tone: "success", message: action === "takeSnapshot" && "snapshotIndex" in data
        ? (data.snapshotIndex > 0 ? `Snapshot is available at index ${data.snapshotIndex}.` : "No committed entries are available to snapshot.")
        : `${actionLabel(action)} completed.` });
    } catch (actionError) {
      setNotice({
        tone: "error",
        message: actionError instanceof Error ? actionError.message : "Action failed",
      });
    } finally {
      refresh();
      setIsBusy(false);
    }
  }

  const snapshotInfo = selectedNode?.status?.snapshot ?? {
    available: false,
    lastIncludedIndex: selectedNode?.status?.lastIncludedIndex ?? 0,
    lastIncludedTerm: selectedNode?.status?.lastIncludedTerm ?? 0,
    sizeBytes: 0,
  };

  const stateEntryCount = Object.keys(selectedNode?.state ?? {}).length;

  return (
    <MotionConfig reducedMotion="user">
    <div className="relative min-h-screen overflow-hidden text-slate-100">
      <div className="pointer-events-none absolute inset-0 grid-mask opacity-25" />
      <div className="pointer-events-none absolute left-[-10rem] top-[-8rem] h-[28rem] w-[28rem] rounded-full bg-cyan-500/10 blur-3xl" />
      <div className="pointer-events-none absolute right-[-8rem] top-[18rem] h-[24rem] w-[24rem] rounded-full bg-emerald-500/10 blur-3xl" />

      <div className="relative mx-auto flex min-h-screen max-w-[1800px] gap-6 px-4 py-4 lg:px-6">
        <Button
          ref={menuButton}
          type="button"
          variant="outline"
          size="sm"
          className="fixed left-4 top-4 z-[60] rounded-full border-white/10 bg-slate-950/90 px-4 shadow-glow backdrop-blur-xl lg:left-6 lg:top-6"
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
                <div className="glass mt-12 flex max-h-[calc(100dvh-5rem)] flex-col overflow-y-auto rounded-[2rem] border border-white/10 p-5 shadow-glow">
                  <div className="flex items-center gap-3">
                    <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-emerald-400/15 text-emerald-300">
                      <Network className="h-5 w-5" />
                    </div>
                    <div>
                      <div className="text-sm font-semibold tracking-[0.08em] text-white">RaftLab</div>
                      <div className="text-xs uppercase tracking-[0.2em] text-slate-400">Distributed systems lab</div>
                    </div>
                  </div>

                  <nav aria-label="Dashboard sections" className="mt-6 space-y-2">
                    {SECTION_IDS.map((section) => (
                      <a
                        key={section}
                        href={`#${section}`}
                        onClick={() => setIsSidebarOpen(false)}
                        className="flex items-center justify-between rounded-2xl border border-white/5 bg-white/[0.02] px-4 py-3 text-sm text-slate-300 transition hover:border-emerald-400/30 hover:bg-white/[0.05] hover:text-white"
                      >
                        <span>{SECTION_LABELS[section] ?? section.charAt(0).toUpperCase() + section.slice(1)}</span>
                        <ArrowUpRight className="h-4 w-4" />
                      </a>
                    ))}
                  </nav>

                  <div className="mt-6 rounded-2xl border border-white/8 bg-slate-950/70 p-4">
                    <div className="flex items-center justify-between text-xs uppercase tracking-[0.18em] text-slate-500">
                      <span>Connection</span>
                      <span
                        className={cn(
                          "rounded-full px-2 py-0.5 text-[10px] font-medium uppercase tracking-widest",
                          mode === "polling"
                            ? "bg-sky-500/15 text-sky-300"
                            : mode === "offline"
                              ? "bg-rose-500/15 text-rose-300"
                              : "bg-white/5 text-slate-400",
                        )}
                      >
                        {mode}
                      </span>
                    </div>
                    <div className="mt-3 flex items-center gap-2 text-sm text-slate-200">
                      <RefreshCw className="h-4 w-4 text-sky-300" />
                      <span>{mode === "offline" ? "Waiting to reconnect" : "Polling every 2s"}</span>
                    </div>
                    <div className="mt-2 text-xs text-slate-500">
                      {lastUpdatedAt ? `Last update ${formatShortDate(lastUpdatedAt)}` : "Waiting for the first cluster sample"}
                    </div>
                  </div>
                </div>
              </motion.aside>
            </>
          ) : null}
        </AnimatePresence>

        <main className="min-w-0 flex-1 space-y-6 pt-12">
          <header id="overview" className="glass scroll-mt-16 rounded-[2rem] border border-white/8 p-4 shadow-glow sm:p-6">
            <div className="flex flex-col gap-6 xl:flex-row xl:items-end xl:justify-between">
              <div className="max-w-3xl space-y-4">
                <div className="flex flex-wrap items-center gap-3">
                  <Badge className={clusterStatusTone}>
                    {mode === "offline" ? "Feed offline" : cluster?.nodes.length ? cluster.summary.status : cluster ? "Setup required" : "Loading"}
                  </Badge>
                  <Badge>{cluster?.summary.leaderId ? `Leader ${cluster.summary.leaderId}` : "No leader yet"}</Badge>
                </div>

                <div>
                  <h1 className="text-4xl font-semibold tracking-tight text-white md:text-5xl">
                    RaftLab Cluster Dashboard
                  </h1>
                  <p className="mt-3 max-w-2xl text-base leading-7 text-slate-400">
                    Monitor Raft nodes, inspect replicated state, and test cluster recovery.
                  </p>
                </div>
              </div>

              <div className="grid w-full gap-3 sm:grid-cols-2 xl:max-w-[420px]">
                <Button variant="outline" onClick={refresh} disabled={isRefreshing}>
                  {isRefreshing ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
                  Refresh
                </Button>
                <Button
                  onClick={() => {
                    const leaderUrl = leader?.baseUrl;
                    void postAction("takeSnapshot", {}, leaderUrl);
                  }}
                  disabled={isBusy || !canSnapshot}
                >
                  <Database className="h-4 w-4" />
                  Take snapshot
                </Button>
              </div>
            </div>

            <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-6">
              <StatusCard
                icon={ShieldAlert}
                label="Cluster status"
                value={mode === "offline" ? "Unavailable" : cluster?.nodes.length ? cluster.summary.status : cluster ? "Unconfigured" : "Loading"}
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
                value={cluster ? `${cluster.summary.healthyNodes}/${cluster.nodes.length}` : "—"}
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
                label="Longest node uptime"
                value={formatDuration(cluster?.summary.uptime)}
                detail="Across reachable nodes"
              />
            </div>
          </header>

          {error ? (
            <Card className="border-rose-400/20 bg-rose-500/10">
              <CardContent className="py-6 text-rose-100">
                <p role="status">{cluster ? `Refresh failed. Showing data from ${formatShortDate(lastUpdatedAt)}.` : "Cluster data could not be loaded."} {error}</p>
              </CardContent>
            </Card>
          ) : null}

          {cluster?.nodes.length === 0 ? (
            <Card>
              <CardContent className="pt-6 text-slate-300">
                Set <code className="text-white">RAFTLAB_NODE_URLS</code> to the management API URLs, then restart the dashboard.
              </CardContent>
            </Card>
          ) : null}

          <SectionCard
            id="topology"
            title="Live Cluster Topology"
            description="Select a node to inspect its Raft state."
            action={
              <div className="flex items-center gap-2">
                <Badge>{cluster?.nodes.length ?? 0} nodes</Badge>
              </div>
            }
          >
            <div className="h-[28rem] min-w-0 overflow-hidden sm:h-[38rem] xl:h-[44rem] rounded-[1.75rem] border border-white/8 bg-slate-950/80">
              {cluster?.nodes.length ? (
                <ReactFlow
                  nodes={flow.flowNodes}
                  edges={flow.flowEdges}
                  nodeTypes={nodeTypes}
                  onNodeClick={(_, node) => setSelectedNodeId(node.id)}
                  onSelectionChange={({ nodes }) => {
                    if (nodes.length === 1) {
                      setSelectedNodeId(nodes[0].id);
                    }
                  }}
                  colorMode="dark"
                  minZoom={0.15}
                  fitView
                  fitViewOptions={{ padding: 0.16, minZoom: 0.15, maxZoom: 1.1 }}
                  proOptions={{ hideAttribution: true }}
                  nodesDraggable={false}
                  nodesConnectable={false}
                >
                  <Background color="rgba(148,163,184,0.12)" gap={28} />
                  <MiniMap zoomable pannable nodeStrokeWidth={3} nodeColor={(node) => nodeColor((node.data as ClusterNodeData).node)} />
                  <Controls />
                </ReactFlow>
              ) : mode === "loading" ? (
                <div className="flex h-full items-center justify-center p-8">
                  <div className="space-y-4 text-center">
                    <Skeleton className="mx-auto h-4 w-48" />
                    <Skeleton className="mx-auto h-4 w-48 sm:w-72" />
                    <Skeleton className="mx-auto h-48 w-52 max-w-full rounded-[2rem] sm:w-[28rem]" />
                  </div>
                </div>
              ) : (
                <div className="flex h-full items-center justify-center p-8 text-slate-400">No topology data available.</div>
              )}
            </div>
          </SectionCard>

          <SectionCard
            id="node"
            title="Node Details"
            description="Inspect the persistent, volatile, snapshot, and storage state for any node."
            action={
              <div className="flex items-center gap-2">
                <Select aria-label="Inspect node" disabled={!selectedNode} value={selectedNode?.baseUrl ?? ""} onChange={(event) => setSelectedNodeId(event.target.value)} className="w-[220px]">
                  {cluster?.nodes.map((node) => (
                    <option key={node.baseUrl} value={node.baseUrl}>
                      {getNodeLabel(node)}
                    </option>
                  ))}
                </Select>
              </div>
            }
          >
            {selectedNode?.status ? (
              <div className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)]">
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
                      <InfoRow label="WAL length" value={String(selectedNode.status?.logLength ?? 0)} />
                      <InfoRow label="Leader changes" value={String(selectedNode.metrics?.leaderChanges ?? 0)} />
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
                      <InfoRow label="WAL entries" value={String(selectedNode.status.logLength)} />
                      <InfoRow label="State keys" value={selectedNode.state ? String(stateEntryCount) : "Unavailable"} />
                      <InfoRow label="Management API response" value={selectedNode.health ? `${selectedNode.latencyMs} ms` : "Unavailable"} />
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
                        {(selectedNode.peers ?? []).length > 0 ? (
                          (selectedNode.peers ?? []).map((peer) => (
                            <TableRow key={peer.peerId}>
                              <TableCell className="font-medium text-white">{peer.peerId}</TableCell>
                              <TableCell>{peer.connectionState}</TableCell>
                              <TableCell>{peer.nextIndex}</TableCell>
                              <TableCell>{peer.matchIndex}</TableCell>
                            </TableRow>
                          ))
                        ) : (
                          <TableRow>
                            <TableCell colSpan={4} className="py-6 text-center text-sm text-slate-500">
                              No peer data available
                            </TableCell>
                          </TableRow>
                        )}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              </div>
            ) : mode === "loading" ? (
              <div className="grid gap-4 md:grid-cols-2">
                <Skeleton className="h-44 rounded-[1.75rem]" />
                <Skeleton className="h-44 rounded-[1.75rem]" />
                <Skeleton className="h-44 rounded-[1.75rem]" />
                <Skeleton className="h-44 rounded-[1.75rem]" />
              </div>
            ) : (
              <p className="text-sm text-slate-400">{selectedNode?.error ?? "No node details available."}</p>
            )}
          </SectionCard>

          <SectionCard
            id="state"
            title="Replicated State Machine"
            description={selectedNode ? `Key-value store on ${getNodeLabel(selectedNode)}.` : "Select a node to inspect its keys and values."}
            action={
              <div className="flex flex-wrap items-center gap-2">
                <Input
                  aria-label="Search keys or values"
                  placeholder="Search keys or values"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  className="w-[220px]"
                />
                <Button variant={viewMode === "table" ? "default" : "outline"} size="sm" aria-pressed={viewMode === "table"} onClick={() => setViewMode("table")}>
                  Table
                </Button>
                <Button variant={viewMode === "json" ? "default" : "outline"} size="sm" aria-pressed={viewMode === "json"} onClick={() => setViewMode("json")}>
                  JSON
                </Button>
                <Button variant="outline" size="sm" onClick={refresh} disabled={isRefreshing}>
                  <RefreshCw className="h-4 w-4" />
                  Refresh
                </Button>
              </div>
            }
          >
            {!selectedNode?.state ? (
              <p className="text-sm text-slate-400">Replicated state is unavailable.</p>
            ) : viewMode === "table" ? (
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
                        <TableCell className="break-all font-medium text-white">{key}</TableCell>
                        <TableCell className="whitespace-pre-wrap break-all font-mono text-xs text-slate-300">{value}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                {filteredStateEntries.length === 0 ? (
                  <div className="px-4 py-8 text-center text-sm text-slate-500">{search.trim() ? "No keys matched the current search." : "The store is empty."}</div>
                ) : null}
              </div>
            ) : (
              <pre tabIndex={0} className="max-h-96 overflow-auto rounded-[1.5rem] border border-white/8 bg-slate-950/80 p-5 text-xs leading-6 text-slate-200">
                {JSON.stringify(Object.fromEntries(filteredStateEntries), null, 2)}
              </pre>
            )}
          </SectionCard>

          <SectionCard
            id="metrics"
            title="Metrics Dashboard"
            description="Election, replication, snapshot, and uptime samples."
          >
            <div className="grid gap-4 xl:grid-cols-2">
              <MetricChart
                title="Elections and RPC activity"
                description="Election outcomes, outgoing AppendEntries, and vote decisions on reachable nodes."
                data={chartData}
                lines={[
                  { key: "electionCount", label: "Elections", color: "#34d399" },
                  { key: "rpcCount", label: "AppendEntries sent + vote decisions", color: "#38bdf8" },
                ]}
              />
              <MetricChart
                title="AppendEntries and votes"
                description="Approximate RPC rates derived from successive samples."
                data={chartData}
                lines={[
                  { key: "appendEntriesSent", label: "AppendEntries sent/s", color: "#a78bfa" },
                  { key: "votesPerSec", label: "Vote decisions/s", color: "#f59e0b" },
                  { key: "appendEntriesReceived", label: "AppendEntries received/s", color: "#22d3ee" },
                ]}
              />
              <MetricChart
                title="Snapshots"
                description="Snapshots created by reachable nodes."
                data={chartData}
                lines={[
                  { key: "snapshotCount", label: "Snapshot count", color: "#34d399" },
                ]}
              />
              <MetricChart
                title="Node uptime"
                description="Observed uptime of the current leader in seconds."
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
            <div className="space-y-3 max-h-[400px] overflow-y-auto pr-2">
              {events.length === 0 ? <p className="py-4 text-sm text-slate-400">Waiting for cluster activity.</p> : null}
              <AnimatePresence initial={false}>
                {events.map((event) => (
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
                    <div className="max-w-24 shrink-0 text-right text-xs text-slate-400">{formatShortDate(event.time)}</div>
                  </motion.div>
                ))}
              </AnimatePresence>
            </div>
          </SectionCard>

          <SectionCard
            id="chaos"
            title="Chaos Control"
            description="Enable, disable, and inject controlled faults from the management API."
            action={cluster?.nodes.length ? (
              <Badge>Chaos enabled on {chaosEnabledCount}/{cluster.nodes.length} nodes</Badge>
            ) : undefined}
          >
            <div className="grid gap-4 xl:grid-cols-[1fr_1.2fr]">
              <div className="space-y-4">
                <Card className="border-white/8 bg-white/[0.02]">
                  <CardHeader>
                    <CardTitle>Cluster-wide controls</CardTitle>
                    <CardDescription>Chaos settings are applied to every configured node to keep the simulated cluster consistent.</CardDescription>
                  </CardHeader>
                  <CardContent className="grid gap-3 sm:grid-cols-3">
                    <Button variant="default" onClick={() => void postAction("enableChaos")} disabled={isBusy || !canControl}>
                      <Zap className="h-4 w-4" />
                      Enable
                    </Button>
                    <Button variant="outline" onClick={() => void postAction("disableChaos")} disabled={isBusy || !canControl}>
                      <PlayCircle className="h-4 w-4" />
                      Disable
                    </Button>
                    <Button variant="destructive" onClick={() => void postAction("resetChaos")} disabled={isBusy || !canControl}>
                      <ShieldAlert className="h-4 w-4" />
                      Reset
                    </Button>
                  </CardContent>
                </Card>

                <Card className="border-white/8 bg-white/[0.02]">
                  <CardHeader>
                    <CardTitle>Injection mode</CardTitle>
                    <CardDescription>Configured faults take effect when chaos is enabled.</CardDescription>
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
                            aria-pressed={active}
                            className={cn(
                              "rounded-2xl border px-4 py-3 text-left transition",
                              active
                                ? "border-emerald-400/40 bg-emerald-500/10 text-white"
                                : "border-white/8 bg-white/[0.02] text-slate-300 hover:bg-white/[0.05]",
                            )}
                          >
                            <Icon className="h-4 w-4" aria-hidden="true" />
                            <div className="mt-2 text-sm font-medium">{item.label}</div>
                          </button>
                        );
                      })}
                    </div>

                    {faultMode === "latency" ? (
                      <div className="grid gap-3 sm:grid-cols-2">
                        <label className="grid gap-1 text-sm">Minimum delay (ms)<Input type="number" min={0} step={1} value={minLatency} onChange={(event) => setMinLatency(event.target.value)} /></label>
                        <label className="grid gap-1 text-sm">Maximum delay (ms)<Input type="number" min={0} step={1} value={maxLatency} onChange={(event) => setMaxLatency(event.target.value)} /></label>
                        <Button
                          className="sm:col-span-2"
                          onClick={() =>
                            void postAction(
                              "setLatency",
                              { minDelayMs: Number(minLatency), maxDelayMs: Number(maxLatency) },
                            )
                          }
                          disabled={isBusy || !canControl || !validLatency}
                        >
                          Inject latency
                        </Button>
                      </div>
                    ) : null}

                    {faultMode === "packetLoss" ? (
                      <div className="grid gap-3">
                        <label className="grid gap-1 text-sm">Packet loss (%)<Input type="number" min={0} max={100} step="any" value={packetLoss} onChange={(event) => setPacketLoss(event.target.value)} /></label>
                        <Button
                          onClick={() =>
                            void postAction("setPacketLoss", { probability: Number(packetLoss) / 100 })
                          }
                          disabled={isBusy || !canControl || !validPacketLoss}
                        >
                          Inject packet loss
                        </Button>
                      </div>
                    ) : null}

                    {faultMode === "partition" ? (
                      <div className="grid gap-3">
                        <label className="grid gap-1 text-sm">
                          Partition groups
                          <Input
                            aria-describedby="partition-help"
                            value={partitionGroups}
                            onChange={(event) => setPartitionGroups(event.target.value)}
                            placeholder={knownNodeIds.length > 1 ? `${knownNodeIds[0]} | ${knownNodeIds.slice(1).join(", ")}` : "node1 | node2"}
                          />
                        </label>
                        <p id="partition-help" className="text-sm text-slate-400">
                          Separate node IDs with commas and groups with |. Omitted nodes are isolated.
                        </p>
                        <Button
                          onClick={() =>
                            void postAction(
                              "setPartition",
                              { groups },
                            )
                          }
                          disabled={isBusy || !canControl || !validPartition}
                        >
                          Inject partition
                        </Button>
                      </div>
                    ) : null}

                    {faultMode === "failure" ? (
                      <div className="grid gap-3">
                        <label className="grid gap-1 text-sm">
                          Target node
                          <Select value={targetNode?.baseUrl ?? ""} onChange={(event) => setTargetNodeId(event.target.value)}>
                            {cluster?.nodes.filter((node) => node.nodeId).map((node) => (
                              <option key={node.baseUrl} value={node.baseUrl}>{getNodeLabel(node)}</option>
                            ))}
                          </Select>
                        </label>
                        <Select aria-label="Node fault action" value={failureMode} onChange={(event) => setFailureMode(event.target.value as FailureMode)}>
                          <option value="crashNode">Crash node</option>
                          <option value="restartNode">Restart node</option>
                          <option value="disconnectNode">Disconnect node</option>
                          <option value="reconnectNode">Reconnect node</option>
                        </Select>
                        <Button
                          variant="destructive"
                          onClick={() =>
                            void postAction(failureMode, { nodeId: targetNode?.nodeId })
                          }
                          disabled={isBusy || !canControl || !targetNode?.nodeId}
                        >
                          {actionLabel(failureMode)}
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
                  {activeFaults.length === 0 ? (
                    <p className="text-sm text-slate-400">
                      {cluster?.nodes.some((node) => node.chaos) ? "No active faults reported." : "Chaos status is unavailable."}
                    </p>
                  ) : null}
                  {activeFaults.map((fault) => (
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
          >
            <div className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
              <Card className="border-white/8 bg-white/[0.02]">
                <CardHeader>
                  <CardTitle>Current Snapshot</CardTitle>
                  <CardDescription>The selected node&apos;s snapshot metadata.</CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 sm:grid-cols-2">
                  <InfoRow label="Available" value={selectedNode?.status ? (snapshotInfo.available ? "Yes" : "No") : "Unknown"} />
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
                    onClick={() => void postAction("takeSnapshot", {}, leader?.baseUrl)}
                    disabled={isBusy || !canSnapshot}
                  >
                    <Database className="h-4 w-4" />
                    Take Snapshot
                  </Button>
                  <div className="text-sm text-slate-400">
                    {canSnapshot ? `Target: ${leader?.nodeId}` : "Waiting for a reachable leader."}
                  </div>
                </CardContent>
              </Card>
            </div>
          </SectionCard>

          <SectionCard
            id="health"
            title="Cluster Health"
            description="A node-by-node health table with latency and replication lag."
          >
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Node</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead>API response</TableHead>
                  <TableHead>Connected peers</TableHead>
                  <TableHead>Replication lag</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
              {cluster?.nodes.map((node) => (
                  <TableRow
                    key={node.baseUrl}
                    className={cn(node.baseUrl === selectedNodeKey && "bg-white/[0.04]")}
                  >
                    <TableCell className="font-medium text-white">
                      <a className="inline-flex whitespace-nowrap text-emerald-200 underline-offset-4 hover:underline" href="#node" onClick={() => setSelectedNodeId(node.baseUrl)}>
                        {getNodeLabel(node)}
                      </a>
                    </TableCell>
                    <TableCell>
                      <Badge className={cn("border-white/10", healthTone(node.healthy))}>
                        {node.healthy ? "Healthy" : "Offline"}
                      </Badge>
                    </TableCell>
                    <TableCell>{formatRole(node.healthy ? node.health?.role : "Offline")}</TableCell>
                    <TableCell>{node.health ? `${node.latencyMs} ms` : "—"}</TableCell>
                    <TableCell>{node.peers ? connectedPeers(node) : "—"}</TableCell>
                    <TableCell>{node.healthy && leader?.status ? replicationLag(node, leader.status.commitIndex) : "—"}</TableCell>
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
            role={notice.tone === "error" ? "alert" : "status"}
            initial={{ opacity: 0, y: 18 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 18 }}
            className={cn(
              "fixed bottom-5 left-4 right-4 z-50 sm:left-auto sm:right-5 sm:max-w-sm rounded-2xl border px-4 py-3 shadow-glow backdrop-blur-xl",
              notice.tone === "success"
                ? "border-emerald-400/30 bg-emerald-500/15 text-emerald-100"
                : "border-rose-400/30 bg-rose-500/15 text-rose-100",
            )}
          >
            <div className="flex items-start gap-3">
              {notice.tone === "success" ? <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" /> : <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />}
              <div className="text-sm">{notice.message}</div>
              <button
                type="button"
                aria-label="Dismiss"
                onClick={() => setNotice(null)}
                className="ml-auto shrink-0 rounded-full p-0.5 opacity-60 transition hover:opacity-100"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          </motion.div>
        ) : null}
      </AnimatePresence>
    </div>
    </MotionConfig>
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
  data: Array<Record<string, number | string | null>>;
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
