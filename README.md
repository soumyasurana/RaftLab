# RaftLab
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-green.svg)
![Status](https://img.shields.io/badge/Status-Active%20Development-orange)
![CI](https://github.com/soumyasurana/RaftLab/actions/workflows/ci.yml/badge.svg)

> A production-inspired implementation of the Raft Consensus Algorithm in Go with fault injection, distributed log replication, and interactive cluster visualization.

## Overview

RaftLab is an educational distributed systems project focused on implementing the **Raft Consensus Algorithm** from scratch in Go.

The project aims to demonstrate how distributed nodes achieve consensus through leader election, log replication, fault tolerance, and replicated state machines while following production-inspired engineering practices.

Rather than being a minimal demonstration, RaftLab is being built as a modular distributed systems laboratory featuring persistent storage, gRPC communication, fault injection, and interactive cluster visualization.

Its primary goal is to deepen understanding of distributed systems while showcasing backend and infrastructure engineering.

---

## Why RaftLab?

RaftLab is designed to bridge the gap between academic explanations of the Raft consensus algorithm and production-grade distributed systems.

Rather than implementing only the core algorithm, the project incorporates engineering practices commonly found in real-world systems, including persistent storage, RPC communication, fault injection, observability, and modular architecture.

---

## Project Status

- ✅ Project architecture
- ✅ Repository organization
- ✅ Configuration system
- ✅ Protocol Buffers
- ✅ gRPC service definitions
- ✅ Write-Ahead Log
- ⏳ State Machine
- ⏳ Raft Node
- ⏳ gRPC Transport
- ⏳ Leader Election
- ⏳ Heartbeats
- ⏳ Log Replication
- ⏳ Commit Index
- ⏳ Snapshots
- ⏳ Chaos Controller
- ⏳ Dashboard

RaftLab is currently under active development.

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

Each node will contain:

- Raft Engine
- gRPC Server
- HTTP Management API
- Write-Ahead Log
- Replicated Key-Value Store

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

### Milestone 1

- Project foundation
- gRPC infrastructure
- Leader election
- Heartbeats

### Milestone 2

- Log replication
- Write-Ahead Log
- State machine
- Crash recovery

### Milestone 3

- Fault injection
- Network partitions
- HTTP management API
- Runtime metrics

### Milestone 4

- Snapshotting
- Dynamic membership
- Cluster visualization
- Benchmarking
- Kubernetes deployment

---

## Learning Objectives

RaftLab explores practical implementation of:

- Distributed Consensus
- Leader Election
- Replicated State Machines
- Write-Ahead Logging
- Fault Tolerance
- Concurrent Programming
- Network Communication
- Failure Recovery
- Distributed Systems Architecture

---

## Future Work

- Snapshot installation
- Joint consensus
- Dynamic membership
- Benchmarking
- Kubernetes deployment
- Distributed tracing
- Prometheus metrics

---

## References

- *In Search of an Understandable Consensus Algorithm (Raft)* — Diego Ongaro & John Ousterhout
- MIT 6.824 Distributed Systems
- etcd
- HashiCorp Consul

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.