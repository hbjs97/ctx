BINARY := ctx
BUILD_DIR := bin
CMD_PATH := ./cmd/ctx

GO ?= go
GOFLAGS ?=
LDFLAGS ?=

.PHONY: build test test-race test-e2e test-all lint coverage clean

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(CMD_PATH)

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-e2e:
	$(GO) test -tags=e2e ./test/e2e/...

test-all: test test-race

lint:
	golangci-lint run ./...

coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html
