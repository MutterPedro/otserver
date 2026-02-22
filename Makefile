.PHONY: build test test-race test-acc bench lint fmt vuln

GOPATH_BIN := $(shell go env GOPATH)/bin
PATH := $(GOPATH_BIN):$(PATH)

build:
	go build -o server ./cmd/server/

test:
	go test -v ./...

test-race:
	go test -race ./...

test-acc:
	go test -run ^TestAcceptance ./...

bench:
	go test -bench . -benchmem ./...

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .

vuln:
	govulncheck ./...
