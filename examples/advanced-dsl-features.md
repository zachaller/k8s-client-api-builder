# Advanced DSL Features Examples

This document demonstrates the advanced DSL features: array indexing, arithmetic operations, and string concatenation.

## Array Indexing

### Example 1: Accessing Array Elements

**Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: MultiContainer
metadata:
  name: my-app
spec:
  containers:
    - name: nginx
      image: nginx:1.25
      port: 80
    - name: sidecar
      image: envoy:latest
      port: 8080
  primaryIndex: 0
```

**Template:**
```yaml
resources:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: $(.metadata.name)
    spec:
      template:
        spec:
          containers:
          # Access first container
          - name: $(.spec.containers[0].name)
            image: $(.spec.containers[0].image)
            ports:
            - containerPort: $(.spec.containers[0].port)
          
          # Access second container
          - name: $(.spec.containers[1].name)
            image: $(.spec.containers[1].image)
            ports:
            - containerPort: $(.spec.containers[1].port)
          
          # Use variable index
          primaryContainer: $(.spec.containers[.spec.primaryIndex].name)
```

## Arithmetic Operations

### Example 2: Dynamic Scaling

**Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: ScalableService
metadata:
  name: api-service
spec:
  baseReplicas: 3
  scaleFactor: 2
  maxCPU: 1000
  cpuPerReplica: 200
```

**Template:**
```yaml
resources:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: $(.metadata.name)
    spec:
      # Calculate replicas
      replicas: $(.spec.baseReplicas * .spec.scaleFactor)
      
      template:
        spec:
          containers:
          - name: app
            resources:
              requests:
                # Calculate CPU based on replicas
                cpu: $(.spec.cpuPerReplica)m
              limits:
                # Calculate max CPU
                cpu: $((.spec.baseReplicas * .spec.cpuPerReplica) + 100)m
  
  - apiVersion: autoscaling/v2
    kind: HorizontalPodAutoscaler
    metadata:
      name: $(.metadata.name)
    spec:
      minReplicas: $(.spec.baseReplicas)
      maxReplicas: $(.spec.baseReplicas * .spec.scaleFactor * 2)
      targetCPUUtilizationPercentage: 70
```

**Generated:**
```yaml
spec:
  replicas: 6  # 3 * 2
  template:
    spec:
      containers:
      - name: app
        resources:
          requests:
            cpu: 200m
          limits:
            cpu: 700m  # (3 * 200) + 100
---
spec:
  minReplicas: 3
  maxReplicas: 12  # 3 * 2 * 2
```

### Example 3: Port Calculations

**Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: ServiceMesh
metadata:
  name: my-service
spec:
  basePort: 8000
  serviceCount: 3
```

**Template:**
```yaml
resources:
  $for(i in [0, 1, 2]):
    - apiVersion: v1
      kind: Service
      metadata:
        name: $(.metadata.name + "-" + i)
      spec:
        ports:
        # Each service gets a different port
        - port: $(.spec.basePort + i)
          targetPort: $(.spec.basePort + i)
```

## String Concatenation

### Example 4: Dynamic Resource Names

**Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: production
spec:
  environment: prod
  version: v1.2.3
  region: us-west-2
```

**Template:**
```yaml
resources:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      # Concatenate namespace and name
      name: $(.metadata.namespace + "-" + .metadata.name)
      labels:
        # Build composite labels
        app: $(.metadata.name)
        environment: $(.spec.environment)
        full-name: $(.spec.environment + "-" + .metadata.name + "-" + .spec.version)
        region-env: $(.spec.region + "-" + .spec.environment)
    spec:
      template:
        spec:
          containers:
          - name: app
            # Build image tag
            image: $("myregistry.io/" + .metadata.name + ":" + .spec.version)
            env:
            - name: APP_NAME
              value: $(.metadata.namespace + "/" + .metadata.name)
            - name: FULL_VERSION
              value: $(.spec.environment + "-" + .spec.version)
```

**Generated:**
```yaml
metadata:
  name: production-my-app
  labels:
    app: my-app
    environment: prod
    full-name: prod-my-app-v1.2.3
    region-env: us-west-2-prod
spec:
  template:
    spec:
      containers:
      - name: app
        image: myregistry.io/my-app:v1.2.3
        env:
        - name: APP_NAME
          value: production/my-app
        - name: FULL_VERSION
          value: prod-v1.2.3
```

## Combined Features

### Example 5: Complex Configuration

**Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: ComplexApp
metadata:
  name: web-app
  namespace: staging
spec:
  environments:
    - name: frontend
      replicas: 2
      port: 3000
    - name: backend
      replicas: 3
      port: 8080
  resourceMultiplier: 2
  domainSuffix: example.com
```

**Template:**
```yaml
resources:
  $for(env in .spec.environments):
    - apiVersion: apps/v1
      kind: Deployment
      metadata:
        # Concatenate multiple parts
        name: $(.metadata.name + "-" + env.name)
        namespace: $(.metadata.namespace)
      spec:
        # Arithmetic: multiply replicas
        replicas: $(env.replicas * .spec.resourceMultiplier)
        template:
          spec:
            containers:
            - name: $(env.name)
              ports:
              - containerPort: $(env.port)
              resources:
                limits:
                  # Calculate CPU based on replicas
                  cpu: $(env.replicas * 100)m
    
    - apiVersion: v1
      kind: Service
      metadata:
        name: $(.metadata.name + "-" + env.name)
      spec:
        ports:
        - port: $(env.port)
          targetPort: $(env.port)
    
    - apiVersion: networking.k8s.io/v1
      kind: Ingress
      metadata:
        name: $(.metadata.name + "-" + env.name)
      spec:
        rules:
        # Build hostname from multiple parts
        - host: $(env.name + "-" + .metadata.name + "." + .spec.domainSuffix)
          http:
            paths:
            - path: /
              backend:
                service:
                  name: $(.metadata.name + "-" + env.name)
                  port:
                    number: $(env.port)
```

**Generated Resources:**
- `web-app-frontend` Deployment with 4 replicas (2 * 2)
- `web-app-backend` Deployment with 6 replicas (3 * 2)
- Services for each
- Ingresses with hosts:
  - `frontend-web-app.example.com`
  - `backend-web-app.example.com`

## Best Practices

### 1. Use Parentheses for Complex Arithmetic

```yaml
# Good: Clear precedence
result: $((.spec.base + .spec.increment) * .spec.multiplier)

# Avoid: Ambiguous evaluation order
result: $(.spec.base + .spec.increment * .spec.multiplier)
```

### 2. Validate Array Indices

Add validation to your Go structs:

```go
type MySpec struct {
    // +kubebuilder:validation:MinItems=1
    Items []string `json:"items"`
    
    // +kubebuilder:validation:Minimum=0
    SelectedIndex int `json:"selectedIndex"`
}
```

### 3. Document Concatenation Logic

```yaml
# Build resource name: <namespace>-<app>-<env>
name: $(.metadata.namespace + "-" + .metadata.name + "-" + .spec.environment)
```

### 4. Use Variables for Repeated Calculations

While the DSL doesn't support variables, you can use Go struct fields:

```go
type MySpec struct {
    BaseReplicas int `json:"baseReplicas"`
    ScaleFactor  int `json:"scaleFactor"`
}
```

Then in templates:
```yaml
minReplicas: $(.spec.baseReplicas)
maxReplicas: $(.spec.baseReplicas * .spec.scaleFactor)
```

## Error Handling

### Array Index Out of Bounds

```yaml
# If .spec.items has 3 elements
value: $(.spec.items[5])  # ERROR: array index 5 out of bounds (length 3)
```

### Division by Zero

```yaml
# If .spec.divisor is 0
result: $(.spec.value / .spec.divisor)  # ERROR: division by zero
```

### Type Mismatches

```yaml
# If .spec.port is a string
result: $(.spec.port + 100)  # ERROR: cannot convert string to float64
```

## Performance Considerations

1. **Array Indexing**: O(1) operation, very fast
2. **Arithmetic**: Minimal overhead, operations are native
3. **String Concatenation**: Efficient for reasonable string lengths
4. **Complex Expressions**: Parsed once, evaluated many times

## Migration from Previous Versions

If you were working around these limitations:

### Before (Workaround):
```yaml
# Had to define every value explicitly
firstContainer: nginx
secondContainer: sidecar
totalReplicas: 6  # Manually calculated
fullName: prod-my-app  # Manually concatenated
```

### After (Using DSL):
```yaml
# Use dynamic expressions
firstContainer: $(.spec.containers[0].name)
secondContainer: $(.spec.containers[1].name)
totalReplicas: $(.spec.baseReplicas * .spec.scaleFactor)
fullName: $(.spec.environment + "-" + .metadata.name)
```

## Summary

The advanced DSL features enable:

- ✅ **Array Indexing**: Access specific elements from arrays
- ✅ **Arithmetic**: Perform calculations for dynamic values
- ✅ **String Concatenation**: Build composite strings
- ✅ **Parentheses**: Control evaluation order
- ✅ **Combined Operations**: Mix all features together

These features make templates more powerful and reduce the need for pre-calculated values in instance files.

