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
