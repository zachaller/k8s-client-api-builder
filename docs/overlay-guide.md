# Overlay Guide - Kustomize Integration

This guide explains how to use Kustomize overlays with KRM SDK to customize generated resources for different environments.

## Overview

KRM SDK integrates natively with Kustomize, allowing you to apply environment-specific customizations to generated resources without modifying the base templates or instance files.

## How It Works

1. **Generate Base Resources**: Your abstractions are hydrated into K8s resources
2. **Write to Base**: Resources are written to `base/` directory with `kustomization.yaml`
3. **Apply Overlay**: Kustomize builds the overlay, applying patches and transformations
4. **Output Final Resources**: Customized resources are returned

## Directory Structure

When you initialize a project with `krm-sdk init`, the following overlay structure is created:

```
my-platform/
├── base/                    # Auto-generated during overlay application
│   └── kustomization.yaml   # Created automatically
├── overlays/
│   ├── dev/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   │       └── replicas.yaml
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   └── prod/
│       ├── kustomization.yaml
│       └── patches/
│           ├── replicas.yaml
│           └── resources.yaml
```

## Usage

### Basic Usage

The `--overlay` flag accepts two formats:

**1. Directory path (containing kustomization.yaml):**
```bash
# Relative path to overlay directory
./bin/my-platform generate -f instances/my-app.yaml --overlay overlays/dev
./bin/my-platform generate -f instances/my-app.yaml --overlay overlays/prod

# Absolute path
./bin/my-platform generate -f instances/my-app.yaml --overlay /path/to/my-overlay

# Custom overlay location
./bin/my-platform generate -f instances/my-app.yaml --overlay /tmp/custom-overlay
```

**2. Direct kustomization.yaml file path:**
```bash
# Point directly to the kustomization file
./bin/my-platform generate -f instances/my-app.yaml --overlay overlays/prod/kustomization.yaml

# Absolute path to kustomization file
./bin/my-platform generate -f instances/my-app.yaml --overlay /path/to/kustomization.yaml
```

### Without Overlay

Generate base resources without any customization:

```bash
./bin/my-platform generate -f instances/my-app.yaml
```

## Kustomization Files

### Development Overlay

```yaml
# overlays/dev/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

patchesStrategicMerge:
  - patches/replicas.yaml

commonLabels:
  environment: dev

namePrefix: dev-
```

### Production Overlay

```yaml
# overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

patchesStrategicMerge:
  - patches/replicas.yaml
  - patches/resources.yaml

patches:
  - target:
      kind: Deployment
    patch: |-
      - op: add
        path: /spec/template/spec/nodeSelector
        value:
          environment: production

commonLabels:
  environment: production
  tier: production

namePrefix: prod-
namespace: production
```

## Patch Files

### Strategic Merge Patches

Strategic merge patches allow you to selectively update fields:

```yaml
# overlays/prod/patches/replicas.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "*"  # Wildcard - applies to all Deployments
spec:
  replicas: 5
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
```

```yaml
# overlays/prod/patches/resources.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "*"
spec:
  template:
    spec:
      containers:
      - name: "*"  # Applies to all containers
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2"
            memory: "2Gi"
```

### JSON Patches

For more precise modifications:

```yaml
patches:
  - target:
      kind: Deployment
      name: my-app
    patch: |-
      - op: add
        path: /spec/template/spec/nodeSelector
        value:
          disktype: ssd
      
      - op: replace
        path: /spec/replicas
        value: 10
      
      - op: remove
        path: /spec/template/spec/containers/0/env/0
```

## Kustomize Features

All standard Kustomize features are supported:

### Common Labels

Add labels to all resources:

```yaml
commonLabels:
  environment: production
  team: platform
  managed-by: krm-sdk
```

### Common Annotations

Add annotations to all resources:

```yaml
commonAnnotations:
  deployed-by: ci-cd-pipeline
  version: "1.2.3"
```

### Name Prefix/Suffix

Add prefixes or suffixes to resource names:

```yaml
namePrefix: prod-
nameSuffix: -v2
```

### Namespace Override

Override the namespace for all resources:

```yaml
namespace: production
```

### Image Tag Updates

Update container images:

```yaml
images:
  - name: nginx
    newTag: 1.25.0
  - name: myapp/api
    newName: myregistry.io/api
    newTag: v2.0.0
```

### ConfigMap/Secret Generators

Generate ConfigMaps and Secrets:

```yaml
configMapGenerator:
  - name: app-config
    literals:
      - ENV=production
      - LOG_LEVEL=info
    files:
      - config.properties

secretGenerator:
  - name: app-secrets
    literals:
      - API_KEY=secret-value
```

## Examples

### Example 1: Environment-Specific Replicas

**Base Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: WebService
metadata:
  name: my-app
spec:
  image: nginx:latest
  replicas: 3
  port: 80
```

**Dev Overlay** (1 replica):
```yaml
# overlays/dev/patches/replicas.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "*"
spec:
  replicas: 1
```

**Prod Overlay** (5 replicas):
```yaml
# overlays/prod/patches/replicas.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "*"
spec:
  replicas: 5
```

### Example 2: Resource Limits by Environment

**Dev** (minimal resources):
```yaml
# overlays/dev/patches/resources.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "*"
spec:
  template:
    spec:
      containers:
      - name: "*"
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "200m"
            memory: "256Mi"
```

**Prod** (production resources):
```yaml
# overlays/prod/patches/resources.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: "*"
spec:
  template:
    spec:
      containers:
      - name: "*"
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2"
            memory: "2Gi"
```

### Example 3: Node Selectors

Add node selectors for production:

```yaml
# overlays/prod/kustomization.yaml
patches:
  - target:
      kind: Deployment
    patch: |-
      - op: add
        path: /spec/template/spec/nodeSelector
        value:
          environment: production
          disktype: ssd
```

## Best Practices

### 1. Keep Overlays Minimal

Only override what's necessary for each environment:

```yaml
# Good: Only change what's needed
patchesStrategicMerge:
  - patches/replicas.yaml

# Avoid: Duplicating entire resources
```

### 2. Use Wildcards

Apply patches to all resources of a kind:

```yaml
metadata:
  name: "*"  # Applies to all
```

### 3. Organize Patches

Group related patches in separate files:

```
patches/
├── replicas.yaml      # Replica counts
├── resources.yaml     # Resource limits
├── networking.yaml    # Network policies
└── monitoring.yaml    # Monitoring config
```

### 4. Document Overlays

Add comments explaining why patches exist:

```yaml
# Production overlay
# - Increases replicas for high availability
# - Sets production-grade resource limits
# - Adds node selectors for dedicated nodes
```

### 5. Test Overlays

Always test overlay output before applying:

```bash
# Generate and review
./bin/my-platform generate -f instances/my-app.yaml --overlay prod

# Dry run in cluster
./bin/my-platform generate -f instances/my-app.yaml --overlay prod | kubectl apply --dry-run=client -f -
```

## Troubleshooting

### Overlay Not Found

```
Error: overlay 'prod' not found at overlays/prod
```

**Solution**: Ensure the overlay directory exists and contains `kustomization.yaml`

### Invalid Kustomization

```
Error: kustomize build failed: ...
```

**Solution**: Validate your kustomization.yaml:
```bash
cd overlays/prod
kustomize build .
```

### Patches Not Applied

**Check:**
1. Target selector matches your resources
2. Patch file is listed in kustomization.yaml
3. YAML syntax is correct

### Name Conflicts

If using `namePrefix`, ensure it doesn't create conflicts:

```yaml
# This might cause issues
namePrefix: my-very-long-prefix-that-exceeds-kubernetes-limits-
```

## Advanced Usage

### Multiple Patch Files

```yaml
patchesStrategicMerge:
  - patches/replicas.yaml
  - patches/resources.yaml
  - patches/affinity.yaml
  - patches/tolerations.yaml
```

### Conditional Patches

Use different overlays for different scenarios:

```bash
# High traffic
./bin/my-platform generate -f instances/api.yaml --overlay prod-high-traffic

# Cost optimized
./bin/my-platform generate -f instances/api.yaml --overlay prod-cost-optimized
```

### Overlay Composition

Create overlay hierarchies:

```
overlays/
├── base-prod/           # Common prod settings
│   └── kustomization.yaml
├── prod-us-east/        # Region-specific
│   └── kustomization.yaml  # resources: - ../base-prod
└── prod-eu-west/
    └── kustomization.yaml  # resources: - ../base-prod
```

## Integration with CI/CD

### GitOps Workflow

```yaml
# .github/workflows/deploy.yml
- name: Generate manifests
  run: |
    ./bin/my-platform generate \
      -f instances/${{ matrix.app }}.yaml \
      --overlay ${{ matrix.environment }} \
      -o manifests/

- name: Commit to GitOps repo
  run: |
    cd gitops-repo
    cp ../manifests/* apps/${{ matrix.app }}/${{ matrix.environment }}/
    git add .
    git commit -m "Update ${{ matrix.app }} in ${{ matrix.environment }}"
    git push
```

### Direct Apply

```yaml
- name: Apply to cluster
  run: |
    ./bin/my-platform generate \
      -f instances/my-app.yaml \
      --overlay ${{ env.ENVIRONMENT }} | \
    kubectl apply -f -
```

## Comparison with Helm Values

| Feature | Helm Values | KRM SDK Overlays |
|---------|-------------|------------------|
| **Customization** | values.yaml | kustomization.yaml + patches |
| **Validation** | JSON Schema | OpenAPI + Go structs |
| **Patches** | Template logic | Strategic merge or JSON |
| **Preview** | helm template | generate --overlay |
| **Standard** | Helm-specific | Standard Kustomize |
| **Tooling** | Helm CLI | kubectl + kustomize |

## Summary

- ✅ Use native Kustomize for overlays
- ✅ Keep overlays minimal and focused
- ✅ Test overlays before applying
- ✅ Document why patches exist
- ✅ Use wildcards for broad changes
- ✅ Organize patches logically

Overlays provide powerful, standardized customization without modifying your abstraction definitions or templates!

