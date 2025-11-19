# Makefile for krm-sdk framework

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=krm-sdk
BINARY_PATH=bin/$(BINARY_NAME)

# Build parameters
BUILD_FLAGS=-v
LDFLAGS=-ldflags "-s -w"

.PHONY: all build clean test install tools help

all: test build

## build: Build the krm-sdk binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/krm-sdk

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## install: Install krm-sdk binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BINARY_PATH) $(GOPATH)/bin/

## tools: Install required tools
tools:
	@echo "Installing tools..."
	$(GOGET) sigs.k8s.io/controller-tools/cmd/controller-gen@latest

## tidy: Tidy go modules
tidy:
	$(GOMOD) tidy

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

