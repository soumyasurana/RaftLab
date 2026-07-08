# RaftLab
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Status](https://img.shields.io/badge/Status-Active%20Development-orange)
![CI](https://github.com/soumyasurana/RaftLab/actions/workflows/ci.yml/badge.svg)

> A production-inspired implementation of the Raft Consensus Algorithm in Go with fault injection, distributed log replication, and interactive cluster visualization.

## Overview

RaftLab is an educational distributed systems project that implements the **Raft Consensus Algorithm** from scratch in Go — bridging the gap between academic explanations of Raft and the engineering practices found in production systems.

Rather than a minimal demonstration, RaftLab is built as a modular distributed systems laboratory: persistent storage, gRPC communication, fault injection, and interactive cluster visualization, all developed incrementally with production-inspired practices like structured logging, observability, and modular architecture.

Its primary goal is to deepen understanding of distributed systems while showcasing backend and infrastructure engineering.

---

## Getting Started

> Note: RaftLab is under active development. Instructions below reflect the current state of the build; some commands will change as features land.

### Prerequisites

- Go 1.25+
- Docker & Docker Compose (for running a local multi-node cluster)
- `protoc` (Protocol Buffers compiler), if regenerating gRPC code

### Clone and build

```bash
git clone https://github.com/soumyasurana/RaftLab.git
cd RaftLab
go build ./...
```

### Run tests

```bash
go test ./...
```

### Run a local cluster

```bash
docker compose -f deployments/docker-compose.yml up
```

_(Cluster bring-up is in progress — this section will be updated with node CLI flags and a walkthrough once the Raft Node and gRPC Transport milestones are complete.)_

---

## Project Status

**Legend:** ✅ Done &nbsp;·&nbsp; ⏳ In Progress

- ✅ Project architecture
- ✅ Repository organization
- ✅ Configuration system
- ✅ Protocol Buffers
- ✅ gRPC service definitions
- ✅ Write-Ahead Log
- ✅ State Machine
- ⏳ Raft Node
- ⏳ gRPC Transport
- ⏳ Leader Election
- ⏳ Heartbeats
- ⏳ Log Replication
- ⏳ Commit Index
- ⏳ Snapshots
- ⏳ Chaos Controller
- ⏳ Dashboard

---

## Planned Features

### Consensus

- Leader Election
- Heartbeat Mechanism
- Log Replication
- Commit Index Management
- State Machine Application
- Leader Failover
- Split Vote Recovery
- Term Management

### Distributed Communication

- gRPC-based node-to-node communication
- Protocol Buffers for RPC definitions
- Configurable peer discovery
- Persistent peer connections

### Persistence

- Write-Ahead Log (WAL)
- Persistent log storage
- Node recovery
- Replicated key-value state machine

### Fault Injection

- Kill individual nodes
- Pause and resume nodes
- Network partitions
- Artificial latency
- Packet loss simulation

### Observability

- Structured logging
- HTTP management API
- Cluster health monitoring
- Runtime metrics
- Live cluster visualization

---

## Target Architecture

### Cluster Topology

```text
                +----------------------+
                |   Chaos Controller   |
                +----------+-----------+
                           |
     -----------------------------------------------
     |             |            |           |       |
+---------+  +---------+  +---------+  +---------+  +---------+
| Node 1  |  | Node 2  |  | Node 3  |  | Node 4  |  | Node 5  |
+----+----+  +----+----+  +----+----+  +----+----+  +----+----+
     |              |             |             |            |
     +--------------+-------------+-------------+------------+
                    gRPC Communication Network
```

### Single Node Internals

```text
+-------------------------------------------+
|                   Node                     |
|                                             |
|  +-----------+       +------------------+  |
|  | gRPC      | <---> |  Raft Engine     |  |
|  | Server    |       |  (Election,      |  |
|  +-----------+       |   Replication,   |  |
|                       |   Commit Index)  |  |
|  +-----------+       +--------+---------+  |
|  | HTTP Mgmt |                |            |
|  | API       |                v            |
|  +-----------+       +------------------+  |
|                       | Write-Ahead Log  |  |
|                       +--------+---------+  |
|                                |            |
|                                v            |
|                       +------------------+  |
|                       | Replicated KV    |  |
|                       | State Machine    |  |
|                       +------------------+  |
+-------------------------------------------+
```

---

## Project Structure

```text
cmd/
    controller/
    node/

internal/
    api/
    chaos/
    config/
    logger/
    raft/
    rpc/
    statemachine/
    storage/
        wal/

pkg/
    types/

proto/
    raft/

deployments/
    configs/

scripts/

tests/

docs/
```

---

## Technology Stack

| Component | Technology |
|----------|------------|
| Language | Go |
| RPC | gRPC |
| Serialization | Protocol Buffers |
| Logging | Zerolog |
| Configuration | YAML |
| Containerization | Docker |
| Local Cluster | Docker Compose |

---

## Roadmap

### Milestone 1 — Foundation
- Project foundation
- gRPC infrastructure
- Leader election
- Heartbeats

### Milestone 2 — Durability
- Log replication
- Write-Ahead Log
- State machine
- Crash recovery

### Milestone 3 — Resilience
- Fault injection
- Network partitions
- HTTP management API
- Runtime metrics

### Milestone 4 — Scale & Visibility
- Snapshotting
- Dynamic membership
- Cluster visualization
- Benchmarking
- Kubernetes deployment

### Stretch Goals
- Snapshot installation via gRPC streaming
- Joint consensus for membership changes
- Distributed tracing (OpenTelemetry)
- Prometheus metrics export

---

## Design Decisions

A few of the "why" behind early architectural choices — more will be added as the project matures.

- **Why gRPC over raw TCP/JSON-RPC?** Protocol Buffers give strongly-typed, versioned RPC contracts and built-in streaming, which matters for snapshot transfer and log replication at scale.
- **Why a Write-Ahead Log before the state machine?** Matches Raft's own durability guarantee — entries must be persisted before they're considered committed, so the WAL is the source of truth on recovery, not the in-memory state machine.
- **Why Zerolog?** Structured, low-allocation logging that's cheap enough to leave on in hot paths like heartbeats and replication, which is important for later observability work.

---

## Learning Objectives

RaftLab is a hands-on exploration of distributed consensus, replicated state machines, write-ahead logging, and failure recovery — implemented with an emphasis on concurrent programming and network communication patterns used in real production systems.

---

## References

- [In Search of an Understandable Consensus Algorithm (Raft)](https://raft.github.io/raft.pdf) — Diego Ongaro & John Ousterhout
- [MIT 6.824: Distributed Systems](https://pdos.csail.mit.edu/6.824/)
- [etcd](https://github.com/etcd-io/etcd)
- [HashiCorp Consul](https://github.com/hashicorp/consul)

---

## Contributing

RaftLab is primarily a personal learning project, but issues, suggestions, and PRs are welcome — especially around test coverage, edge cases in the Raft protocol, or documentation improvements. Please open an issue before submitting large changes so we can discuss approach first.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.