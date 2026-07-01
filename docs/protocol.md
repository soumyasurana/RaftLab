# Raft Communication Protocol

## Overview

RaftLab nodes communicate using **gRPC** with **Protocol Buffers** as the serialization format. Every node exposes the same RPC interface, allowing any node to participate in leader election, log replication, and cluster coordination.

The communication protocol is based on the Raft consensus algorithm described in *In Search of an Understandable Consensus Algorithm* by Diego Ongaro and John Ousterhout.

---

# Communication Model

All nodes are peers in the cluster.

```text
        Node A
       /  |   \
      /   |    \
 Node B  Node C  Node D
      \   |    /
       \  |   /
        Node E
```

Nodes communicate only through RPCs. Clients interact with the current leader, while followers replicate state through the consensus protocol.

---

# Transport Layer

| Property        | Value            |
| --------------- | ---------------- |
| Protocol        | gRPC             |
| Serialization   | Protocol Buffers |
| Connection Type | Persistent TCP   |
| Communication   | Request–Response |
| Encoding        | Binary           |

---

# RPC Services

The protocol exposes two core RPCs required by the Raft algorithm.

## RequestVote

Used during leader election when a candidate requests votes from other nodes.

### Purpose

* Start a new election
* Elect a leader
* Prevent multiple leaders in the same term

### Request

| Field            | Description                         |
| ---------------- | ----------------------------------- |
| `term`           | Candidate's current term            |
| `candidate_id`   | Candidate requesting the vote       |
| `last_log_index` | Index of candidate's last log entry |
| `last_log_term`  | Term of candidate's last log entry  |

### Response

| Field          | Description                            |
| -------------- | -------------------------------------- |
| `term`         | Receiver's current term                |
| `vote_granted` | Indicates whether the vote was granted |

---

## AppendEntries

Used by the leader to replicate log entries and maintain authority through heartbeats.

### Purpose

* Replicate log entries
* Send heartbeat messages
* Advance commit index
* Maintain leader authority

### Request

| Field            | Description              |
| ---------------- | ------------------------ |
| `term`           | Leader's current term    |
| `leader_id`      | Leader identifier        |
| `prev_log_index` | Previous log index       |
| `prev_log_term`  | Previous log term        |
| `entries`        | Log entries to replicate |
| `leader_commit`  | Leader's commit index    |

### Response

| Field     | Description                   |
| --------- | ----------------------------- |
| `term`    | Receiver's current term       |
| `success` | Indicates replication success |

---

# Message Flow

## Leader Election

```text
Election Timeout
       │
       ▼
Candidate
       │
       ├──────────────► RequestVote
       ├──────────────► RequestVote
       ├──────────────► RequestVote
       │
Majority Votes Received
       │
       ▼
Leader
```

---

## Heartbeat

```text
Leader
   │
AppendEntries (Empty Entries)
   │
   ├────────► Follower
   ├────────► Follower
   └────────► Follower
```

Heartbeats are sent periodically to prevent followers from starting a new election.

---

## Log Replication

```text
Client Command
      │
      ▼
Leader
      │
Append to Local Log
      │
AppendEntries RPC
      │
Followers Persist Entry
      │
Majority Acknowledgement
      │
Commit Entry
      │
Apply to State Machine
```

---

# Log Entry Format

Each replicated log entry contains metadata required for deterministic replication.

| Field     | Description                      |
| --------- | -------------------------------- |
| `index`   | Sequential log index             |
| `term`    | Term when entry was created      |
| `command` | Serialized state machine command |

Entries are applied only after they have been committed by a majority of the cluster.

---

# Node States

A node exists in one of three states.

## Follower

Default state.

Responsibilities:

* Respond to RPCs
* Accept replicated entries
* Vote in elections

---

## Candidate

Entered after an election timeout.

Responsibilities:

* Increment current term
* Vote for itself
* Request votes
* Become leader on majority

---

## Leader

Elected by a majority of nodes.

Responsibilities:

* Accept client requests
* Replicate log entries
* Send heartbeats
* Advance commit index
* Coordinate the cluster

---

# Failure Handling

The protocol is designed to tolerate common failure scenarios.

Supported scenarios include:

* Leader failure
* Follower failure
* Temporary network outages
* Network partitions
* Delayed messages
* Duplicate RPCs
* Node restarts

Consensus is maintained as long as a majority of nodes remain reachable.

---

# Consistency Guarantees

RaftLab follows the guarantees defined by the Raft protocol.

* Election Safety
* Leader Append-Only
* Log Matching
* Leader Completeness
* State Machine Safety

These properties ensure that all healthy nodes eventually converge on the same committed log and replicated state.

---

# Timeouts

The implementation uses configurable timing parameters.

| Parameter          | Purpose                   |
| ------------------ | ------------------------- |
| Election Timeout   | Detect leader failure     |
| Heartbeat Interval | Maintain leader authority |
| RPC Timeout        | Bound request latency     |

Timeout values are loaded from node configuration files and can be tuned for experimentation.

---

# Configuration

Each node is configured independently.

Typical configuration includes:

* Node ID
* Host
* RPC Port
* HTTP Port
* Peer Addresses
* Election Timeout
* Heartbeat Interval
* Data Directory

---

# Future Protocol Extensions

The protocol is intentionally minimal and focuses on the core Raft algorithm. Future versions may introduce:

* Snapshot Installation (`InstallSnapshot` RPC)
* Log Compaction
* Dynamic Cluster Membership
* TLS Encryption
* Authentication
* Request Compression
* Metrics Streaming
* Distributed Tracing

---

# Summary

RaftLab implements the essential communication protocol required for distributed consensus using gRPC and Protocol Buffers. The protocol enables leader election, log replication, heartbeat coordination, and fault tolerance while maintaining a simple, modular design that closely follows the Raft specification.
