# Version from git or fallback
# All commands (folders in cmd/)

.ONESHELL:
.SHELLFLAGS := -Eeuo pipefail -c

define track_file
	trap 'rm -f $(1)' ERR
	mkdir -p .gen
	date >$(1)
endef

# ldflags for version info
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X '$(MODULE)/internal/common.Version=$(VERSION)'"


all: build test
.PHONY: all

CMDS := $(notdir $(wildcard cmd/*))
build: $(CMDS:%=bin/%)
.PHONY: build


test: .gen/proto .gen/deps
	go test ./...
.PHONY: test

clean:
	rm -rf bin
.PHONY: clean

distclean: clean
	rm -rf .gen
.PHONY: distclean


generate: .gen/proto
.PHONY: generate

install-tools: .gen/tools
.PHONY: install-tools

deps: .gen/deps
.PHONY: deps

bin/%: .gen/deps .gen/proto
	go build $(LDFLAGS) -o ./$@ ./cmd/$*;

# stupid Make
bin/client: $(shell find pkg api cmd/client -name '*.go' 2> /dev/null)
bin/server: $(shell find pkg api cmd/server -name '*.go' 2> /dev/null)

.gen/proto: .gen/tools $(shell find proto -name '*.proto' 2> /dev/null)
	$(call track_file, $@)
	go generate ./...

.gen/deps: go.mod go.sum
	$(call track_file, $@)
	go mod download
	go mod tidy

.gen/tools:
	$(call track_file, $@)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
