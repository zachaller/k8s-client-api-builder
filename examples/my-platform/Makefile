# Makefile for my-platform

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=my-platform
BINARY_PATH=bin/$(BINARY_NAME)

# Build parameters
BUILD_FLAGS=-v
LDFLAGS=-ldflags "-s -w"

# controller-gen
CONTROLLER_GEN=$(shell which controller-gen)
ifeq ($(CONTROLLER_GEN),)
CONTROLLER_GEN=$(shell go env GOPATH)/bin/controller-gen
endif

.PHONY: all build clean test generate help

all: generate build

## build: Build the project binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/$(BINARY_NAME)

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf bin/

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## generate: Generate code and manifests
generate: controller-gen
	@echo "Generating code..."
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	$(CONTROLLER_GEN) crd:crdVersions=v1 paths="./api/..." output:crd:artifacts:config=config/crd

## controller-gen: Install controller-gen if not present
controller-gen:
	@if [ ! -f "$(CONTROLLER_GEN)" ]; then \
		echo "Installing controller-gen..."; \
		$(GOGET) sigs.k8s.io/controller-tools/cmd/controller-gen@latest; \
	fi

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
