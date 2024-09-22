.PHONY: proto build test run-server run-client

proto:
	./scripts/generate_proto.sh

build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/test test/main.go

unit-test:
	go test ./...

run-server:
	./bin/server

run-test:
	./bin/test