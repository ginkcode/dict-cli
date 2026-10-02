MODULE  := github.com/gink/dict-cli

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -s -w \
	-X 'main.version=$(VERSION)' \
	-X 'main.commit=$(COMMIT)' \
	-X 'main.date=$(DATE)'

BUILD_DIR := build
DIST_DIR  := dist
GO        := go
GOFLAGS   := -trimpath -ldflags "$(LDFLAGS)"

.PHONY: all build dict gram clean test vet install release dist version help
.DEFAULT_GOAL := build

all: build

build: dict gram ## Build both binaries

dict: ## Build dict
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/dict ./cmd/dict

gram: ## Build gram
	$(GO) build $(GOFLAGS) -o $(BUILD_DIR)/gram ./cmd/gram

clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR) $(DIST_DIR) vendor/

test: ## Run tests
	$(GO) test ./...

vet: ## Run go vet
	$(GO) vet ./...

install: build ## Install to $GOPATH/bin
	install -m 755 $(BUILD_DIR)/dict $(shell $(GO) env GOPATH)/bin/dict
	install -m 755 $(BUILD_DIR)/gram $(shell $(GO) env GOPATH)/bin/gram

PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64

release: ## Cross-build dict and gram and package local release archives
	@set -eu; \
	mkdir -p $(DIST_DIR); \
	stage=$$(mktemp -d); \
	trap 'rm -rf "$$stage"' EXIT; \
	for pair in $(PLATFORMS); do \
		goos=$${pair%%/*}; goarch=$${pair##*/}; \
		out=$(BUILD_DIR)/$$goos-$$goarch; \
		mkdir -p "$$out"; \
		echo "building $$goos/$$goarch"; \
		CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch $(GO) build $(GOFLAGS) -o "$$out/dict" ./cmd/dict; \
		CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch $(GO) build $(GOFLAGS) -o "$$out/gram" ./cmd/gram; \
		rm -rf "$$stage"/*; \
		if [ "$$goos" = "linux" ]; then \
			mkdir -p "$$stage/usr/bin"; \
			install -m 755 "$$out/dict" "$$stage/usr/bin/dict"; \
			install -m 755 "$$out/gram" "$$stage/usr/bin/gram"; \
			tar -czf $(DIST_DIR)/dict-cli_$(VERSION)_$${goos}_$${goarch}.tar.gz -C "$$stage" usr; \
		else \
			install -m 755 "$$out/dict" "$$stage/dict"; \
			install -m 755 "$$out/gram" "$$stage/gram"; \
			tar -czf $(DIST_DIR)/dict-cli_$(VERSION)_$${goos}_$${goarch}.tar.gz -C "$$stage" dict gram; \
		fi; \
	done

dist: release ## Alias for release

version: ## Print version
	@echo $(VERSION)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'
