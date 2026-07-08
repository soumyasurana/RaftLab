#!/usr/bin/env bash

set -e

protoc \
  -I proto \
  --go_out=internal/pb \
  --go_opt=paths=source_relative \
  --go-grpc_out=internal/pb \
  --go-grpc_opt=paths=source_relative \
  raft/raft.proto