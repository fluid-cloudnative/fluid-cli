# Copyright 2026 The Fluid Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BINARY      ?= fluid
CMD         ?= ./cmd/fluid
BIN_DIR     ?= bin
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE  ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -w -s \
	-X github.com/fluid-cloudnative/fluid-cli/cmd/fluid/internal/version.Version=$(VERSION) \
	-X github.com/fluid-cloudnative/fluid-cli/cmd/fluid/internal/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/fluid-cloudnative/fluid-cli/cmd/fluid/internal/version.BuildDate=$(BUILD_DATE)

GO      ?= go
GOOS    ?= $(shell $(GO) env GOOS)
GOARCH  ?= $(shell $(GO) env GOARCH)

.PHONY: all build fmt vet test install-plugin uninstall-plugin clean help

all: build

## build: Build the fluid binary for the current OS/arch.
build: fmt vet
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD)
	@echo "Built $(BIN_DIR)/$(BINARY)"

## build-linux-amd64: Cross-compile for linux/amd64.
build-linux-amd64:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-amd64 $(CMD)

## build-linux-arm64: Cross-compile for linux/arm64.
build-linux-arm64:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-linux-arm64 $(CMD)

## build-darwin-amd64: Cross-compile for darwin/amd64.
build-darwin-amd64:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-darwin-amd64 $(CMD)

## build-darwin-arm64: Cross-compile for darwin/arm64.
build-darwin-arm64:
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
		$(GO) build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY)-darwin-arm64 $(CMD)

## build-all: Cross-compile for all supported platforms.
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64

## fmt: Run go fmt.
fmt:
	$(GO) fmt ./...

## vet: Run go vet.
vet:
	$(GO) vet ./...

## test: Run unit tests.
test:
	$(GO) test ./... -v -count=1

## install-plugin: Install the binary into the same directory as kubectl (as kubectl-fluid).
install-plugin: build
	@KUBECTL_DIR=$$(dirname $$(which kubectl 2>/dev/null) 2>/dev/null); \
	if [ -z "$$KUBECTL_DIR" ]; then \
		echo "kubectl not found in PATH; installing to /usr/local/bin instead"; \
		KUBECTL_DIR=/usr/local/bin; \
	fi; \
	cp $(BIN_DIR)/$(BINARY) $$KUBECTL_DIR/$(BINARY); \
	echo "Installed $$KUBECTL_DIR/$(BINARY)"; \
	echo "Run: fluid --help"

## uninstall-plugin: Remove the installed plugin binary.
uninstall-plugin:
	@KUBECTL_DIR=$$(dirname $$(which kubectl 2>/dev/null) 2>/dev/null); \
	if [ -z "$$KUBECTL_DIR" ]; then KUBECTL_DIR=/usr/local/bin; fi; \
	rm -f $$KUBECTL_DIR/$(BINARY); \
	echo "Removed $$KUBECTL_DIR/$(BINARY)"

## clean: Remove build artifacts.
clean:
	rm -rf $(BIN_DIR)

## help: Print this help message.
help:
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /'
