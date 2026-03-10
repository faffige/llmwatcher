VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build run test lint clean

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o bin/llmwatcher ./cmd/llmwatcher

run: build
	./bin/llmwatcher

test:
	go test ./... -race -cover

lint:
	golangci-lint run

clean:
	rm -rf bin/
