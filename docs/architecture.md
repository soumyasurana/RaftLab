# RaftLab Architecture

## Overview

RaftLab is a production-inspired implementation of the **Raft Consensus Algorithm** written in Go. The project simulates a distributed cluster of independent nodes that cooperate to maintain a strongly consistent replicated state machine.

Each node runs as an independent process with its own persistent storage, networking layer, and Raft engine. Nodes communicate exclusively through gRPC using Protocol Buffers.

The system is designed to demonstrate the core principles of distributed consensus while maintaining a clean, modular architecture suitable for extension and experimentation.

---

# High-Level Architecture

```text
                           +----------------------+
                           |  Chaos Controller    |
                           |----------------------|
                           | Kill Nodes           |
                           | Pause Nodes          |
                           | Network Partition    |
                           | Latency Injection    |
                           +----------+-----------+
                                      |
     --------------------------------------------------------------------
     |              |               |               |                   |
+-----------+ +-----------+ +-----------+ +-----------+ +-----------+
|  Node 1   | |  Node 2   | |  Node 3   | |  Node 4   | |  Node 5   |
+-----------+ +-----------+ +-----------+ +-----------+ +-----------+
       \            |              |              |              /
        \___________|______________|______________|_____________/
                         gRPC Communication Network
```

Every node is identical and capable of becoming the leader.

---

# Node Architecture

Each node consists of several independent components.

```text
+------------------------------------------------------+
|                    Raft Node                         |
|------------------------------------------------------|
|                                                      |
|  HTTP Management API                                 |
|                                                      |
|  gRPC Server                                         |
|                                                      |
|  Raft Engine                                         |
|     • Leader Election                               |
|     • Heartbeats                                    |
|     • Log Replication                               |
|     • Commit Logic                                  |
|                                                      |
|  Write Ahead Log (WAL)                              |
|                                                      |
|  Replicated Key-Value State Machine                 |
|                                                      |
|  Configuration                                      |
|                                                      |
|  Structured Logger                                  |
+------------------------------------------------------+
```

---

# Major Components

## Raft Engine

The Raft engine contains the consensus implementation responsible for coordinating the cluster.

Responsibilities include:

* Leader election
* Vote requests
* Heartbeat generation
* Log replication
* Commit index advancement
* Applying committed entries
* Term management
* State transitions

The engine is isolated from networking and storage to keep the implementation modular.

---

## RPC Layer

The RPC layer enables communication between nodes using gRPC.

Responsibilities include:

* RequestVote RPC
* AppendEntries RPC
* Connection management
* Message serialization
* Timeout handling
* Error propagation

All inter-node communication passes through this layer.

---

## Write-Ahead Log

Each node maintains its own persistent log.

Responsibilities:

* Append log entries
* Read log entries
* Recover after restart
* Preserve ordering
* Store metadata

The WAL is the source of truth for replicated commands.

---

## State Machine

The replicated state machine represents the application's data.

Initial implementation:

* In-memory key-value store

Responsibilities:

* Apply committed commands
* Maintain deterministic state
* Serve read requests

Only committed log entries are applied.

---

## Chaos Controller

The chaos controller simulates real-world failures.

Supported operations:

* Kill node
* Pause node
* Resume node
* Introduce latency
* Create network partitions
* Restore connectivity

This component is independent of the consensus engine.

---

## HTTP Management API

Each node exposes a lightweight HTTP API for inspection and administration.

Typical endpoints include:

* Node status
* Current leader
* Current term
* Commit index
* Log information
* Cluster health

The API is intended for debugging and future dashboard integration.

---

# Communication Flow

## Leader Election

```text
Follower
    │
Election Timeout
    │
    ▼
Candidate
    │
RequestVote RPC
    │
Majority Votes
    │
    ▼
Leader
```

---

## Log Replication

```text
Client Request
      │
      ▼
Leader
      │
Append to WAL
      │
AppendEntries RPC
      │
Followers Append
      │
Majority Acknowledgement
      │
Commit Entry
      │
Apply to State Machine
```

---

# Package Organization

## cmd/

Contains executable entry points.

* node
* controller

---

## internal/raft

Core consensus implementation.

Responsibilities:

* Node lifecycle
* Leader election
* Heartbeats
* Replication
* State transitions

---

## internal/rpc

Networking layer.

Responsibilities:

* gRPC server
* gRPC client
* Request routing

---

## internal/wal

Persistent storage implementation.

Responsibilities:

* Log segments
* Entry persistence
* Recovery

---

## internal/statemachine

Replicated application state.

Responsibilities:

* Command execution
* Key-value storage

---

## internal/chaos

Fault injection framework.

Responsibilities:

* Simulated failures
* Network manipulation

---

## internal/api

HTTP management endpoints.

Responsibilities:

* Health endpoints
* Cluster information
* Administrative APIs

---

## internal/config

Configuration loading and validation.

---

## internal/logger

Centralized structured logging.

---

## proto/

Protocol Buffer definitions shared across all nodes.

---

## deployments/

Docker and cluster configuration.

---

## scripts/

Development utilities.

---

## docs/

Project documentation.

---

# Design Principles

The project follows several architectural principles.

### Modular Components

Each subsystem has a single responsibility and minimal coupling.

### Interface-Oriented Design

Components communicate through clearly defined interfaces to simplify testing and future extensions.

### Separation of Concerns

Networking, consensus, persistence, and state management remain independent.

### Deterministic State Machine

Every committed command produces identical state across all nodes.

### Failure-Oriented Design

The architecture assumes that nodes, networks, and processes may fail at any time.

---

# Future Extensions

The architecture is intentionally designed to support future enhancements without major refactoring.

Potential additions include:

* Log compaction
* Snapshotting
* Dynamic cluster membership
* TLS-secured communication
* Authentication
* Metrics and distributed tracing
* Web dashboard
* Kubernetes deployment
* Multi-Raft support
* Benchmarking and performance profiling

---

# Summary

RaftLab models the core architecture of a modern distributed consensus system while remaining compact enough for educational purposes. By separating consensus, networking, persistence, state management, and fault injection into independent modules, the project provides a maintainable foundation for exploring distributed systems concepts and implementing additional features over time.
