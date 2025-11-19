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

.PHONY: all build clean test test-integration test-golden test-golden-update test-all coverage install tools help

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

## test: Run unit tests
test:
	@echo "Running unit tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./pkg/...

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

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -race ./test/integration/...

## test-golden: Run golden file tests
test-golden:
	@echo "Running golden file tests..."
	$(GOTEST) -v ./test/integration/... -golden

## test-golden-update: Update golden files
test-golden-update:
	@echo "Updating golden files..."
	$(GOTEST) -v ./test/integration/... -golden -update

## test-all: Run all tests
test-all: test test-integration

## coverage: Generate coverage report
coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

