APP_NAME=trailarr
BIN_DIR=bin
SRC_DIR=cmd/trailarr

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

ifeq ($(GOOS),windows)
	BIN_EXT=.exe
else
	BIN_EXT=
endif

.PHONY: build clean run

.PHONY: test test-fast

build:
	go mod tidy
	go mod vendor
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -mod=vendor -o $(BIN_DIR)/$(APP_NAME)$(BIN_EXT) $(SRC_DIR)/main.go
	ls -l $(BIN_DIR)

run: build
ifeq ($(GOOS),windows)
	$(BIN_DIR)\\$(APP_NAME)$(BIN_EXT)
else
	./$(BIN_DIR)/$(APP_NAME)$(BIN_EXT)
endif

clean:
ifeq ($(GOOS),windows)
	del /Q $(BIN_DIR)\\*
else
	rm -rf $(BIN_DIR)/*
endif

# Run the full test suite in a CI-friendly way (skip embedded redis startup)
test:
	@echo "Running full test suite (embedded redis disabled)"
	go test ./... -v

# Faster, package-scoped tests for quick cycles
test-fast:
	@echo "Running internal package tests (embedded redis disabled)"
	go test ./internal -v

# Generate coverage report (coverage/coverage.out)
coverage:
	@echo "Generating coverage report (coverage/coverage.out)"
	@mkdir -p coverage
	go test ./... -coverprofile=coverage/coverage.out

# Generate HTML coverage (coverage/coverage.html)
coverage-html: coverage
	@echo "Generating HTML coverage report (coverage/coverage.html)"
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
