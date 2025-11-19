# My Platform - Example KRM SDK Project

This is an example project demonstrating the KRM SDK framework for building client-side Kubernetes abstractions.

## What This Example Shows

This project demonstrates:

1. **WebService Abstraction**: A custom abstraction that simplifies deploying web services
2. **Type-safe API**: Go structs with kubebuilder validation markers
3. **DSL-based Hydration**: YAML templates with inline expressions
4. **Multi-resource Generation**: One abstraction expands to multiple K8s resources

## Project Structure

```
my-platform/
├── api/v1alpha1/
│   ├── web_service_types.go        # Go struct definition with validations
│   ├── web_service_template.yaml   # Hydration template
│   ├── groupversion_info.go        # API group/version metadata
│   └── register.go                 # Scheme registration
├── cmd/my-platform/
│   ├── main.go                     # Binary entry point
│   └── commands/
│       └── root.go                 # CLI commands
├── instances/
│   ├── nginx-app.yaml              # Example: Simple nginx deployment
│   └── api-service.yaml            # Example: Production API service
├── config/
│   ├── crd/                        # Generated CRD manifests
│   └── samples/                    # Sample instances
└── Makefile                        # Build and generation targets
```

## How It Works

### 1. Define the Abstraction (Go Struct)

The `WebService` type is defined in `api/v1alpha1/web_service_types.go`:

```go
type WebServiceSpec struct {
    // +kubebuilder:validation:MinLength=1
    Image string `json:"image"`
    
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=100
    Replicas int32 `json:"replicas,omitempty"`
    
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=65535
    Port int32 `json:"port"`
    
    EnableHA bool `json:"enableHA,omitempty"`
}
```

### 2. Define the Hydration Template

The template in `api/v1alpha1/web_service_template.yaml` defines how to expand the abstraction:

```yaml
resources:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: $(.metadata.name)
    spec:
      replicas: $(.spec.replicas)
      template:
        spec:
          containers:
          - image: $(.spec.image)
            ports:
            - containerPort: $(.spec.port)
      $if(.spec.enableHA):
        affinity:
          podAntiAffinity: ...
  
  - apiVersion: v1
    kind: Service
    ...
```

### 3. Create Instances

Users create simple YAML files (`instances/nginx-app.yaml`):

```yaml
apiVersion: platform.example.com/v1alpha1
kind: WebService
metadata:
  name: nginx-app
spec:
  image: nginx:1.25
  replicas: 3
  port: 80
  enableHA: true
```

### 4. Generate Resources

The tool hydrates the abstraction into standard Kubernetes resources:

```bash
./bin/my-platform generate -f instances/nginx-app.yaml
```

Output:
- Deployment with 3 replicas
- Service exposing port 80
- Pod anti-affinity rules (because enableHA: true)

## Usage Examples

### Build the Project

```bash
# Install dependencies
go mod tidy

# Generate CRDs and code
make generate

# Build the binary
make build
```

### Generate Resources

```bash
# Generate to stdout
./bin/my-platform generate -f instances/nginx-app.yaml

# Generate to directory
./bin/my-platform generate -f instances/ -o output/

# Generate single file
./bin/my-platform generate -f instances/api-service.yaml -o output/
```

### Generate with Overlays

```bash
# Development environment (1 replica, minimal resources)
./bin/my-platform generate -f instances/nginx-app.yaml --overlay overlays/dev

# Staging environment (default settings with staging labels)
./bin/my-platform generate -f instances/nginx-app.yaml --overlay overlays/staging

# Production environment (5 replicas, high resources, node selectors)
./bin/my-platform generate -f instances/nginx-app.yaml --overlay overlays/prod

# Output to directory
./bin/my-platform generate -f instances/nginx-app.yaml --overlay overlays/prod -o output/

# Apply directly to cluster
./bin/my-platform generate -f instances/nginx-app.yaml --overlay overlays/prod | kubectl apply -f -
```

### Validate Instances

```bash
# Validate before generating
./bin/my-platform validate -f instances/nginx-app.yaml

# Validate all instances
./bin/my-platform validate -f instances/
```

### Apply to Cluster

```bash
# Generate and pipe to kubectl
./bin/my-platform generate -f instances/nginx-app.yaml | kubectl apply -f -

# Or use the apply command (when implemented)
./bin/my-platform apply -f instances/nginx-app.yaml
```

## DSL Features Demonstrated

### Variable Substitution

```yaml
name: $(.metadata.name)
image: $(.spec.image)
replicas: $(.spec.replicas)
```

### Conditionals

```yaml
$if(.spec.enableHA):
  affinity:
    podAntiAffinity:
      ...
```

### Functions (available but not used in this example)

```yaml
labels:
  app: $(lower(.metadata.name))
  hash: $(sha256(.spec.image))
```

## Extending This Example

### Add a New Field

1. Edit `api/v1alpha1/web_service_types.go`:
   ```go
   // +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
   ServiceType string `json:"serviceType,omitempty"`
   ```

2. Update `api/v1alpha1/web_service_template.yaml`:
   ```yaml
   - apiVersion: v1
     kind: Service
     spec:
       type: $(.spec.serviceType)
   ```

3. Regenerate:
   ```bash
   make generate
   make build
   ```

### Add a New Abstraction

```bash
# From the project root
krm-sdk create api --group platform --version v1alpha1 --kind Database

# Edit the generated files
# - api/v1alpha1/database_types.go
# - api/v1alpha1/database_template.yaml

make generate
make build
```

## What Gets Generated

From a single `WebService` instance, the tool generates:

1. **Deployment**: Manages pods with your container
2. **Service**: Exposes the deployment
3. **Optional HA Config**: Pod anti-affinity when `enableHA: true`

All with proper labels, selectors, and metadata automatically configured!

## Overlay System

This example project includes Kustomize overlays for different environments.

### Overlay Structure

```
overlays/
├── dev/
│   ├── kustomization.yaml
│   └── patches/
│       └── replicas.yaml      # 1 replica, minimal resources
├── staging/
│   └── kustomization.yaml     # Default settings with staging labels
└── prod/
    ├── kustomization.yaml
    └── patches/
        ├── replicas.yaml      # 5 replicas, rolling updates
        └── resources.yaml     # Production resource limits
```

### What Each Overlay Does

**Development (`overlays/dev/`):**
- Sets replicas to 1
- Minimal resource requests (100m CPU, 128Mi memory)
- Adds `environment: dev` label
- Adds `dev-` name prefix

**Staging (`overlays/staging/`):**
- Uses default replica count from instance
- Adds `environment: staging` label
- Adds `staging-` name prefix
- Overrides namespace to `staging`

**Production (`overlays/prod/`):**
- Sets replicas to 5
- Production resource limits (500m-2 CPU, 512Mi-2Gi memory)
- Adds node selector for production nodes
- Rolling update strategy (maxSurge: 1, maxUnavailable: 0)
- Adds `environment: production` and `tier: production` labels
- Adds `prod-` name prefix
- Overrides namespace to `production`

### Comparing Outputs

**Base (no overlay):**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-app
  namespace: default
spec:
  replicas: 3
  # ... standard configuration
```

**With Dev Overlay:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dev-nginx-app  # Prefix added
  namespace: default
  labels:
    environment: dev   # Label added
spec:
  replicas: 1          # Changed from 3
  template:
    spec:
      containers:
      - name: app
        resources:
          requests:
            cpu: "100m"      # Added
            memory: "128Mi"  # Added
```

**With Prod Overlay:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prod-nginx-app  # Prefix added
  namespace: production # Namespace changed
  labels:
    environment: production  # Labels added
    tier: production
spec:
  replicas: 5  # Changed from 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      nodeSelector:
        environment: production  # Added via JSON patch
      containers:
      - name: app
        resources:
          requests:
            cpu: "500m"     # Added
            memory: "512Mi" # Added
          limits:
            cpu: "2"        # Added
            memory: "2Gi"   # Added
```

### Customizing Overlays

You can modify the overlays to suit your needs:

1. **Edit kustomization.yaml** to change labels, prefixes, namespaces
2. **Add patches** in the `patches/` directory
3. **Use JSON patches** for precise modifications
4. **Add ConfigMaps/Secrets** using generators

See the [Overlay Guide](../../docs/overlay-guide.md) for more details.

## Benefits Over Plain YAML

1. **Type Safety**: Go structs catch errors at definition time
2. **Validation**: Kubebuilder markers enforce constraints
3. **Abstraction**: Hide complexity from users
4. **Consistency**: Templates ensure uniform resource generation
5. **Reusability**: Define once, use many times
6. **Composability**: Abstractions can reference other abstractions

## Next Steps

- Add more fields to `WebServiceSpec`
- Create additional abstractions (Database, CronJob, etc.)
- Add environment-specific overlays in `overlays/`
- Implement custom validation logic
- Add more sophisticated hydration templates

