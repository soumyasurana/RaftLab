export type NodeRole = "Leader" | "Follower" | "Candidate" | "Offline" | string;

export interface HealthResponse {
  nodeId: string;
  role: NodeRole;
  currentTerm: number;
  leaderId?: string;
  uptime: string | number;
  buildVersion: string;
}

export interface SnapshotStatus {
  available: boolean;
  lastIncludedIndex: number;
  lastIncludedTerm: number;
  sizeBytes?: number;
}

export interface StatusResponse {
  role: NodeRole;
  currentTerm: number;
  votedFor?: string;
  commitIndex: number;
  lastApplied: number;
  lastIncludedIndex: number;
  lastIncludedTerm: number;
  logLength: number;
  snapshot: SnapshotStatus;
}

export interface PeerResponse {
  peerId: string;
  address: string;
  connectionState: string;
  nextIndex: number;
  matchIndex: number;
}

export interface MetricsResponse {
  electionsWon: number;
  electionsLost: number;
  votesGranted: number;
  votesRejected: number;
  appendEntriesSent: number;
  appendEntriesReceived: number;
  snapshotsCreated: number;
  snapshotsInstalled: number;
  rpcFailures: number;
  leaderChanges: number;
  uptime: string | number;
}

export interface SnapshotResponse {
  snapshotIndex: number;
  snapshotTerm: number;
  success: boolean;
}

export interface ActionResponse {
  success: boolean;
  message?: string;
}

export interface ChaosGroup {
  groups: string[][];
}

export interface ChaosNodeState {
  disconnected: boolean;
  crashed: boolean;
}

export interface ChaosStatusResponse {
  enabled: boolean;
  packetDropProbability: number;
  minDelayMs: number;
  maxDelayMs: number;
  partitions: ChaosGroup[];
  nodes: Record<string, ChaosNodeState>;
}

export interface NodeSnapshot {
  baseUrl: string;
  healthy: boolean;
  latencyMs: number;
  error?: string;
  health?: HealthResponse;
  status?: StatusResponse;
  peers?: PeerResponse[];
  state?: Record<string, string>;
  metrics?: MetricsResponse;
  chaos?: ChaosStatusResponse;
}

export interface ClusterSummary {
  status: "healthy" | "degraded" | "unstable";
  leaderId?: string;
  currentTerm: number;
  uptime: string;
  healthyNodes: number;
  followers: number;
  commitIndex: number;
  lastApplied: number;
  lastSnapshotIndex: number;
  totalLogEntries: number;
}

export interface ClusterResponse {
  observedAt: string;
  nodes: NodeSnapshot[];
  summary: ClusterSummary;
}

export interface TimelineEvent {
  id: string;
  type: string;
  title: string;
  detail: string;
  nodeId?: string;
  severity: "info" | "success" | "warning" | "critical";
  time: string;
}
