You are an expert Go backend architect.

Create ONLY the initial project structure for a production-grade distributed systems project called **RaftLab**.

IMPORTANT:
- Do NOT implement the Raft algorithm.
- Do NOT write business logic.
- Create folders, packages, configuration, protobuf definitions, Docker files, Makefile, and minimal initialization code only.
- Every file should contain only enough code to compile or act as a placeholder.
- Add TODO comments where future implementation belongs.

Project Overview
----------------
RaftLab is an educational distributed systems project implementing the Raft Consensus Algorithm.

Architecture:

- Multiple Go nodes
- gRPC communication
- Protocol Buffers
- Write Ahead Log
- Key-Value State Machine
- Chaos Controller
- HTTP API
- Web Dashboard (created later)
- Docker Compose for local cluster

The repository should be organized as a real production Go project.

=====================================================
Required Folder Structure
=====================================================

raftlab/

├── cmd/
│   ├── node/
│   │   └── main.go
│   └── controller/
│       └── main.go
│
├── internal/
│   ├── raft/
│   │   ├── node.go
│   │   ├── state.go
│   │   ├── election.go
│   │   ├── replication.go
│   │   ├── heartbeat.go
│   │   ├── storage.go
│   │   ├── transport.go
│   │   ├── config.go
│   │   └── errors.go
│   │
│   ├── rpc/
│   │   ├── server.go
│   │   ├── client.go
│   │   └── interceptor.go
│   │
│   ├── wal/
│   │   ├── wal.go
│   │   ├── segment.go
│   │   └── entry.go
│   │
│   ├── statemachine/
│   │   ├── kv.go
│   │   └── apply.go
│   │
│   ├── chaos/
│   │   ├── partition.go
│   │   ├── latency.go
│   │   ├── kill.go
│   │   └── controller.go
│   │
│   ├── api/
│   │   ├── server.go
│   │   ├── routes.go
│   │   └── handlers.go
│   │
│   ├── config/
│   │   └── config.go
│   │
│   └── logger/
│       └── logger.go
│
├── proto/
│   └── raft.proto
│
├── pkg/
│   └── models/
│       ├── log.go
│       ├── rpc.go
│       └── types.go
│
├── deployments/
│   ├── docker-compose.yml
│   ├── Dockerfile
│   └── configs/
│       ├── node1.yaml
│       ├── node2.yaml
│       ├── node3.yaml
│       ├── node4.yaml
│       └── node5.yaml
│
├── scripts/
│   ├── generate_proto.sh
│   ├── run_cluster.sh
│   └── stop_cluster.sh
│
├── test/
│   ├── integration/
│   └── unit/
│
├── docs/
│   ├── architecture.md
│   ├── protocol.md
│   └── roadmap.md
│
├── Makefile
├── go.mod
├── go.sum
├── README.md
├── .gitignore
└── .env.example

=====================================================
Initialization Requirements
=====================================================

1. Initialize Go module.

2. Use latest stable Go version.

3. Install dependencies:

- grpc
- protobuf
- yaml
- zerolog
- cobra (optional)
- uuid

4. Create minimal protobuf service:

service RaftService

RPCs:

RequestVote
AppendEntries

Only message definitions.

No logic.

5. Configure Makefile with commands:

make proto
make build
make run
make test
make fmt
make lint
make clean

6. Dockerfile should compile Go binary.

7. docker-compose should start:

node1
node2
node3
node4
node5

with different config files.

No dashboard.

8. Each node main.go should:

- load config
- initialize logger
- create raft node
- start grpc server
- start http server
- block forever

Only placeholders.

9. Create configuration structs.

Fields:

NodeID
Host
RPCPort
HTTPPort
Peers
ElectionTimeout
HeartbeatInterval
DataDirectory

10. README should include:

- Project overview
- Folder explanation
- Build instructions
- Proto generation
- Running cluster
- Future roadmap

=====================================================
Code Style
=====================================================

- Clean Architecture
- Idiomatic Go
- Small packages
- No unnecessary abstractions
- No implementation beyond placeholders
- Every package should compile
- Include TODO comments indicating future work.

Generate the complete scaffold with all files.