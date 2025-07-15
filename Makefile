# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Go files
SERVER_GO_FILES := $(shell find cmd/server server common -name '*.go')
CLIENT_GO_FILES := $(shell find cmd/client client common -name '*.go')

# Binaries
SERVER_BINARY=frps
CLIENT_BINARY=frpc

# Directories
BIN_DIR=bin

all: build

build: $(BIN_DIR)/$(SERVER_BINARY) $(BIN_DIR)/$(CLIENT_BINARY)

$(BIN_DIR)/$(SERVER_BINARY): $(SERVER_GO_FILES)
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(SERVER_BINARY) ./cmd/server

$(BIN_DIR)/$(CLIENT_BINARY): $(CLIENT_GO_FILES)
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -o $(BIN_DIR)/$(CLIENT_BINARY) ./cmd/client

clean:
	rm -rf $(BIN_DIR)

.PHONY: all build clean
