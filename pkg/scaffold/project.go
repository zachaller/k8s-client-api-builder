package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectConfig holds configuration for project scaffolding
type ProjectConfig struct {
	Name    string
	Domain  string
	Repo    string
	Verbose bool
}

// ProjectScaffolder handles project initialization
type ProjectScaffolder struct {
	config ProjectConfig
}

// NewProjectScaffolder creates a new project scaffolder
func NewProjectScaffolder(config ProjectConfig) *ProjectScaffolder {
	// Set default repo if not provided
	if config.Repo == "" {
		config.Repo = fmt.Sprintf("github.com/example/%s", config.Name)
	}
	return &ProjectScaffolder{config: config}
}

// Scaffold creates a new project structure
func (s *ProjectScaffolder) Scaffold() error {
	projectDir := s.config.Name
	
	// Check if directory already exists
	if _, err := os.Stat(projectDir); err == nil {
		return fmt.Errorf("directory '%s' already exists", projectDir)
	}
	
	if s.config.Verbose {
		fmt.Printf("Creating project directory: %s\n", projectDir)
	}
	
	// Create project directory
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}
	
	// Create subdirectories
	dirs := []string{
		"cmd/" + s.config.Name,
		"cmd/" + s.config.Name + "/commands",
		"api/v1alpha1",
		"config/crd",
		"config/samples",
		"overlays/dev",
		"overlays/staging",
		"overlays/prod",
		"instances",
		"hack",
	}
	
	for _, dir := range dirs {
		path := filepath.Join(projectDir, dir)
		if s.config.Verbose {
			fmt.Printf("Creating directory: %s\n", path)
		}
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}
	
	// Generate files
	files := map[string]string{
		"go.mod":                                        s.generateGoMod(),
		"PROJECT":                                       s.generateProjectFile(),
		"Makefile":                                      s.generateMakefile(),
		"README.md":                                     s.generateReadme(),
		".gitignore":                                    s.generateGitignore(),
		"cmd/" + s.config.Name + "/main.go":            s.generateMain(),
		"cmd/" + s.config.Name + "/commands/root.go":   s.generateCommands(),
		"api/v1alpha1/groupversion_info.go":            s.generateGroupVersionInfo(),
		"api/v1alpha1/register.go":                     s.generateRegister(),
		"hack/boilerplate.go.txt":                      s.generateBoilerplate(),
	}
	
	for filename, content := range files {
		path := filepath.Join(projectDir, filename)
		if s.config.Verbose {
			fmt.Printf("Creating file: %s\n", path)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}
	
	return nil
}

func (s *ProjectScaffolder) generateGoMod() string {
	return fmt.Sprintf(`module %s

go 1.21

require (
	github.com/spf13/cobra v1.10.1
	github.com/yourusername/krm-sdk v0.1.0
	k8s.io/apimachinery v0.34.2
	k8s.io/apiextensions-apiserver v0.34.2
	sigs.k8s.io/yaml v1.6.0
)
`, s.config.Repo)
}

func (s *ProjectScaffolder) generateProjectFile() string {
	return fmt.Sprintf(`domain: %s
repo: %s
projectName: %s
version: "1"
`, s.config.Domain, s.config.Repo, s.config.Name)
}

func (s *ProjectScaffolder) generateMakefile() string {
	return fmt.Sprintf(`# Makefile for %s

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary names
BINARY_NAME=%s
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
`, s.config.Name, s.config.Name)
}

func (s *ProjectScaffolder) generateReadme() string {
	return fmt.Sprintf(`# %s

A KRM-based platform abstraction project.

## Overview

This project provides custom Kubernetes abstractions that expand into multiple K8s resources.

## Getting Started

### Prerequisites

- Go 1.21+
- kubectl
- Access to a Kubernetes cluster (for applying resources)

### Building

Build the project binary:

`+"```"+`bash
make build
`+"```"+`

### Generating Code

After modifying API types, regenerate code and CRDs:

`+"```"+`bash
make generate
`+"```"+`

### Usage

Create an instance of your abstraction:

`+"```"+`yaml
# instances/example.yaml
apiVersion: %s/v1alpha1
kind: YourKind
metadata:
  name: example
  namespace: default
spec:
  # Add your spec fields here
`+"```"+`

Generate Kubernetes resources:

`+"```"+`bash
./bin/%s generate -f instances/example.yaml
`+"```"+`

Apply with overlays:

`+"```"+`bash
./bin/%s apply -f instances/example.yaml --overlay prod
`+"```"+`

## Project Structure

- `+"`api/`"+` - API type definitions (Go structs with kubebuilder markers)
- `+"`cmd/`"+` - Main application entry point
- `+"`config/`"+` - Generated CRDs and sample manifests
- `+"`instances/`"+` - Instance files for your abstractions
- `+"`overlays/`"+` - Environment-specific overlays (dev/staging/prod)

## Adding New Abstractions

Use krm-sdk to scaffold new API types:

`+"```"+`bash
krm-sdk create api --group <group> --version <version> --kind <Kind>
make generate
make build
`+"```"+`

## License

Apache 2.0
`, strings.ToUpper(s.config.Name), s.config.Domain, s.config.Name, s.config.Name)
}

func (s *ProjectScaffolder) generateGitignore() string {
	return `# Binaries
*.exe
*.exe~
*.dll
*.so
*.dylib
bin/

# Test binary
*.test
*.out

# Go workspace
go.work

# Dependencies
vendor/

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store

# Temporary
tmp/
temp/
`
}

func (s *ProjectScaffolder) generateMain() string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"os"

	"%s/cmd/%s/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %%v\n", err)
		os.Exit(1)
	}
}
`, s.config.Repo, s.config.Name)
}

func (s *ProjectScaffolder) generateGroupVersionInfo() string {
	return fmt.Sprintf(`// Package v1alpha1 contains API Schema definitions for the platform v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=%s
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "%s", Version: "v1alpha1"}

	// SchemeGroupVersion is an alias for GroupVersion (for compatibility)
	SchemeGroupVersion = GroupVersion
)
`, s.config.Domain, s.config.Domain)
}

func (s *ProjectScaffolder) generateRegister() string {
	return `package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	
	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

// addKnownTypes adds the set of types defined in this package to the supplied scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion)
	return nil
}
`
}

func (s *ProjectScaffolder) generateCommands() string {
	return fmt.Sprintf(`package commands

import (
	"github.com/yourusername/krm-sdk/pkg/cli"
)

// Execute runs the root command
func Execute() error {
	rootCmd := cli.BuildRootCommand("%s")
	return rootCmd.Execute()
}
`, s.config.Name)
}

func (s *ProjectScaffolder) generateBoilerplate() string {
	return `/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
`
}

