# Makefile for dust — common dev + release commands.
#
# Day-to-day:
#   make build              # local binary at ./dust
#   make run                # build + run with no args (TUI)
#   make test               # go test ./...
#   make vet                # go vet ./...
#   make tidy               # go mod tidy
#   make fmt                # gofmt -w
#
# Release flow:
#   make snapshot           # build a fake release locally with goreleaser (no upload)
#   make release-tag VERSION=v0.1.0
#                           # tag + push; GitHub Actions runs goreleaser

PKG := github.com/ariefsn/dust
CMD := ./cmd/dust
BIN := dust

# Version metadata baked into local builds. The release flow uses goreleaser,
# which sets these from git tags + commit + build time.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null)
COMMIT  := $(shell git rev-parse --verify HEAD 2>/dev/null)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Defaults when not in a git repo yet (or before the first commit).
ifeq ($(strip $(VERSION)),)
VERSION := dev
endif
ifeq ($(strip $(COMMIT)),)
COMMIT := none
endif

LDFLAGS := -s -w \
	-X $(PKG)/internal/cli.version=$(VERSION) \
	-X $(PKG)/internal/cli.commit=$(COMMIT) \
	-X $(PKG)/internal/cli.date=$(DATE)

GO ?= go

.PHONY: help build run install test vet tidy fmt clean snapshot release-check release-tag

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build ./dust for the current platform
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD)

build-all: ## Build all four platform binaries into ./build/
	@mkdir -p build
	GOOS=darwin GOARCH=arm64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o build/dust-darwin-arm64 $(CMD)
	GOOS=darwin GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o build/dust-darwin-amd64 $(CMD)
	GOOS=linux  GOARCH=arm64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o build/dust-linux-arm64  $(CMD)
	GOOS=linux  GOARCH=amd64 $(GO) build -trimpath -ldflags "$(LDFLAGS)" -o build/dust-linux-amd64  $(CMD)
	@ls -la build/

run: build ## Build and run dust (TUI)
	./$(BIN)

install: ## Install dust to $GOBIN/$GOPATH/bin
	$(GO) install -trimpath -ldflags "$(LDFLAGS)" $(CMD)

test: ## Run all tests
	$(GO) test ./...

vet: ## go vet ./...
	$(GO) vet ./...

tidy: ## go mod tidy
	$(GO) mod tidy

fmt: ## gofmt -w on every .go file
	gofmt -w .

clean: ## Remove the local binary and dist/
	rm -f $(BIN)
	rm -rf dist/

completions: build ## Generate completion scripts into ./completions/
	mkdir -p completions
	./$(BIN) completion bash       > completions/dust.bash
	./$(BIN) completion zsh        > completions/_dust
	./$(BIN) completion fish       > completions/dust.fish
	./$(BIN) completion powershell > completions/dust.ps1
	@echo "Completions written to ./completions/"

install-completion-omz: build ## Install zsh completion into ~/.oh-my-zsh/completions
	@mkdir -p ~/.oh-my-zsh/completions
	./$(BIN) completion zsh > ~/.oh-my-zsh/completions/_dust
	@echo "Installed. Restart your shell (or run 'exec zsh') to pick it up."

# --- release ---

snapshot: ## Run `goreleaser release --snapshot` locally (no publish)
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed: brew install goreleaser/tap/goreleaser"; exit 1; }
	goreleaser release --snapshot --clean

release-check: ## Validate .goreleaser.yaml + run a local build
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed: brew install goreleaser/tap/goreleaser"; exit 1; }
	goreleaser check
	$(MAKE) snapshot

# Tag + push. GitHub Actions handles the rest (goreleaser builds + uploads).
#   make release-tag VERSION=v0.1.0
release-tag: ## Tag + push a release (VERSION=v0.1.0 required)
	@if [ -z "$(VERSION)" ] || [ "$(VERSION)" = "dev" ]; then \
		echo "VERSION required, e.g. make release-tag VERSION=v0.1.0"; exit 1; \
	fi
	@case "$(VERSION)" in v[0-9]*.[0-9]*.[0-9]*) ;; \
		*) echo "VERSION must look like v0.1.0 (got: $(VERSION))"; exit 1 ;; \
	esac
	@if ! git diff --quiet || ! git diff --cached --quiet; then \
		echo "uncommitted changes — commit or stash first"; exit 1; \
	fi
	git tag -a $(VERSION) -m "release $(VERSION)"
	git push origin $(VERSION)
	@echo
	@echo "Tag pushed. Watch the release at: https://github.com/ariefsn/dust/actions"
