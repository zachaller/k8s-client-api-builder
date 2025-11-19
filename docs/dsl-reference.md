# DSL Reference

The KRM SDK uses a declarative YAML-based DSL with inline expressions for hydrating abstractions into Kubernetes resources.

## Design Principles

1. **YAML-native**: Looks and feels like regular YAML
2. **Clear syntax**: Easy to distinguish between static content and dynamic expressions
3. **Type-safe**: Expressions are validated against Go struct schemas
4. **Composable**: Supports conditionals, loops, and functions

## Feature Summary

The DSL includes the following capabilities:

### Core Features
- **Variable Substitution**: `$(.path.to.field)` - Access instance fields
- **Block Conditionals**: `$if(condition):` - Conditionally include YAML blocks
- **Inline Conditionals**: `$if(condition, trueValue, falseValue)` - Ternary operator
- **Loops**: `$for(var in .path):` - Iterate over arrays
- **Nested Loops**: Inner loops can reference outer loop variables
- **Resource References**: `$(resource(apiVersion, kind, name).field)` - Cross-resource field access

### Operations
- **Arithmetic**: `+`, `-`, `*`, `/`, `%` with parentheses for grouping
- **Comparison**: `==`, `!=`, `>`, `<`, `>=`, `<=`
- **String Concatenation**: `+` operator for combining strings
- **Array Indexing**: `[0]` for accessing array elements

### Built-in Functions
- **String Functions**: `lower()`, `upper()`, `trim()`, `replace()`
- **Hash Functions**: `sha256()`
- **Utility Functions**: `default()`, `if()`
- **Nested Functions**: Functions can be composed: `lower(trim(value))`

### Advanced Capabilities
- **Complex Resource Names**: Use expressions and functions in resource references
- **Optional Field Handling**: Missing fields in conditionals evaluate to `false`
- **Multi-dimensional Data**: Nested loops for complex data structures

## Syntax Overview

### Variable Substitution

Use `$(.path.to.field)` to substitute values from the instance:

```yaml
metadata:
  name: $(.metadata.name)
  namespace: $(.metadata.namespace)
spec:
  replicas: $(.spec.replicas)
  image: $(.spec.image)
```

**Path Rules:**
- Must start with `.` (refers to the root instance)
- Use dot notation to navigate nested fields
- Field names are case-sensitive and match Go struct field names (in JSON form)

**Examples:**

```yaml
# Simple field access
name: $(.metadata.name)

# Nested field access
cpu: $(.spec.resources.cpu)

# Array indexing
firstItem: $(.spec.items[0])
port: $(.spec.ports[0].number)
```

### Conditionals

#### Block-Level Conditionals

Use `$if(condition):` to conditionally include entire blocks of YAML:

```yaml
spec:
  $if(.spec.enableHA):
    affinity:
      podAntiAffinity:
        requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchLabels:
              app: $(.metadata.name)
```

**Condition Syntax:**

```yaml
# Boolean field
$if(.spec.enabled):
  feature: enabled

# Comparison operators
$if(.spec.replicas > 1):
  strategy:
    type: RollingUpdate

# Equality check
$if(.spec.type == "production"):
  resources:
    limits:
      cpu: "2"
```

#### Inline If Statements (Ternary)

Use `$if(condition, trueValue, falseValue)` for inline conditional values:

```yaml
metadata:
  annotations:
    # Set annotation based on condition
    ingress-enabled: $if(.spec.enableIngress, "true", "false")

spec:
  # Choose service type based on ingress setting
  type: $if(.spec.enableIngress, "ClusterIP", "LoadBalancer")
  
  # Use default value if field is missing
  replicas: $if(.spec.replicas, .spec.replicas, 1)
```

**Inline If Examples:**

```yaml
# Conditional resource limits
resources:
  limits:
    cpu: $if(.spec.highPerformance, "2000m", "500m")
    memory: $if(.spec.highPerformance, "4Gi", "1Gi")

# Conditional image tag
image: $(.spec.image + ":" + if(.spec.useLatest, "latest", .spec.version))

# Nested in strings
url: $("http://" + if(.spec.enableSSL, "secure", "standard") + ".example.com")
```

**Supported Operators:**
- `==` - Equality
- `!=` - Inequality
- `>` - Greater than
- `<` - Less than
- `>=` - Greater than or equal
- `<=` - Less than or equal

**Truthiness:**
- Boolean `true` → true
- Boolean `false` → false
- Non-empty string → true
- Empty string → false
- Number != 0 → true
- Number == 0 → false
- nil → false
- Missing optional fields → false

### Loops

Use `$for(var in .path):` to iterate over arrays:

```yaml
containers:
  $for(container in .spec.containers):
    - name: $(container.name)
      image: $(container.image)
      ports:
        $for(port in container.ports):
          - containerPort: $(port)
```

**Loop Syntax:**

```yaml
# Basic loop over root path
$for(item in .spec.items):
  - name: $(item.name)
    value: $(item.value)

# Nested loops with loop variable references
$for(service in .spec.services):
  - name: $(service.name)
    ports:
      # Inner loop references outer loop variable
      $for(port in service.ports):
        - port: $(port)

# Nested loops with complex data
$for(container in .spec.containers):
  - name: $(container.name)
    env:
      $for(envVar in container.env):
        - name: $(envVar.name)
          value: $(envVar.value)
```

**Loop with Conditionals:**

```yaml
# Only include env if provided
$for(app in .spec.apps):
  - name: $(app.name)
    $if(app.env):
      env:
        $for(envVar in app.env):
          - name: $(envVar.name)
            value: $(envVar.value)
```

**Loop Variable Scope:**
- Loop variables are only available within the loop body
- Outer instance fields are still accessible: `$(.metadata.name)`
- Inner loops can reference outer loop variables: `$(container.ports)`
- Loop variables shadow outer variables with the same name

**Iteration Paths:**
- Root paths start with `.` and reference the instance: `.spec.items`
- Loop variable paths reference fields from outer loop variables: `container.ports`
- Both types can be used in the same template

### Functions

Use `$(function(args))` to transform values:

```yaml
labels:
  app: $(lower(.metadata.name))
  version: $(trim(.spec.version))
  hash: $(sha256(.spec.image))
```

**Nested Functions:**

Functions can be nested to combine transformations:

```yaml
labels:
  # Trim then convert to lowercase
  name: $(lower(trim(.spec.name)))
  
  # Replace then convert to uppercase
  env: $(upper(replace(.spec.environment, "-", "_")))
  
  # Triple nesting
  url: $(upper(trim(replace(.spec.domain, "http://", ""))))

# Use in conditionals
type: $if(.spec.production, upper(.spec.type), lower(.spec.type))

# Use with default values
replicas: $(default(lower(.spec.size), "small"))
```

### Resource References

Reference fields from other resources in the same template:

```yaml
# Reference a Service's ClusterIP
serviceIP: $(resource("v1", "Service", "my-app").spec.clusterIP)

# Reference with array indexing
servicePort: $(resource("v1", "Service", "my-app").spec.ports[0].port)

# Reference with expression for name
secretName: $(resource("v1", "Secret", .metadata.name + "-secret").metadata.name)
```

**Complex Name Expressions:**

Resource names can use functions and complex expressions:

```yaml
# Name with concatenation
configValue: $(resource("v1", "ConfigMap", .metadata.name + "-config").data.key)

# Name with lowercase transformation
serviceIP: $(resource("v1", "Service", lower(.metadata.name)).spec.clusterIP)

# Name with trim and concatenation
secret: $(resource("v1", "Secret", trim(.spec.namespace) + "-secret").data.token)

# Name with conditional expression
endpoint: $(resource("v1", "Service", if(.spec.ha, .metadata.name + "-ha", .metadata.name)).spec.clusterIP)

# Combining multiple functions
data: $(resource("v1", "ConfigMap", upper(trim(.spec.name))).data.value)
```

## Resource References

### Overview

Resource references allow you to access fields from other resources defined in the same template. This is useful for:
- Referencing Service names in Ingresses
- Using Service ClusterIPs in ConfigMaps
- Referencing Secret names in Deployments
- Any cross-resource field access

### Syntax

**Format**: `$(resource(apiVersion, kind, name).path.to.field)`

**Arguments**:
- `apiVersion` - API version (e.g., "v1", "apps/v1")
- `kind` - Resource kind (e.g., "Service", "Secret")
- `name` - Resource name (can be a literal or expression)

**Field Path**: After the function, use dot notation to access fields

### Examples

#### Basic Reference

```yaml
resources:
  - apiVersion: v1
    kind: Service
    metadata:
      name: my-app
    spec:
      clusterIP: 10.0.0.1
      ports:
      - port: 80
  
  - apiVersion: v1
    kind: ConfigMap
    metadata:
      name: app-config
    data:
      # Reference the Service's ClusterIP
      SERVICE_HOST: $(resource("v1", "Service", "my-app").spec.clusterIP)
      
      # Reference the Service's port
      SERVICE_PORT: $(resource("v1", "Service", "my-app").spec.ports[0].port)
```

#### Ingress Referencing Service

```yaml
resources:
  - apiVersion: v1
    kind: Service
    metadata:
      name: backend
    spec:
      ports:
      - name: http
        port: 8080
  
  - apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: backend-ingress
    spec:
      rules:
      - host: api.example.com
        http:
          paths:
          - path: /
            backend:
              service:
                # Reference Service name
                name: $(resource("v1", "Service", "backend").metadata.name)
                port:
                  # Reference Service port
                  number: $(resource("v1", "Service", "backend").spec.ports[0].port)
```

#### Secret Reference in Deployment

```yaml
resources:
  - apiVersion: v1
    kind: Secret
    metadata:
      name: db-credentials
    stringData:
      password: secret123
  
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: app
    spec:
      template:
        spec:
          containers:
          - name: app
            env:
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  # Reference Secret name
                  name: $(resource("v1", "Secret", "db-credentials").metadata.name)
                  key: password
```

#### Dynamic Resource Names

```yaml
resources:
  - apiVersion: v1
    kind: Service
    metadata:
      name: $(.metadata.name)  # From instance
    spec:
      ports:
      - port: 80
  
  - apiVersion: v1
    kind: ConfigMap
    metadata:
      name: $(.metadata.name + "-config")
    data:
      # Reference using expression for name
      SERVICE_NAME: $(resource("v1", "Service", .metadata.name).metadata.name)
```

#### Building URLs

```yaml
resources:
  - apiVersion: v1
    kind: Service
    metadata:
      name: api
    spec:
      clusterIP: 10.0.0.1
      ports:
      - port: 8080
  
  - apiVersion: v1
    kind: ConfigMap
    metadata:
      name: endpoints
    data:
      # Build complete URL
      API_URL: $("http://" + resource("v1", "Service", "api").spec.clusterIP + ":" + resource("v1", "Service", "api").spec.ports[0].port)
```

### How It Works

Resource references use **two-pass processing**:

1. **Pass 1**: Generate all resources without resolving cross-resource references
2. **Pass 2**: Resolve resource references using the generated resources

This allows resources to reference any other resource in the template, regardless of order.

### Limitations

1. **Same template only**: Can only reference resources in the same template
2. **No circular references**: Resources can't reference each other in a loop
3. **Static fields only**: Can only reference fields that exist after pass 1
4. **No external resources**: Can't reference existing cluster resources

### Error Messages

**Resource not found:**
```
Error: resource not found: v1/Service/my-app
Available resources: [v1/ConfigMap/app-config, apps/v1/Deployment/my-app]
```

**Field not found:**
```
Error: field 'spec.clusterIP' not found in resource
```

## Built-in Functions

### String Functions

#### `lower(string)`
Converts string to lowercase.

```yaml
name: $(lower(.metadata.name))
# Input: "MyApp" → Output: "myapp"
```

#### `upper(string)`
Converts string to uppercase.

```yaml
env: $(upper(.spec.environment))
# Input: "prod" → Output: "PROD"
```

#### `trim(string)`
Removes leading and trailing whitespace.

```yaml
value: $(trim(.spec.description))
# Input: "  hello  " → Output: "hello"
```

#### `replace(string, old, new)`
Replaces all occurrences of old with new.

```yaml
sanitized: $(replace(.spec.name, "_", "-"))
# Input: "my_app" → Output: "my-app"
```

### Hash Functions

#### `sha256(string)`
Computes SHA256 hash of the input.

```yaml
configHash: $(sha256(.spec.config))
# Input: "config-data" → Output: "a1b2c3..."
```

### Utility Functions

#### `default(value, defaultValue)`
Returns defaultValue if value is nil or empty.

```yaml
replicas: $(default(.spec.replicas, 1))
# If .spec.replicas is empty → Output: 1
```

#### `if(condition, trueValue, falseValue)`
Returns trueValue if condition is true, otherwise returns falseValue. This is the inline/ternary form of conditionals.

```yaml
# Simple conditional value
serviceType: $if(.spec.enableIngress, "ClusterIP", "LoadBalancer")

# With function calls
cpu: $if(.spec.production, upper("high"), lower("LOW"))

# Nested in expressions
image: $(.spec.name + ":" + if(.spec.useLatest, "latest", .spec.version))

# Can also be used inside $() blocks
annotation: $(if(.spec.enabled, "true", "false"))
```

**Note:** The `if()` function can be used with or without the `$()` wrapper:
- `$if(condition, trueValue, falseValue)` - Direct syntax
- `$(if(condition, trueValue, falseValue))` - Function call syntax

Both forms are equivalent and produce the same result.

## Complete Examples

### Example 1: Simple Deployment

**Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: WebService
metadata:
  name: my-app
  namespace: production
spec:
  image: nginx:1.25
  replicas: 3
  port: 80
```

**Template:**
```yaml
resources:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: $(.metadata.name)
      namespace: $(.metadata.namespace)
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
```

**Generated:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
      - name: app
        image: nginx:1.25
        ports:
        - containerPort: 80
```

### Example 2: Conditional Features

**Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: WebService
metadata:
  name: api-service
spec:
  image: myapp/api:v1
  replicas: 5
  port: 8080
  enableHA: true
  enableMetrics: true
  metricsPort: 9090
```

**Template:**
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
          - name: app
            image: $(.spec.image)
            ports:
            - containerPort: $(.spec.port)
            $if(.spec.enableMetrics):
              - containerPort: $(.spec.metricsPort)
                name: metrics
      $if(.spec.enableHA):
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchLabels:
                    app: $(.metadata.name)
                topologyKey: kubernetes.io/hostname
```

### Example 3: Loops for Multiple Resources

**Instance:**
```yaml
apiVersion: platform.example.com/v1alpha1
kind: MultiService
metadata:
  name: microservices
spec:
  services:
    - name: frontend
      image: myapp/frontend:v1
      port: 3000
    - name: backend
      image: myapp/backend:v1
      port: 8080
    - name: worker
      image: myapp/worker:v1
      port: 9000
```

**Template:**
```yaml
resources:
  $for(svc in .spec.services):
    - apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: $(.metadata.name)-$(svc.name)
      spec:
        replicas: 1
        template:
          spec:
            containers:
            - name: $(svc.name)
              image: $(svc.image)
              ports:
              - containerPort: $(svc.port)
    
    - apiVersion: v1
      kind: Service
      metadata:
        name: $(.metadata.name)-$(svc.name)
      spec:
        ports:
        - port: $(svc.port)
```

## Advanced Patterns

### Nested Functions

Functions can be deeply nested to create complex transformations:

```yaml
labels:
  # Double nesting
  app: $(lower(trim(.metadata.name)))
  
  # Triple nesting
  domain: $(upper(trim(replace(.spec.url, "http://", ""))))
  
  # With hash
  configHash: $(sha256(lower(.spec.image)))

# In conditionals
serviceType: $if(.spec.production, upper(trim(.spec.type)), lower(.spec.type))

# In resource references
data: $(resource("v1", "ConfigMap", lower(trim(.spec.name))).data.key)
```

### Nested Loops

Loops can be nested to process multi-dimensional data:

```yaml
# Containers with ports
$for(container in .spec.containers):
  - name: $(container.name)
    ports:
      $for(port in container.ports):
        - containerPort: $(port)
    env:
      $for(envVar in container.env):
        - name: $(envVar.name)
          value: $(envVar.value)

# Services with endpoints
$for(service in .spec.services):
  - name: $(service.name)
    endpoints:
      $for(endpoint in service.endpoints):
        - address: $(endpoint.ip)
          port: $(endpoint.port)
```

### Loops with Conditionals

Combine loops and conditionals for flexible resource generation:

```yaml
# Optional nested data
$for(app in .spec.apps):
  - name: $(app.name)
    # Only include env if app has environment variables
    $if(app.env):
      env:
        $for(envVar in app.env):
          - name: $(envVar.name)
            value: $(envVar.value)
```

### Nested Conditionals

```yaml
$if(.spec.enableFeatureA):
  featureA:
    $if(.spec.featureAMode == "advanced"):
      advancedSettings: true
```

### Complex Conditions

```yaml
$if(.spec.replicas > 1):
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
```

### Inline Conditionals in Complex Expressions

```yaml
# Multiple inline conditionals
annotations:
  ingress: $if(.spec.enableIngress, "true", "false")
  monitoring: $if(.spec.enableMonitoring, "enabled", "disabled")
  env: $if(.spec.production, "prod", "dev")

# Inline conditionals with concatenation
image: $(.spec.registry + "/" + .spec.name + ":" + if(.spec.useLatest, "latest", .spec.version))

# Nested inline conditionals
cpu: $if(.spec.production, if(.spec.highPerformance, "4000m", "2000m"), "500m")
```

## Best Practices

1. **Keep templates readable**: Don't nest too deeply - limit nesting to 3-4 levels
2. **Use meaningful variable names**: `$for(container in .spec.containers)` not `$for(c in .spec.containers)`
3. **Prefer inline conditionals for simple values**: Use `$if(condition, true, false)` for single values, block conditionals for complex structures
4. **Validate early**: Use kubebuilder markers to enforce constraints on your API types
5. **Document complex logic**: Add YAML comments to explain complex nested functions or conditionals
6. **Test thoroughly**: Create sample instances for all code paths, especially optional fields and conditionals
7. **Avoid excessive nesting**: If you have more than 3 levels of nested functions, consider simplifying
8. **Use resource references wisely**: Resource references add a second pass, so use them only when necessary
9. **Handle optional fields**: Remember that missing fields in conditionals evaluate to `false`, not an error
10. **Test nested loops**: Multi-dimensional data structures can be complex - test with various data shapes

## Advanced Features

### Array Indexing

Access array elements by index:

```yaml
# Simple numeric index
firstItem: $(.spec.items[0])
secondItem: $(.spec.items[1])

# Variable index
selectedItem: $(.spec.items[.spec.selectedIndex])

# Nested array access
container: $(.spec.pods[0].containers[1])
```

**Examples:**

```yaml
# Instance
spec:
  items: ["apple", "banana", "cherry"]
  selectedIndex: 1

# Template
name: $(.spec.items[0])           # "apple"
selected: $(.spec.items[.spec.selectedIndex])  # "banana"
```

### Arithmetic Operations

Perform calculations in expressions:

```yaml
# Addition
total: $(.spec.base + .spec.increment)

# Subtraction
remaining: $(.spec.total - .spec.used)

# Multiplication
doubled: $(.spec.replicas * 2)

# Division
half: $(.spec.total / 2)

# Modulo
remainder: $(.spec.value % 3)

# Complex expressions with parentheses
result: $((.spec.a + .spec.b) * .spec.c)
```

**Supported Operators:**
- `+` - Addition
- `-` - Subtraction
- `*` - Multiplication
- `/` - Division
- `%` - Modulo

**Note:** Use parentheses `()` to control evaluation order. Without parentheses, operations are evaluated left-to-right.

**Examples:**

```yaml
# Instance
spec:
  replicas: 3
  increment: 2

# Template
minReplicas: $(.spec.replicas + 1)      # 4
maxReplicas: $(.spec.replicas * 3)      # 9
scaled: $((.spec.replicas + .spec.increment) * 2)  # 10
```

### String Concatenation

Combine strings and values:

```yaml
# Simple concatenation
fullName: $(.spec.prefix + "-" + .spec.name)

# With multiple parts
url: $(.spec.protocol + "://" + .spec.host + ":" + .spec.port)

# Mixing paths and literals
label: $(.metadata.namespace + "/" + .metadata.name)
```

**Examples:**

```yaml
# Instance
metadata:
  namespace: production
  name: my-app
spec:
  prefix: app
  version: v1

# Template
resourceName: $(.spec.prefix + "-" + .metadata.name)  # "app-my-app"
fullPath: $(.metadata.namespace + "/" + .metadata.name)  # "production/my-app"
imageTag: $(.metadata.name + ":" + .spec.version)  # "my-app:v1"
```

## Limitations

Current limitations (may be addressed in future versions):

1. **No operator precedence**: Arithmetic operations are evaluated left-to-right without standard precedence rules. Use parentheses to control order: `$((.spec.a + .spec.b) * .spec.c)` instead of `$(.spec.a + .spec.b * .spec.c)`
2. **No custom functions in templates**: Functions must be registered in Go code
3. **No array slicing**: Can't do `.spec.items[0:5]` (only single index access is supported)
4. **No external resource references**: Can only reference resources defined in the same template, not existing cluster resources
5. **No circular references**: Resources cannot reference each other in a loop

## Error Messages

Common errors and their meanings:

- `path must start with '.'` - Variable paths must begin with a dot
- `field 'X' not found in struct` - Referenced field doesn't exist
- `invalid for loop expression` - Loop syntax is incorrect
- `condition must evaluate to boolean` - Conditional expression didn't return true/false
- `iteration path must evaluate to an array` - Loop target is not a slice

## Future Enhancements

Planned features for future versions:

- **Array slicing**: Support for range syntax like `.spec.items[0:5]`
- **Operator precedence**: Standard mathematical operator precedence for arithmetic
- **Custom function registration in templates**: Define functions directly in YAML
- **Template includes/imports**: Import and reuse template fragments
- **Macro definitions**: Define reusable template macros
- **External resource references**: Reference existing cluster resources
- **Advanced loop controls**: `break`, `continue`, and loop indices
- **Pattern matching**: More sophisticated conditional logic

