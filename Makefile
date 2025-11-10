# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Volant Build System
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#
# Volant is a unified single-binary VM platform with the following components:
#
# Core Binaries:
#   • volant      - CLI with embedded BuildKit builder (NO Docker daemon!)
#   • volantd     - Control plane server/daemon
#   • kestrel     - In-VM guest agent
#   • driftd      - Network control daemon with eBPF
#
# The volant CLI includes a native builder that requires (Linux only):
#   • skopeo      - OCI image manipulation
#   • umoci       - OCI layer unpacking
#   • mksquashfs  - Squashfs compression
#
# Quick start:
#   make build              - Build all core binaries
#   make show-info          - Show build environment
#   make check-builder-deps - Check native builder dependencies
#   make install            - Install binaries to /usr/local/bin
#
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

GO ?= go
INSTALL_DIR ?= /usr/local/bin
SYSTEMD_DIR ?= /etc/systemd/system
BIN_DIR ?= bin
CLANG ?= clang
LLVM_STRIP ?= llvm-strip

# Volant native builder dependencies (Linux only)
SKOPEO ?= skopeo
UMOCI ?= umoci
MKSQUASHFS ?= mksquashfs

UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

BPF_DIR := internal/drift/bpf
BPF_INGRESS_SRC := $(BPF_DIR)/drift_l4_ingress.c
BPF_EGRESS_SRC := $(BPF_DIR)/drift_l4_egress.c
BPF_INGRESS_OBJ := $(BPF_DIR)/bin/drift_l4_ingress.bpf.o
BPF_EGRESS_OBJ := $(BPF_DIR)/bin/drift_l4_egress.bpf.o
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

.PHONY: all
all: build ## Build all core binaries (alias for 'build')

.PHONY: build
build: build-server build-agent build-cli build-drift ## Build all core binaries into $(BIN_DIR)

.PHONY: build-drift-bpf
build-drift-bpf: ## Build the Drift eBPF objects (Linux only)
ifeq ($(UNAME_S),Linux)
build-drift-bpf: $(VMLINUX_H)
	mkdir -p $(BPF_DIR)/bin
	$(CLANG) $(BPF_CFLAGS) $(BPF_CINCLUDES) -c $(BPF_INGRESS_SRC) -o $(BPF_INGRESS_OBJ)
	$(CLANG) $(BPF_CFLAGS) $(BPF_CINCLUDES) -c $(BPF_EGRESS_SRC) -o $(BPF_EGRESS_OBJ)
	-$(LLVM_STRIP) -g $(BPF_INGRESS_OBJ)
	-$(LLVM_STRIP) -g $(BPF_EGRESS_OBJ)
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
build-cli: ## Build the volant CLI binary (includes embedded BuildKit builder)
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/volant ./cmd/volant

.PHONY: build-compose
build-compose: ## Build the volant-compose CLI binary
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/volant-compose ./cmd/volant-compose

.PHONY: build-drift
ifeq ($(UNAME_S),Linux)
build-drift: build-drift-bpf
endif
build-drift: ## Build the driftd control daemon
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/driftd ./cmd/driftd
	@if [ -f "$(BPF_INGRESS_OBJ)" ]; then cp "$(BPF_INGRESS_OBJ)" "$(BIN_DIR)/drift_l4_ingress.bpf.o"; fi
	@if [ -f "$(BPF_EGRESS_OBJ)" ]; then cp "$(BPF_EGRESS_OBJ)" "$(BIN_DIR)/drift_l4_egress.bpf.o"; fi

.PHONY: build-openapi-export
build-openapi-export: ## Build the openapi-export utility
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/openapi-export ./cmd/openapi-export

.PHONY: openapi-export
openapi-export: build-openapi-export ## Generate OpenAPI JSON to docs/6_reference/api/openapi.json
\t$(BIN_DIR)/openapi-export -server https://docs.volantvm.com -output docs/6_reference/api/openapi.json

.PHONY: check-builder-deps
check-builder-deps: ## Check if native builder dependencies are installed (Linux only)
ifeq ($(UNAME_S),Linux)
	@echo "Checking native builder dependencies..."
	@command -v $(SKOPEO) >/dev/null 2>&1 && echo "✓ skopeo found" || echo "✗ skopeo not found (install: apt install skopeo)"
	@command -v $(UMOCI) >/dev/null 2>&1 && echo "✓ umoci found" || echo "✗ umoci not found (install from: github.com/opencontainers/umoci)"
	@command -v $(MKSQUASHFS) >/dev/null 2>&1 && echo "✓ mksquashfs found" || echo "✗ mksquashfs not found (install: apt install squashfs-tools)"
	@echo ""
	@echo "Native builder requires: skopeo, umoci, mksquashfs"
	@echo "See PHASE2_NATIVE_BUILDER.md for installation instructions"
else
	@echo "Native builder is only available on Linux (current: $(UNAME_S))"
	@echo "The volant CLI will build successfully but 'volant build' will show an error"
endif

.PHONY: show-info
show-info: ## Show build environment information
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "🔧 Volant Build Environment"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "Platform:     $(UNAME_S)"
	@echo "Architecture: $(UNAME_M)"
	@echo "Go:           $$($(GO) version)"
	@echo "Install dir:  $(INSTALL_DIR)"
	@echo "Binary dir:   $(BIN_DIR)"
	@echo ""
	@echo "Native Builder (Linux only):"
ifeq ($(UNAME_S),Linux)
	@echo "  Status:     Available"
	@echo "  Skopeo:     $$(command -v $(SKOPEO) >/dev/null 2>&1 && echo 'installed' || echo 'NOT INSTALLED')"
	@echo "  Umoci:      $$(command -v $(UMOCI) >/dev/null 2>&1 && echo 'installed' || echo 'NOT INSTALLED')"
	@echo "  Mksquashfs: $$(command -v $(MKSQUASHFS) >/dev/null 2>&1 && echo 'installed' || echo 'NOT INSTALLED')"
else
	@echo "  Status:     Not available (requires Linux)"
endif
	@echo ""
	@echo "Run 'make check-builder-deps' to verify builder dependencies"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

.PHONY: install
install: build build-drift ## Install core binaries and drift into INSTALL_DIR (default: /usr/local/bin)
	mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BIN_DIR)/volantd $(INSTALL_DIR)/volantd
	install -m 0755 $(BIN_DIR)/kestrel $(INSTALL_DIR)/kestrel
	install -m 0755 $(BIN_DIR)/volant $(INSTALL_DIR)/volant
	install -m 0755 $(BIN_DIR)/driftd $(INSTALL_DIR)/driftd
	@if [ -f "$(BIN_DIR)/drift_l4_ingress.bpf.o" ]; then install -m 0644 "$(BIN_DIR)/drift_l4_ingress.bpf.o" "$(INSTALL_DIR)/drift_l4_ingress.bpf.o"; fi
	@if [ -f "$(BIN_DIR)/drift_l4_egress.bpf.o" ]; then install -m 0644 "$(BIN_DIR)/drift_l4_egress.bpf.o" "$(INSTALL_DIR)/drift_l4_egress.bpf.o"; fi

.PHONY: install-drift
install-drift: build-drift ## Install driftd binary, BPF objects, and systemd unit
	mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BIN_DIR)/driftd $(INSTALL_DIR)/driftd
	@rm -f "$(INSTALL_DIR)/drift_l4.bpf.o" "$(INSTALL_DIR)/drift_l4_xdp.bpf.o"  # Remove old BPF objects
	@if [ -f "$(BIN_DIR)/drift_l4_ingress.bpf.o" ]; then install -m 0644 "$(BIN_DIR)/drift_l4_ingress.bpf.o" "$(INSTALL_DIR)/drift_l4_ingress.bpf.o"; fi
	@if [ -f "$(BIN_DIR)/drift_l4_egress.bpf.o" ]; then install -m 0644 "$(BIN_DIR)/drift_l4_egress.bpf.o" "$(INSTALL_DIR)/drift_l4_egress.bpf.o"; fi
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
clean: ## Remove build artifacts (binaries and BPF objects)
	rm -rf $(BIN_DIR)
	rm -f $(BPF_INGRESS_OBJ) $(BPF_EGRESS_OBJ)
	rm -f $(VMLINUX_H)
	@echo "✓ Build artifacts cleaned"
	@echo "Note: Native builder cache is preserved (see 'make clean-builder-cache')"

.PHONY: clean-builder-cache
clean-builder-cache: ## Remove native builder cache (BuildKit state)
	@echo "Removing native builder cache..."
	@if [ -n "$${VOLANT_BUILDKIT_STATE_DIR}" ] && [ -d "$${VOLANT_BUILDKIT_STATE_DIR}" ]; then \
		rm -rf "$${VOLANT_BUILDKIT_STATE_DIR}"; \
		echo "✓ Removed cache from: $${VOLANT_BUILDKIT_STATE_DIR}"; \
	elif [ -d "$${XDG_CACHE_HOME:-$$HOME/.cache}/volant/buildkit" ]; then \
		rm -rf "$${XDG_CACHE_HOME:-$$HOME/.cache}/volant/buildkit"; \
		echo "✓ Removed cache from: $${XDG_CACHE_HOME:-$$HOME/.cache}/volant/buildkit"; \
	elif [ -d "/tmp/volant-buildkit" ]; then \
		rm -rf /tmp/volant-buildkit; \
		echo "✓ Removed cache from: /tmp/volant-buildkit"; \
	else \
		echo "No builder cache found"; \
	fi

.PHONY: clean-all
clean-all: clean clean-builder-cache ## Remove all build artifacts and caches
