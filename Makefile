APP_NAME=trailarr
BIN_DIR=bin
SRC_DIR=cmd/trailarr

.PHONY: build clean run

build:
	go mod tidy
	go mod vendor
	# Build frontend and copy built web assets into assets/dist so they can be embedded via go:embed
	# Run frontend build (requires node/npm)
	npm run build --prefix web
	if [ -d web/dist ]; then rm -rf assets/dist && mkdir -p assets && cp -r web/dist assets/dist; fi
	go build -mod=vendor -o $(BIN_DIR)/$(APP_NAME) $(SRC_DIR)/main.go

run: build
	./$(BIN_DIR)/$(APP_NAME)

clean:
	rm -rf $(BIN_DIR)/*
