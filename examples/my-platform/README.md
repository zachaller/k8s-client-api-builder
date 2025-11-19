# MY-PLATFORM

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

```bash
make build
```

### Generating Code

After modifying API types, regenerate code and CRDs:

```bash
make generate
```

### Usage

Create an instance of your abstraction:

```yaml
# instances/example.yaml
apiVersion: platform.example.com/v1alpha1
kind: YourKind
metadata:
  name: example
  namespace: default
spec:
  # Add your spec fields here
```

Generate Kubernetes resources:

```bash
./bin/my-platform generate -f instances/example.yaml
```

Apply with overlays:

```bash
./bin/my-platform generate -f instances/example.yaml --overlay prod | kubectl apply -f -
```

## Using Overlays

This project uses Kustomize for environment-specific configuration.

### Generate with Overlay

```bash
# Development environment
./bin/my-platform generate -f instances/example.yaml --overlay dev

# Staging environment
./bin/my-platform generate -f instances/example.yaml --overlay staging

# Production environment
./bin/my-platform generate -f instances/example.yaml --overlay prod
```

### Customize Overlays

Edit the kustomization files:
- `overlays/dev/kustomization.yaml`
- `overlays/staging/kustomization.yaml`
- `overlays/prod/kustomization.yaml`

Add patches in the `patches/` directories.

### Kustomize Features Supported

- Strategic merge patches
- JSON patches
- Common labels/annotations
- Name prefixes/suffixes
- Namespace overrides
- ConfigMap/Secret generators
- Image tag updates

## Project Structure

- `api/` - API type definitions (Go structs with kubebuilder markers)
- `cmd/` - Main application entry point
- `config/` - Generated CRDs and sample manifests
- `instances/` - Instance files for your abstractions
- `base/` - Base resources (auto-generated during overlay application)
- `overlays/` - Environment-specific overlays (dev/staging/prod)

## Adding New Abstractions

Use krm-sdk to scaffold new API types:

```bash
krm-sdk create api --group <group> --version <version> --kind <Kind>
make generate
make build
```

## License

Apache 2.0
