# My Platform - Demo Walkthrough

This document provides a hands-on demonstration of the KRM SDK framework using the `my-platform` example.

## Prerequisites

- Go 1.21+
- The krm-sdk framework built (`make build` in the root directory)

## Step 1: Build the Example Project

```bash
cd examples/my-platform

# Install dependencies
go mod tidy

# Build the project binary
make build
```

## Step 2: Generate Base Resources

Generate resources without any overlay:

```bash
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false
```

**Output:**
- Deployment with 3 replicas (from instance spec)
- Service exposing port 80
- Standard labels and selectors

## Step 3: Generate with Dev Overlay

Apply the development overlay:

```bash
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay overlays/dev
```

**Changes Applied:**
- ✅ Replicas reduced to 1
- ✅ Minimal resources (100m CPU, 128Mi memory)
- ✅ `environment: dev` label added
- ✅ `dev-` name prefix added
- ✅ Name becomes: `dev-nginx-app`

## Step 4: Generate with Prod Overlay

Apply the production overlay:

```bash
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay overlays/prod
```

**Changes Applied:**
- ✅ Replicas increased to 5
- ✅ Production resources (500m-2 CPU, 512Mi-2Gi memory)
- ✅ Rolling update strategy configured
- ✅ Node selector added (environment: production)
- ✅ `environment: production` and `tier: production` labels added
- ✅ `prod-` name prefix added
- ✅ Namespace changed to `production`
- ✅ Name becomes: `prod-nginx-app`

## Step 5: Compare Outputs

### Base vs Dev vs Prod

| Aspect | Base | Dev | Prod |
|--------|------|-----|------|
| **Replicas** | 3 | 1 | 5 |
| **Name** | nginx-app | dev-nginx-app | prod-nginx-app |
| **Namespace** | default | default | production |
| **CPU Request** | - | 100m | 500m |
| **CPU Limit** | - | 200m | 2 |
| **Memory Request** | - | 128Mi | 512Mi |
| **Memory Limit** | - | 256Mi | 2Gi |
| **Node Selector** | - | - | environment: production |
| **Update Strategy** | - | - | RollingUpdate (maxSurge: 1) |
| **Labels** | app, managed-by | + environment: dev | + environment: production, tier: production |

## Step 6: Output to Files

Generate and save to directory:

```bash
# Dev environment
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay overlays/dev -o output/dev/

# Prod environment
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay overlays/prod -o output/prod/

# Compare
diff output/dev/deployment-dev-nginx-app.yaml output/prod/deployment-prod-nginx-app.yaml
```

## Step 7: Apply to Cluster

Apply to a Kubernetes cluster:

```bash
# Dev environment
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay overlays/dev | kubectl apply -f -

# Prod environment  
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay overlays/prod | kubectl apply -f -

# Verify
kubectl get deployments -n default
kubectl get deployments -n production
```

## Step 8: Multiple Instances

Generate multiple applications with the same overlay:

```bash
# Generate all instances with prod overlay
./bin/my-platform generate -f instances/ --validate=false --overlay overlays/prod -o output/prod-all/

# This generates:
# - prod-nginx-app (Deployment + Service)
# - prod-api-service (Deployment + Service)
```

## Step 9: Custom Overlay

Create a custom overlay for a specific use case:

```bash
mkdir -p overlays/custom/patches

cat > overlays/custom/kustomization.yaml <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - ../../base

commonLabels:
  team: platform
  cost-center: engineering

namePrefix: custom-
EOF

cat > overlays/custom/patches/replicas.yaml <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-app
spec:
  replicas: 10
EOF

# Generate with custom overlay
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay overlays/custom
```

## Step 10: Using Absolute Paths

You can use overlays from anywhere:

```bash
# Copy overlay to temp location
cp -r overlays/prod /tmp/my-custom-overlay

# Use absolute path
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay /tmp/my-custom-overlay

# Or point directly to kustomization.yaml
./bin/my-platform generate -f instances/nginx-app.yaml --validate=false --overlay /tmp/my-custom-overlay/kustomization.yaml
```

## Key Takeaways

1. **Simple Abstractions**: Write simple YAML instances
2. **Powerful Templates**: DSL templates generate multiple resources
3. **Flexible Overlays**: Kustomize provides environment-specific customization
4. **No Cluster Access**: Everything runs client-side
5. **Standard Tools**: Works with kubectl, GitOps tools, etc.

## Next Steps

- Modify `api/v1alpha1/web_service_types.go` to add more fields
- Update `api/v1alpha1/web_service_template.yaml` to use new fields
- Create new overlays for different scenarios
- Add more abstractions (Database, CronJob, etc.)
- Integrate with your CI/CD pipeline

## Troubleshooting

### "schema not found" error

Run `make generate` to create CRDs, or use `--validate=false` to skip validation.

### "overlay not found" error

Ensure the overlay path exists and contains `kustomization.yaml`.

### Kustomize warnings

The warnings about deprecated fields are from kustomize itself. You can safely ignore them or run `kustomize edit fix` in the overlay directory to update.

---

**Congratulations!** You've successfully used KRM SDK to create type-safe, validated, client-side Kubernetes abstractions with environment-specific overlays!

