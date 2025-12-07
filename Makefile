
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-X '$(MODULE)/internal/common.Version=$(VERSION)'"

PROTOC_DEP := $(shell go env GOPATH)/bin/protoc-gen-go

all: build test
.PHONY: all

CMDS := client server
build: $(CMDS:%=bin/%)
.PHONY: build

test: .gen/proto .gen/deps
	@echo -e "\e[32m→\e[0m Executing Tests"
	@go test ./...
.PHONY: test

clean:
	@echo -e "\e[32m→\e[0m Cleanup"
	@rm -rf bin
.PHONY: clean

distclean:
	@echo -e "\e[32m→\e[0m Complete Cleanup"
	@rm -rf bin api .gen
.PHONY: distclean

generate: .gen/proto
.PHONY: generate

install-tools:
	make -B $(PROTOC_DEP)
.PHONY: install-tools

deps: .gen/deps
.PHONY: deps

bin/%: .gen/deps .gen/proto
	@echo -e "\e[32m→\e[0m Building \e[34m$*\e[0m"
	@go build $(LDFLAGS) -o ./$@ ./cmd/$*;
# stupid Make
bin/client: $(shell find pkg api cmd/client -name '*.go' 2> /dev/null)
bin/server: $(shell find pkg api cmd/server -name '*.go' 2> /dev/null)

.gen/proto: $(PROTOC_DEP) $(shell find proto -name '*.proto' 2> /dev/null)
	@echo -e "\e[32m→\e[0m Generating Protobuf"
	@go generate ./...
	@mkdir -p .gen;rm -f .gen/proto
	@touch -r "$$(find api -type f -printf '%T@ %p\n' | sort -k 1gr,1 | head -n1 | cut -d' ' -f2-)" .gen/proto


.gen/deps: go.mod go.sum
	@echo -e "\e[32m→\e[0m Updating Dependencies"
	@go mod download
	@go mod tidy
	@mkdir -p .gen;rm -f .gen/deps
	@touch -r "$$(find $^ -type f -printf '%T@ %p\n' | sort -k 1gr,1 | head -n1 | cut -d' ' -f2-)" .gen/deps

$(PROTOC_DEP):
	@echo -e "\e[32m→\e[0m Installing Tools"
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
