GO ?= go
INSTALL_DIR ?= /usr/local/bin
SYSTEMD_DIR ?= /etc/systemd/system
BIN_DIR ?= bin
CLANG ?= clang
LLVM_STRIP ?= llvm-strip

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

BPF_SRC := internal/drift/bpf/drift_l4.c
BPF_OBJ := internal/drift/bpf/bin/drift_l4.bpf.o
BPF_DIR := $(patsubst %/,%,$(dir $(BPF_SRC)))
VMLINUX_H := $(BPF_DIR)/vmlinux.h

ifeq ($(UNAME_M),x86_64)
BPF_ARCH_DEF ?= -D__TARGET_ARCH_x86
else ifeq ($(UNAME_M),aarch64)
BPF_ARCH_DEF ?= -D__TARGET_ARCH_arm64
else ifeq ($(UNAME_M),arm64)
BPF_ARCH_DEF ?= -D__TARGET_ARCH_arm64
else ifeq ($(UNAME_M),ppc64le)
BPF_ARCH_DEF ?= -D__TARGET_ARCH_powerpc
else
BPF_ARCH_DEF ?=
endif

BPF_CFLAGS ?= -O2 -g -target bpf $(BPF_ARCH_DEF)
LIBBPF_SRC_DIR := /root/volant/libbpf/src
BTF_SOURCE ?= /sys/kernel/btf/vmlinux
BPFTOOL ?= bpftool
BPF_CINCLUDES ?= -I/usr/include/bpf -I$(LIBBPF_SRC_DIR)
BPF_CINCLUDES += -I$(BPF_DIR)


.PHONY: help
help: ## Show available make targets
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*##"} {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: build-server build-agent build-cli build-drift ## Build all core binaries into $(BIN_DIR)

.PHONY: build-drift-bpf
build-drift-bpf: ## Build the Drift eBPF object (Linux only)
ifeq ($(UNAME_S),Linux)
build-drift-bpf: $(VMLINUX_H)
	mkdir -p $(dir $(BPF_OBJ))
	$(CLANG) $(BPF_CFLAGS) $(BPF_CINCLUDES) -c $(BPF_SRC) -o $(BPF_OBJ)
	-$(LLVM_STRIP) -g $(BPF_OBJ)
else
	@echo "build-drift-bpf skipped: requires Linux (current: $(UNAME_S))"
endif

$(VMLINUX_H):
ifeq ($(UNAME_S),Linux)
	@mkdir -p $(BPF_DIR)
	@if ! command -v $(BPFTOOL) >/dev/null 2>&1; then \
			echo "Error: bpftool not found in PATH. Install bpftool to build drift BPF programs." >&2; \
			exit 1; \
		fi
	@if [ ! -r "$(BTF_SOURCE)" ]; then \
			echo "Error: BTF source '$(BTF_SOURCE)' not found or unreadable. Set BTF_SOURCE=/path/to/vmlinux." >&2; \
			exit 1; \
		fi
	$(BPFTOOL) btf dump file $(BTF_SOURCE) format c > $@
else
	@echo "vmlinux.h generation skipped on non-Linux host (current: $(UNAME_S))"
endif

.PHONY: build-server
build-server: ## Build the volantd binary
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/volantd ./cmd/volantd

.PHONY: build-agent
build-agent: ## Build the kestrel agent binary
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/kestrel ./cmd/kestrel

.PHONY: build-cli
build-cli: ## Build the volar CLI binary
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/volar ./cmd/volar

.PHONY: build-drift
ifeq ($(UNAME_S),Linux)
build-drift: build-drift-bpf
endif
build-drift: ## Build the driftd control daemon
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/driftd ./cmd/driftd
	@if [ -f "$(BPF_OBJ)" ]; then cp "$(BPF_OBJ)" "$(BIN_DIR)/drift_l4.bpf.o"; fi

.PHONY: build-openapi-export
build-openapi-export: ## Build the openapi-export utility
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/openapi-export ./cmd/openapi-export

.PHONY: openapi-export
openapi-export: build-openapi-export ## Generate OpenAPI JSON to docs/api-reference/openapi.json
\t$(BIN_DIR)/openapi-export -server https://docs.volantvm.com -output docs/api-reference/openapi.json

.PHONY: install
install: build ## Install core binaries into INSTALL_DIR (default: /usr/local/bin)
	mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BIN_DIR)/volantd $(INSTALL_DIR)/volantd
	install -m 0755 $(BIN_DIR)/kestrel $(INSTALL_DIR)/kestrel
	install -m 0755 $(BIN_DIR)/volar $(INSTALL_DIR)/volar

.PHONY: install-drift
install-drift: build-drift ## Install driftd binary, BPF object, and systemd unit
	mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BIN_DIR)/driftd $(INSTALL_DIR)/driftd
	@if [ -f "$(BIN_DIR)/drift_l4.bpf.o" ]; then install -m 0644 "$(BIN_DIR)/drift_l4.bpf.o" "$(INSTALL_DIR)/drift_l4.bpf.o"; fi
	mkdir -p $(SYSTEMD_DIR)
	install -m 0644 build/systemd/driftd.service $(SYSTEMD_DIR)/driftd.service

.PHONY: test
test: ## Run unit tests
	$(GO) test ./...

.PHONY: fmt
fmt: ## Format Go sources
	$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: ci
ci: fmt vet test

.PHONY: tidy
tidy: ## Sync go.mod
	$(GO) mod tidy

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR)
	rm -rf $(dir $(BPF_OBJ))
