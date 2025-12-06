# Project settings
MODULE := hanashite

# Version from git or fallback
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")

# All commands (folders in cmd/)
CMDS := $(notdir $(wildcard cmd/*))

# ldflags for version info
LDFLAGS := -ldflags="-X '$(MODULE)/internal/common.Version=$(VERSION)'"

.PHONY: all clean distclean build test generate install-tools deps

all: build test

build: $(CMDS:%=bin/%)

test: .gen/proto .gen/deps
	go test ./...

clean:
	rm -rf bin

distclean:
	rm -rf bin api .gen

generate: .gen/proto

install-tools: .gen/tools

deps: .gen/deps

bin/%: .gen/deps .gen/proto
	@echo -e "\033[32m→\033[0m Building $@"; \
	go build $(LDFLAGS) -o ./$@ ./cmd/$*;

# stupid Make
bin/client: $(shell find pkg api cmd/client -name '*.go' 2> /dev/null)
bin/server: $(shell find pkg api cmd/server -name '*.go' 2> /dev/null)

.gen/proto: .gen/tools $(shell find proto -name '*.proto' 2> /dev/null)
	go generate ./...
	@mkdir -p .gen
	@date >.gen/proto

.gen/deps: go.mod go.sum
	go mod download
	go mod tidy
	@mkdir -p .gen
	@date >.gen/deps

.gen/tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@mkdir -p .gen
	@date >.gen/tools