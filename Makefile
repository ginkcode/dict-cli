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

.PHONY: all build dict gram clean test vet install dist version help
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

dist: build ## Build tarballs for all platforms (local use; CI uses GHA)
	@mkdir -p $(DIST_DIR)
	@for bin in dict gram; do \
		for pair in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64; do \
			goos=$${pair%%/*}; goarch=$${pair##*/}; \
			GOOS=$$goos GOARCH=$$goarch $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$$bin-$$goos-$$goarch ./cmd/$$bin; \
			out=$(DIST_DIR)/$${bin}_$(VERSION)_$${goos}_$${goarch}.tar.gz; \
			if [ "$$goos" = "linux" ]; then \
				install -D -m 755 $(BUILD_DIR)/$$bin-$$goos-$$goarch $(DIST_DIR)/_tmp/usr/bin/$$bin; \
				tar -czf $$out -C $(DIST_DIR)/_tmp usr; \
			else \
				install -D -m 755 $(BUILD_DIR)/$$bin-$$goos-$$goarch $(DIST_DIR)/_tmp/$$bin; \
				tar -czf $$out -C $(DIST_DIR)/_tmp $$bin; \
			fi; \
			rm -rf $(DIST_DIR)/_tmp; \
		done; \
	done

version: ## Print version
	@echo $(VERSION)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-16s\033[0m %s\n", $$1, $$2}'
