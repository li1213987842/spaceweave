.PHONY: proto build test run-server run-client

proto:
	./scripts/generate_proto.sh

build:
	go build -o bin/server cmd/server/main.go
	go build -o bin/test test/client-test/main.go

unit-test:
	go test ./...

run-server:
	./bin/server

run-test:
	./bin/client-test

benchmark:
	go test -bench=. -benchtime=1x -timeout=120m ./test/bench/benchmark_test.go
