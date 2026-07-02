SHELL := /bin/sh
.DEFAULT_GOAL := help

.PHONY: help deps dev-setup fmt fmt-check tidy-check vet lint security validate \
	test test-race test-coverage test-integration test-all build build-debug \
	build-binaries build-all checksums verify-artifacts clean check ci \
	check-git-clean check-release-tag pre-release-checks release \
	release-snapshot release-dry-run tag install install-local uninstall \
	docker docker-run docker-shell docker-push docker-clean version info \
	update-deps outdated

MODULE := github.com/justjundana/govman
MAIN_PACKAGE := ./cmd/govman
BINARY_NAME := govman
BUILD_DIR := build
DIST_DIR := dist
COVERAGE_DIR := coverage

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || printf '%s' v0.0.0-dev)
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || printf '%s' none)
DATE ?= $(shell git show -s --format=%cI HEAD 2>/dev/null || printf '%s' unknown)
BUILD_BY ?= make
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0
BUILD_TAGS ?= netgo
GOFMT ?= $(shell go env GOROOT)/bin/gofmt

LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.Date=$(DATE) \
	-X $(MODULE)/internal/version.BuildBy=$(BUILD_BY)

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 \
	windows/amd64 windows/arm64 windows/386
RELEASE_ASSETS := \
	govman-linux-amd64 \
	govman-linux-arm64 \
	govman-darwin-amd64 \
	govman-darwin-arm64 \
	govman-windows-amd64.exe \
	govman-windows-arm64.exe \
	govman-windows-386.exe

GOLANGCI_LINT_VERSION := v2.11.4
STATICCHECK_VERSION := 2026.1
GOSEC_VERSION := v2.24.7
GOVULNCHECK_VERSION := v1.1.4
GORELEASER_VERSION := v2.16.0

help: ## Show available targets
	@printf 'govman build targets (version %s)\n\n' '$(VERSION)'
	@awk 'BEGIN {FS = ":.*?## "}; /^[a-zA-Z0-9_-]+:.*?## / {printf "  %-22s %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

deps: ## Download and verify dependencies without changing module files
	go mod download
	go mod verify

dev-setup: deps ## Install pinned development tools
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	go install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)

fmt: ## Format Go source files
	$(GOFMT) -w $$(find . -type f -name '*.go' -not -path './vendor/*')

fmt-check: ## Check Go formatting without modifying files
	@files=$$($(GOFMT) -l $$(find . -type f -name '*.go' -not -path './vendor/*')); \
	if [ -n "$$files" ]; then printf 'Go files need formatting:\n%s\n' "$$files" >&2; exit 1; fi

tidy-check: ## Check go.mod and go.sum without modifying files
	go mod tidy -diff

vet: ## Run go vet
	go vet ./...

lint: ## Run pinned static-analysis tools already installed on PATH
	@command -v golangci-lint >/dev/null || { echo 'golangci-lint is required; run make dev-setup' >&2; exit 1; }
	@command -v staticcheck >/dev/null || { echo 'staticcheck is required; run make dev-setup' >&2; exit 1; }
	golangci-lint run --timeout=5m
	staticcheck ./...

security: ## Run pinned security tools already installed on PATH
	@command -v gosec >/dev/null || { echo 'gosec is required; run make dev-setup' >&2; exit 1; }
	@command -v govulncheck >/dev/null || { echo 'govulncheck is required; run make dev-setup' >&2; exit 1; }
	gosec -quiet ./...
	govulncheck ./...

validate: fmt-check tidy-check vet ## Run read-only source validation

test: ## Run unit tests
	go test -timeout=10m -tags=$(BUILD_TAGS) ./...

test-race: ## Run unit tests with the race detector
	go test -race -timeout=10m -tags=$(BUILD_TAGS) ./...

test-coverage: ## Generate an HTML coverage report
	mkdir -p $(COVERAGE_DIR)
	go test -timeout=10m -tags=$(BUILD_TAGS) -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o $(COVERAGE_DIR)/coverage.html
	go tool cover -func=coverage.out | tail -n 1

test-integration: ## Run maintained installer integration tests
	bash test/installers.sh
	@if command -v pwsh >/dev/null 2>&1; then pwsh -NoProfile -File test/windows-path.ps1; fi

test-all: test test-race test-integration ## Run all local tests

build: ## Build for GOOS/GOARCH
	@set -eu; \
	mkdir -p '$(BUILD_DIR)'; \
	suffix=''; if [ '$(GOOS)' = windows ]; then suffix='.exe'; fi; \
	CGO_ENABLED='$(CGO_ENABLED)' GOOS='$(GOOS)' GOARCH='$(GOARCH)' \
		go build -trimpath -tags='$(BUILD_TAGS)' -ldflags "$(LDFLAGS)" \
		-o '$(BUILD_DIR)/$(BINARY_NAME)'"$$suffix" '$(MAIN_PACKAGE)'

build-debug: ## Build a native binary with debug symbols
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build -gcflags='all=-N -l' -tags='$(BUILD_TAGS)' \
		-o $(BUILD_DIR)/$(BINARY_NAME)-debug $(MAIN_PACKAGE)

build-binaries: ## Build every supported raw release asset, failing on first error
	@set -eu; \
	rm -rf '$(DIST_DIR)'; \
	mkdir -p '$(DIST_DIR)'; \
	for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		asset='$(BINARY_NAME)-'"$$os"'-'"$$arch"; \
		if [ "$$os" = windows ]; then asset="$$asset.exe"; fi; \
		printf 'Building %s -> %s\n' "$$platform" "$$asset"; \
		CGO_ENABLED=0 GOOS="$$os" GOARCH="$$arch" \
			go build -trimpath -tags='$(BUILD_TAGS)' -ldflags "$(LDFLAGS)" \
			-o '$(DIST_DIR)'/"$$asset" '$(MAIN_PACKAGE)'; \
	done

build-all: build-binaries ## Alias for build-binaries

checksums: ## Create a sorted SHA-256 manifest for final release assets
	@set -eu; \
	cd '$(DIST_DIR)'; \
	: > checksums.txt.tmp; \
	for asset in $(RELEASE_ASSETS); do \
		[ -f "$$asset" ]; \
		if command -v sha256sum >/dev/null 2>&1; then \
			sha256sum "$$asset" >> checksums.txt.tmp; \
		else \
			shasum -a 256 "$$asset" >> checksums.txt.tmp; \
		fi; \
	done; \
	LC_ALL=C sort checksums.txt.tmp > checksums.txt; \
	rm checksums.txt.tmp

verify-artifacts: ## Verify the exact release asset set and checksum manifest
	@set -eu; \
	[ -s '$(DIST_DIR)/checksums.txt' ]; \
	for asset in $(RELEASE_ASSETS); do \
		[ -s '$(DIST_DIR)'/"$$asset" ] || { echo "missing release asset: $$asset" >&2; exit 1; }; \
		grep -Eq "^[0-9a-fA-F]{64}  $$asset$$" '$(DIST_DIR)/checksums.txt' || { echo "missing checksum: $$asset" >&2; exit 1; }; \
	done; \
	count=$$(find '$(DIST_DIR)' -maxdepth 1 -type f ! -name checksums.txt | wc -l | tr -d ' '); \
	[ "$$count" -eq 7 ] || { echo "unexpected release asset count: $$count" >&2; exit 1; }

check-git-clean: ## Require a clean working tree
	@set -eu; \
	if [ -n "$$(git status --porcelain)" ]; then git status --short >&2; echo 'working tree must be clean' >&2; exit 1; fi

check-release-tag: ## Require an annotated SemVer tag pointing at HEAD
	@set -eu; \
	tag=$$(git describe --tags --exact-match HEAD 2>/dev/null) || { echo 'HEAD must have a release tag' >&2; exit 1; }; \
	printf '%s\n' "$$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)[0-9]*)?$$' || { echo "invalid release tag: $$tag" >&2; exit 1; }; \
	[ "$$(git cat-file -t refs/tags/$$tag)" = tag ] || { echo "release tag must be annotated: $$tag" >&2; exit 1; }; \
	[ "$$(git rev-list -n 1 "$$tag")" = "$$(git rev-parse HEAD)" ] || { echo 'release tag does not point at HEAD' >&2; exit 1; }

pre-release-checks: check-git-clean check-release-tag validate test test-race test-integration ## Run release gates

release: pre-release-checks ## Publish a release with GoReleaser
	@command -v goreleaser >/dev/null || { echo 'goreleaser is required; run make dev-setup' >&2; exit 1; }
	goreleaser release --clean

release-snapshot: ## Build a non-publishing GoReleaser snapshot
	@command -v goreleaser >/dev/null || { echo 'goreleaser is required; run make dev-setup' >&2; exit 1; }
	goreleaser release --snapshot --clean

release-dry-run: ## Exercise GoReleaser without publishing
	@command -v goreleaser >/dev/null || { echo 'goreleaser is required; run make dev-setup' >&2; exit 1; }
	goreleaser release --snapshot --clean --skip=publish

tag: check-git-clean ## Create and push an annotated release tag (TAG=v1.2.3)
	@set -eu; \
	[ -n '$(TAG)' ] || { echo 'TAG is required' >&2; exit 1; }; \
	printf '%s\n' '$(TAG)' | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)[0-9]*)?$$' || { echo 'TAG must be SemVer' >&2; exit 1; }; \
	! git rev-parse -q --verify 'refs/tags/$(TAG)' >/dev/null || { echo 'tag already exists' >&2; exit 1; }; \
	git tag -a '$(TAG)' -m 'Release $(TAG)'; \
	git push origin '$(TAG)'

install: ## Install the current source with embedded metadata
	go install -trimpath -tags='$(BUILD_TAGS)' -ldflags "$(LDFLAGS)" $(MAIN_PACKAGE)

install-local: build ## Install a development binary under ~/.govman/bin
	mkdir -p $(HOME)/.govman/bin
	cp $(BUILD_DIR)/$(BINARY_NAME) $(HOME)/.govman/bin/$(BINARY_NAME)
	chmod 0755 $(HOME)/.govman/bin/$(BINARY_NAME)

uninstall: ## Run the maintained Unix uninstaller
	bash scripts/uninstall.sh

docker: ## Build the local Docker image
	docker build -t govman:$(VERSION) -t govman:latest \
		--build-arg VERSION='$(VERSION)' --build-arg COMMIT='$(COMMIT)' \
		--build-arg DATE='$(DATE)' .

docker-run: docker ## Run the image version command
	docker run --rm govman:$(VERSION) --version

docker-shell: docker ## Open a shell in the image
	docker run --rm -it --entrypoint /bin/sh govman:$(VERSION)

docker-push: docker ## Push version and latest tags (DOCKER_REGISTRY required)
	@test -n '$(DOCKER_REGISTRY)' || { echo 'DOCKER_REGISTRY is required' >&2; exit 1; }
	docker tag govman:$(VERSION) $(DOCKER_REGISTRY)/govman:$(VERSION)
	docker tag govman:$(VERSION) $(DOCKER_REGISTRY)/govman:latest
	docker push $(DOCKER_REGISTRY)/govman:$(VERSION)
	docker push $(DOCKER_REGISTRY)/govman:latest

docker-clean: ## Remove project Docker image tags
	-docker rmi govman:$(VERSION) govman:latest

version: ## Print resolved build metadata
	@printf 'version=%s\ncommit=%s\ndate=%s\nbuild_by=%s\n' '$(VERSION)' '$(COMMIT)' '$(DATE)' '$(BUILD_BY)'

info: version ## Print Go build environment
	@go version
	@go env GOOS GOARCH GOPATH GOROOT CGO_ENABLED

clean: ## Remove project-local build artifacts only
	rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	rm -f coverage.out coverage.html gosec-report.sarif

check: validate test test-integration ## Run standard local quality gates

ci: deps validate test test-race test-integration build-binaries checksums verify-artifacts ## Run the complete local CI pipeline

update-deps: ## Explicitly update dependencies and module files
	go get -u ./...
	go mod tidy

outdated: ## List dependencies with available updates
	go list -u -m all
