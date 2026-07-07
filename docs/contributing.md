# Contributing to RaftLab

Thank you for your interest in **RaftLab**.

## Requirements

Before getting started, ensure you have the following installed:

* Go 1.25 or later
* Protocol Buffers (`protoc`)
* `protoc-gen-go`
* `protoc-gen-go-grpc`

## Setup

Clone the repository and download the project dependencies:

```bash
go mod download
```

Generate the protobuf files:

```bash
./scripts/generate_proto.sh
```

Run the test suite:

```bash
go test ./...
```

Build the project:

```bash
go build ./...
```

## Code Style

Please follow these guidelines when contributing:

* Follow standard Go formatting conventions.
* Keep packages small and focused on a single responsibility.
* Write unit tests for new functionality whenever possible.
* Run `go fmt ./...` before committing.
* Ensure `go vet ./...` and `go test ./...` pass before opening a pull request.
