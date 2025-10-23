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
	# Ensure covdata tool (used by Go toolchain for coverage handling) is installed.
	# Installing is a no-op if already present in module cache/bin.
	@echo "Ensuring covdata tool is available..."
	# Do not auto-install covdata in CI: recent golang.org/x/tools @latest may require
	# a newer Go toolchain which would force 'go' to download & switch versions on the runner.
	# If covdata is needed in your environment, install it manually or ensure the runner
	# has it available. We skip automatic installation to keep CI stable.
	@which covdata >/dev/null 2>&1 || echo "covdata not found; skipping automatic install (optional)."
	go test ./... -coverprofile=coverage/coverage.out

# Generate HTML coverage (coverage/coverage.html)
coverage-html: coverage
	@echo "Generating HTML coverage report (coverage/coverage.html)"
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
