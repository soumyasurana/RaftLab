.PHONY: proto build run test fmt lint clean

proto:
	./scripts/generate_proto.sh

build:
	go build -o bin/node ./cmd/node
	go build -o bin/controller ./cmd/controller

run:
	./scripts/run_cluster.sh

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
	./scripts/stop_cluster.sh
