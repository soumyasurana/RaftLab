#!/bin/bash
# TODO: implement protoc command
echo "Generating proto files..."
protoc --go_out=. --go-grpc_out=. proto/raft.proto
