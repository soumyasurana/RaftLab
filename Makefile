generate:
	bash scripts/generate_proto.sh

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
