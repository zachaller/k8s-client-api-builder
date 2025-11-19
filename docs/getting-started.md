# Getting Started with KRM SDK

This guide will walk you through creating your first client-side Kubernetes abstraction using KRM SDK.

## Prerequisites

- Go 1.21 or later
- kubectl (for applying generated resources)
- Access to a Kubernetes cluster (optional, for testing)

## Installation

### Install the Framework CLI

```bash
go install github.com/zachaller/k8s-client-api-builder/cmd/krm-sdk@latest
```

Verify the installation:

```bash
krm-sdk version
```

## Create Your First Project

### 1. Initialize a New Project

```bash
krm-sdk init my-platform --domain platform.mycompany.com
cd my-platform
```

This creates a new project with:
- Go module configuration
- Makefile for building and code generation
- API directory structure
- Configuration directories
- Example overlays for different environments

### 2. Create Your First Abstraction

Let's create a `WebService` abstraction that simplifies deploying web applications:

```bash
krm-sdk create api --group platform --version v1alpha1 --kind WebService
```

This generates:
- `api/v1alpha1/web_service_types.go` - Go struct definition
- `api/v1alpha1/web_service_template.yaml` - Hydration template
- `config/samples/web_service.yaml` - Sample instance

### 3. Define the API

Edit `api/v1alpha1/web_service_types.go` to add fields:

```go
type WebServiceSpec struct {
    // Image is the container image to deploy
    // +kubebuilder:validation:MinLength=1
    Image string `json:"image"`
    
    // Replicas is the number of pod replicas
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=100
    // +kubebuilder:default=1
    Replicas int32 `json:"replicas,omitempty"`
    
    // Port is the container port to expose
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=65535
    Port int32 `json:"port"`
}
```

### 4. Define the Hydration Template

Edit `api/v1alpha1/web_service_template.yaml`:

```yaml
resources:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: $(.metadata.name)
      namespace: $(.metadata.namespace)
      labels:
        app: $(.metadata.name)
    spec:
      replicas: $(.spec.replicas)
      selector:
        matchLabels:
          app: $(.metadata.name)
      template:
        metadata:
          labels:
            app: $(.metadata.name)
        spec:
          containers:
          - name: app
            image: $(.spec.image)
            ports:
            - containerPort: $(.spec.port)
  
  - apiVersion: v1
    kind: Service
    metadata:
      name: $(.metadata.name)
      namespace: $(.metadata.namespace)
    spec:
      selector:
        app: $(.metadata.name)
      ports:
      - port: $(.spec.port)
        targetPort: $(.spec.port)
```

### 5. Generate Code and Build

```bash
# Generate CRDs and Go code
make generate

# Build the project binary
make build
```

### 6. Create an Instance

Create `instances/my-app.yaml`:

```yaml
apiVersion: platform.mycompany.com/v1alpha1
kind: WebService
metadata:
  name: my-app
  namespace: default
spec:
  image: nginx:latest
  replicas: 3
  port: 80
```

### 7. Generate Resources

```bash
# Generate to stdout
./bin/my-platform generate -f instances/my-app.yaml

# Generate to directory
./bin/my-platform generate -f instances/my-app.yaml -o output/

# Validate before generating
./bin/my-platform validate -f instances/my-app.yaml
```

### 8. Apply to Cluster

```bash
# Generate and apply
./bin/my-platform generate -f instances/my-app.yaml | kubectl apply -f -

# Or save to files first
./bin/my-platform generate -f instances/my-app.yaml -o output/
kubectl apply -f output/
```

## Understanding the DSL

The hydration templates use a simple, YAML-native DSL:

### Variable Substitution

```yaml
name: $(.metadata.name)
image: $(.spec.image)
replicas: $(.spec.replicas)
```

### Conditionals

```yaml
$if(.spec.enableSSL):
  tls:
    - secretName: $(lower(.metadata.name))-tls
```

### Loops

```yaml
$for(env in .spec.envVars):
  - name: $(env.name)
    value: $(env.value)
```

### Functions

Built-in functions for string manipulation:

```yaml
labels:
  app: $(lower(.metadata.name))
  version: $(trim(.spec.version))
  hash: $(sha256(.spec.image))
```

## Next Steps

- [DSL Reference](dsl-reference.md) - Complete DSL syntax guide
- [API Development Guide](api-development.md) - Advanced API patterns
- [Examples](../examples/) - More complex examples
- [Best Practices](best-practices.md) - Tips for building abstractions

## Common Patterns

### Adding Validation

Use kubebuilder markers for validation:

```go
// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
ServiceType string `json:"serviceType,omitempty"`

// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
Name string `json:"name"`

// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=253
Domain string `json:"domain"`
```

### Optional Fields with Defaults

```go
// +kubebuilder:default=1
// +optional
Replicas int32 `json:"replicas,omitempty"`
```

### Nested Structures

```go
type WebServiceSpec struct {
    Image string `json:"image"`
    
    // +optional
    Resources *ResourceRequirements `json:"resources,omitempty"`
}

type ResourceRequirements struct {
    CPU    string `json:"cpu,omitempty"`
    Memory string `json:"memory,omitempty"`
}
```

### Conditional Resource Generation

```yaml
$if(.spec.enableIngress):
  - apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: $(.metadata.name)
    spec:
      rules:
      - host: $(.spec.domain)
        http:
          paths:
          - path: /
            backend:
              service:
                name: $(.metadata.name)
                port:
                  number: $(.spec.port)
```

## Troubleshooting

### Validation Errors

If validation fails, check:
1. Required fields are present
2. Values are within specified ranges
3. Enums match allowed values

### Hydration Errors

If hydration fails, check:
1. Template syntax is correct
2. Paths reference valid fields
3. Conditionals evaluate to boolean values

### Build Errors

If `make generate` fails:
1. Ensure controller-gen is installed
2. Check Go syntax in types files
3. Verify kubebuilder markers are valid

## Getting Help

- Check the [examples](../examples/) directory
- Read the [DSL Reference](dsl-reference.md)
- Review error messages carefully - they usually point to the issue

