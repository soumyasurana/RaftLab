# RaftLab

> A production-inspired implementation of the Raft Consensus Algorithm in Go with fault injection, distributed log replication, and interactive cluster visualization.

## Overview

RaftLab is an educational distributed systems project that implements the **Raft Consensus Algorithm** from scratch. The project demonstrates how distributed nodes achieve consensus, elect leaders, replicate logs, recover from failures, and maintain a consistent replicated state machine.

Rather than being a toy implementation, RaftLab is structured like a production-grade distributed system with modular packages, gRPC communication, persistent storage, configurable nodes, and a chaos testing framework.

The project is designed to deepen understanding of distributed systems concepts while showcasing backend engineering skills relevant to modern infrastructure and platform teams.

---

## Features

### Consensus

* Leader Election
* Heartbeat Mechanism
* Log Replication
* Commit Index Management
* State Machine Application
* Leader Failover
* Split Vote Recovery
* Term Management

### Distributed Communication

* gRPC-based node-to-node communication
* Protocol Buffers for RPC definitions
* Configurable peer discovery
* Persistent connections between cluster members

### Persistence

* Write-Ahead Log (WAL)
* Persistent log entries
* Node recovery from disk
* Replicated key-value state machine

### Fault Injection

* Kill individual nodes
* Pause and resume nodes
* Simulate network partitions
* Inject network latency
* Simulate packet loss

### Observability

* Structured logging
* HTTP management endpoints
* Cluster health monitoring
* Runtime metrics
* Live cluster status

---

## Architecture

```
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

Each node contains:

* Raft Engine
* gRPC Server
* HTTP Management API
* Write-Ahead Log
* Replicated Key-Value Store

---

## Project Structure

```
cmd/
    node/
    controller/

internal/
    raft/
    rpc/
    wal/
    statemachine/
    chaos/
    api/
    config/
    logger/

pkg/
    models/

proto/

deployments/

scripts/

docs/

test/
```

---

## Technology Stack

| Component        | Technology       |
| ---------------- | ---------------- |
| Language         | Go               |
| RPC              | gRPC             |
| Serialization    | Protocol Buffers |
| Logging          | Zerolog          |
| Configuration    | YAML             |
| Containerization | Docker           |
| Cluster          | Docker Compose   |

---

## Running the Project

### Clone

```bash
git clone <repository-url>

cd raftlab
```

### Generate Protobuf Code

```bash
make proto
```

### Build

```bash
make build
```

### Run Local Cluster

```bash
make run
```

or

```bash
docker compose up
```

---

## Available Commands

```bash
make proto
```

Generate gRPC code.

```bash
make build
```

Build all binaries.

```bash
make run
```

Start the local cluster.

```bash
make test
```

Run unit and integration tests.

```bash
make fmt
```

Format source code.

```bash
make lint
```

Run linters.

```bash
make clean
```

Remove generated artifacts.

---

## Learning Objectives

This project demonstrates practical implementation of:

* Distributed Consensus
* Leader Election
* Replicated State Machines
* Write-Ahead Logging
* Fault Tolerance
* Network Communication
* Concurrent Programming
* Failure Recovery
* Distributed Systems Architecture

---

## Future Improvements

* Snapshotting and Log Compaction
* Dynamic Cluster Membership
* Linearizable Read Optimization
* Persistent Metadata Store
* Web-based Cluster Dashboard
* Metrics and Tracing
* Authentication and TLS
* Benchmarking Suite
* Kubernetes Deployment
* Multi-Raft Support

---

## References

* *In Search of an Understandable Consensus Algorithm (Raft)* — Diego Ongaro & John Ousterhout
* MIT 6.824 Distributed Systems
* etcd
* HashiCorp Consul

---

## License

This project is intended for educational and learning purposes.
