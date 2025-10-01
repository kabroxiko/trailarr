APP_NAME=gozarr
BIN_DIR=bin
SRC_DIR=cmd/gozarr

.PHONY: build clean run

build:
# 	go mod download
# 	go mod vendor
	go build -mod=vendor -o $(BIN_DIR)/$(APP_NAME) $(SRC_DIR)/main.go

run: build
	./$(BIN_DIR)/$(APP_NAME)

clean:
	rm -rf $(BIN_DIR)/*
