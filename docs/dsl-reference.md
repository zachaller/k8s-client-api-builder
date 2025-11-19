# DSL Reference

The KRM SDK uses a declarative YAML-based DSL with inline expressions for hydrating abstractions into Kubernetes resources.

## Design Principles

1. **YAML-native**: Looks and feels like regular YAML
2. **Clear syntax**: Easy to distinguish between static content and dynamic expressions
3. **Type-safe**: Expressions are validated against Go struct schemas
4. **Composable**: Supports conditionals, loops, and functions

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

# Array index (not yet supported)
# firstItem: $(.spec.items[0])
```

### Conditionals

Use `$if(condition):` to conditionally include content:

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
# Basic loop
$for(item in .spec.items):
  - name: $(item.name)
    value: $(item.value)

# Nested loops
$for(service in .spec.services):
  - name: $(service.name)
    ports:
      $for(port in service.ports):
        - port: $(port)
```

**Loop Variable Scope:**
- Loop variables are only available within the loop body
- Outer instance fields are still accessible: `$(.metadata.name)`
- Loop variables shadow outer variables with the same name

### Functions

Use `$(function(args))` to transform values:

```yaml
labels:
  app: $(lower(.metadata.name))
  version: $(trim(.spec.version))
  hash: $(sha256(.spec.image))
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

### Combining Functions

```yaml
labels:
  app: $(lower(.metadata.name))
  hash: $(sha256(lower(.spec.image)))
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

## Best Practices

1. **Keep templates readable**: Don't nest too deeply
2. **Use meaningful variable names**: `$for(container in .spec.containers)` not `$for(c in .spec.containers)`
3. **Validate early**: Use kubebuilder markers to enforce constraints
4. **Document complex logic**: Add comments in templates
5. **Test thoroughly**: Create sample instances for all code paths

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
3. **No nested array slicing**: Can't do `.spec.items[0:5]`

## Error Messages

Common errors and their meanings:

- `path must start with '.'` - Variable paths must begin with a dot
- `field 'X' not found in struct` - Referenced field doesn't exist
- `invalid for loop expression` - Loop syntax is incorrect
- `condition must evaluate to boolean` - Conditional expression didn't return true/false
- `iteration path must evaluate to an array` - Loop target is not a slice

## Future Enhancements

Planned features for future versions:

- Array indexing and slicing
- Arithmetic expressions
- String concatenation operator
- Custom function registration in templates
- Template includes/imports
- Macro definitions

