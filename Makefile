PROTO_DIR=proto/raft

generate:
	protoc \
		--go_out=. \
		--go-grpc_out=. \
		$(PROTO_DIR)/raft.proto

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./...

build:
	go build ./...

lint: fmt vet test

all: generate build test