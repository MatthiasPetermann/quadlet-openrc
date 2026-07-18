.PHONY: build build-static test clean

GO      ?= go
BIN_DIR := bin
BIN     := $(BIN_DIR)/quadlet-openrc

build:
	mkdir -p $(BIN_DIR)
	$(GO) build \
		-trimpath \
		-o $(BIN) \
		./cmd/quadlet-openrc

build-static:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 \
	GOOS=linux \
	GOARCH=amd64 \
	$(GO) build \
		-trimpath \
		-buildvcs=false \
		-ldflags="-s -w -extldflags '-static'" \
		-o $(BIN)-linux-amd64 \
		./cmd/quadlet-openrc

test:
	$(GO) test ./...

clean:
	rm -rf $(BIN_DIR)
