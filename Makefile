APP_NAME    := ssherpa
MAIN        := ./cmd/ssherpa
DIST        := dist
COVER       := cover.out

VERSION     := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      := $(shell git rev-parse HEAD 2>/dev/null || echo "none")
DATE        := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
MODULE      := github.com/florianriquelme/ssherpa/internal/version

LDFLAGS     := -s -w \
               -X $(MODULE).Version=$(VERSION) \
               -X $(MODULE).Commit=$(COMMIT) \
               -X $(MODULE).Date=$(DATE)

.PHONY: build run install test test-v test-cover test-race lint vet fmt tidy clean release-dry demo demo-setup demo-clean help

## Build & Run ---------------------------------------------------------------

build: ## Build the binary
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $(DIST)/$(APP_NAME) $(MAIN)

run: build ## Build and run
	$(DIST)/$(APP_NAME)

install: ## Install to $GOPATH/bin
	CGO_ENABLED=0 go install -ldflags '$(LDFLAGS)' $(MAIN)

## Testing -------------------------------------------------------------------

test: ## Run tests
	go test ./...

test-v: ## Run tests (verbose)
	go test -v ./...

test-cover: ## Run tests with coverage report
	go test -coverprofile=$(COVER) ./...
	go tool cover -func=$(COVER)

test-race: ## Run tests with race detector
	go test -race ./...

## Code Quality --------------------------------------------------------------

lint: vet ## Run linters (golangci-lint + go vet)
	@which golangci-lint > /dev/null 2>&1 && golangci-lint run ./... \
		|| echo "golangci-lint not installed — skipping (install: https://golangci-lint.run/welcome/install)"

vet: ## Run go vet
	go vet ./...

fmt: ## Format code
	gofmt -w -s .

tidy: ## Tidy modules
	go mod tidy

## Release -------------------------------------------------------------------

release-dry: ## GoReleaser dry run (no publish)
	goreleaser release --snapshot --clean

## Housekeeping --------------------------------------------------------------

clean: ## Remove build artifacts
	rm -rf $(DIST) $(COVER)

## Help ----------------------------------------------------------------------

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
