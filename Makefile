# Project settings
MODULE := hanashite
CMD_DIR := cmd
BIN_DIR := bin

# Version from git or fallback
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")

# All commands (folders in cmd/)
CMDS := $(notdir $(wildcard $(CMD_DIR)/*))

# ldflags for version info
LDFLAGS := -ldflags="-X '$(MODULE)/internal/common.Version=$(VERSION)'"

.PHONY: all clean build test install-tools generate

all: build test

build: $(CMDS:%=bin/%)

test:
	go test ./...

clean:
	rm -rf $(BIN_DIR) $(PROTO_OUT_DIR)


bin/%: $(PB_GO_FILES)
	@echo -e "\033[32m→\033[0m Building $@"; \
	go build $(LDFLAGS) -o ./$@ ./cmd/$*;

# stupid Make
bin/client: $(shell find pkg proto cmd/client -name '*.go' 2> /dev/null)
bin/server: $(shell find pkg proto cmd/server -name '*.go' 2> /dev/null)

generate:
	go generate ./...


install-tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest