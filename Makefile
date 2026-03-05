.PHONY: build test test-race test-acc bench lint fmt vuln

GOPATH_BIN := $(shell go env GOPATH)/bin
PATH := $(GOPATH_BIN):$(PATH)

build:
	go build -o server ./cmd/server/

test:
	go test -v -json ./... | gotestfmt

test-race:
	go test -race -json ./... | gotestfmt

test-acc:
	go test -run ^TestAcceptance -json ./... | gotestfmt

bench:
	go test -bench . -benchmem -json ./... | gotestfmt

lint:
	golangci-lint run ./...

fmt:
	gofumpt -w .

vuln:
	govulncheck ./...
