# KRM SDK - Project Context

## Project Overview

**KRM SDK** is a framework for building client-side Kubernetes abstractions. It's essentially **kubebuilder for client-side hydrators** - providing the same developer experience as kubebuilder but for generating Kubernetes resources client-side instead of running controllers server-side.

### Core Concept

The framework enables platform teams to create type-safe, validated abstractions (like `WebService`, `Database`) that expand into multiple Kubernetes resources using a custom YAML-based DSL.

## Architecture

### Two-Component System

1. **Framework CLI (`krm-sdk`)** - Scaffolding tool
   - Installed once: `go install .../krm-sdk`
   - Used to create and manage projects
   - Commands: `init`, `create api`, `generate`, `version`

2. **Project Binary** (generated per project)
   - Built from generated project
   - Contains project-specific abstractions
   - Commands: `generate`, `validate`, `apply`
   - Validates and hydrates instances

### Data Flow

```
Instance YAML (user input)
    ↓
Validation (OpenAPI Schema)
    ↓
Hydration (DSL Template)
    ↓
Generated K8s Resources
    ↓
kubectl apply
```

## Technology Stack

- **Language**: Go 1.21+
- **CLI Framework**: cobra + viper
- **K8s APIs**: k8s.io/apimachinery, k8s.io/apiextensions-apiserver
- **YAML**: sigs.k8s.io/yaml, gopkg.in/yaml.v3
- **Code Gen**: sigs.k8s.io/controller-tools (controller-gen)
- **Validation**: OpenAPI v3 schemas

## Project Structure

```
krm-sdk/
├── cmd/krm-sdk/              # Framework CLI
│   ├── main.go
│   └── commands/             # CLI commands (init, create, generate, version)
├── pkg/
│   ├── scaffold/             # Project scaffolding
│   │   ├── project.go       # Generate new projects
│   │   ├── api.go           # Generate new API types
│   │   └── templates/       # Code generation templates
│   ├── dsl/                  # DSL engine ⭐
│   │   ├── parser.go        # Parse DSL expressions
│   │   └── evaluator.go     # Evaluate expressions
│   ├── hydrator/             # Hydration engine
│   │   └── hydrator.go      # Process templates, generate resources
│   ├── validation/           # Validation
│   │   └── validator.go     # OpenAPI schema validation
│   ├── cli/                  # CLI runtime for generated projects
│   │   ├── generator.go     # Resource generation logic
│   │   └── commands.go      # CLI commands for projects
│   └── overlay/              # Kustomize-style overlays (planned)
├── examples/
│   ├── my-platform/          # Example project with WebService
│   └── advanced-dsl-features.md
├── docs/
│   ├── getting-started.md
│   └── dsl-reference.md
├── hack/
│   └── tools.go              # Build tool dependencies
├── Makefile
├── go.mod
└── README.md
```

## DSL (Domain-Specific Language)

### Design Philosophy

- **YAML-native**: Looks and feels like regular YAML
- **Clear syntax**: Easy to distinguish static content from dynamic expressions
- **Type-safe**: Expressions validated against Go struct schemas
- **No Go text templates**: Custom syntax designed for Kubernetes

### Syntax Overview

#### Variable Substitution
```yaml
name: $(.metadata.name)
namespace: $(.metadata.namespace)
replicas: $(.spec.replicas)
```

#### Conditionals
```yaml
$if(.spec.enableHA):
  affinity:
    podAntiAffinity:
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: $(.metadata.name)
```

#### Loops
```yaml
containers:
  $for(container in .spec.containers):
    - name: $(container.name)
      image: $(container.image)
      ports:
        $for(port in container.ports):
          - containerPort: $(port)
```

#### Functions
```yaml
labels:
  app: $(lower(.metadata.name))
  version: $(trim(.spec.version))
  hash: $(sha256(.spec.image))
```

#### Array Indexing ⭐ NEW
```yaml
firstItem: $(.spec.items[0])
selectedItem: $(.spec.items[.spec.selectedIndex])
container: $(.spec.pods[0].containers[1])
```

#### Arithmetic Operations ⭐ NEW
```yaml
total: $(.spec.base + .spec.increment)
doubled: $(.spec.replicas * 2)
result: $((.spec.a + .spec.b) * .spec.c)
```

#### String Concatenation ⭐ NEW
```yaml
fullName: $(.spec.prefix + "-" + .spec.name)
url: $(.spec.protocol + "://" + .spec.host + ":" + .spec.port)
```

### Built-in Functions

- `lower(string)` - Convert to lowercase
- `upper(string)` - Convert to uppercase
- `trim(string)` - Remove whitespace
- `replace(string, old, new)` - Replace all occurrences
- `sha256(string)` - Compute SHA256 hash
- `default(value, defaultValue)` - Return default if value is empty

## Key Components

### 1. Parser (`pkg/dsl/parser.go`)

**Purpose**: Tokenize and parse DSL expressions

**Expression Types**:
- `ExprPath` - Simple path like `.spec.name`
- `ExprFunction` - Function call like `lower(.spec.name)`
- `ExprBinary` - Binary operation like `.spec.a + .spec.b`
- `ExprLiteral` - Literal value like `"hello"` or `42`
- `ExprArrayIndex` - Array indexing like `.spec.items[0]`
- `ExprConcat` - String concatenation like `.a + "-" + .b`

**Key Functions**:
- `ParseExpression()` - Main entry point
- `parseArrayIndexExpr()` - Parse array indexing
- `tryParseConcatExpr()` - Parse concatenation
- `parseBinaryExpr()` - Parse binary operations
- `parseFunctionExpr()` - Parse function calls

**Parsing Strategy**:
1. Check for concatenation (if contains `+` with quotes)
2. Check for arithmetic (operators: `-`, `*`, `/`, `%`, `+`)
3. Check for functions (but not parenthesized expressions)
4. Check for array indexing
5. Check for comparisons
6. Fall back to paths or literals

### 2. Evaluator (`pkg/dsl/evaluator.go`)

**Purpose**: Evaluate parsed expressions against data

**Key Functions**:
- `Evaluate()` - Main evaluation dispatcher
- `EvaluateString()` - Evaluate string with `$()` substitutions
- `evaluatePath()` - Navigate through data structure
- `evaluateFunction()` - Execute built-in functions
- `evaluateBinary()` - Execute binary operations
- `evaluateArrayIndex()` - Access array elements
- `evaluateConcat()` - Concatenate strings
- `performArithmetic()` - Arithmetic operations

**Type Conversions**:
- `toFloat64()` - Convert to float for arithmetic
- `toInt()` - Convert to integer for array indexing

### 3. Hydrator (`pkg/hydrator/hydrator.go`)

**Purpose**: Process templates and generate K8s resources

**Key Functions**:
- `Hydrate()` - Main hydration entry point
- `processResource()` - Process single resource template
- `processValue()` - Recursively process values
- `processConditional()` - Handle `$if()` blocks
- `processLoop()` - Handle `$for()` blocks

**Template Structure**:
```yaml
resources:
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: $(.metadata.name)
    spec:
      replicas: $(.spec.replicas)
```

### 4. Validator (`pkg/validation/validator.go`)

**Purpose**: Validate instances against OpenAPI schemas

**Key Functions**:
- `LoadSchemas()` - Load CRD schemas from config/crd/
- `Validate()` - Validate instance against schema
- `ValidateFile()` - Validate from file

**Validation Flow**:
1. Load CRD manifests
2. Extract OpenAPI v3 schemas
3. Validate instance structure
4. Check field constraints (min/max, patterns, etc.)

### 5. Scaffolder (`pkg/scaffold/`)

**Purpose**: Generate new projects and API types

**Project Scaffolder** (`project.go`):
- Creates directory structure
- Generates go.mod, Makefile, README
- Sets up API directories
- Creates main.go and commands

**API Scaffolder** (`api.go`):
- Generates Go struct files with kubebuilder markers
- Creates hydration template YAML
- Updates registration code
- Generates sample instances

## Generated Project Structure

When you run `krm-sdk init my-platform`:

```
my-platform/
├── cmd/my-platform/
│   ├── main.go
│   └── commands/
│       └── root.go           # CLI commands (uses pkg/cli)
├── api/v1alpha1/
│   ├── groupversion_info.go  # API metadata
│   ├── register.go           # Scheme registration
│   ├── webservice_types.go   # Go struct with validations
│   └── webservice_template.yaml  # Hydration template
├── config/
│   ├── crd/                  # Generated CRD manifests
│   └── samples/              # Sample instances
├── instances/                # User instance files
├── overlays/                 # Environment-specific overlays
│   ├── dev/
│   ├── staging/
│   └── prod/
├── hack/
│   └── boilerplate.go.txt
├── Makefile
├── go.mod
├── PROJECT                   # Project metadata
└── README.md
```

## Workflow

### Platform Team (Framework Users)

1. **Install Framework**:
   ```bash
   go install github.com/yourusername/krm-sdk/cmd/krm-sdk@latest
   ```

2. **Create Project**:
   ```bash
   krm-sdk init my-platform --domain platform.mycompany.com
   cd my-platform
   ```

3. **Create Abstraction**:
   ```bash
   krm-sdk create api --group platform --version v1alpha1 --kind WebService
   ```

4. **Define API** (edit `api/v1alpha1/webservice_types.go`):
   ```go
   type WebServiceSpec struct {
       // +kubebuilder:validation:MinLength=1
       Image string `json:"image"`
       
       // +kubebuilder:validation:Minimum=1
       // +kubebuilder:validation:Maximum=100
       Replicas int32 `json:"replicas"`
       
       // +kubebuilder:validation:Minimum=1
       // +kubebuilder:validation:Maximum=65535
       Port int32 `json:"port"`
   }
   ```

5. **Define Template** (edit `api/v1alpha1/webservice_template.yaml`):
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
   ```

6. **Generate & Build**:
   ```bash
   make generate  # Generate CRDs
   make build     # Build project binary
   ```

### Application Team (Platform Users)

1. **Create Instance**:
   ```yaml
   # instances/my-app.yaml
   apiVersion: platform.mycompany.com/v1alpha1
   kind: WebService
   metadata:
     name: my-app
     namespace: production
   spec:
     image: nginx:1.25
     replicas: 3
     port: 80
   ```

2. **Generate Resources**:
   ```bash
   ./bin/my-platform generate -f instances/my-app.yaml
   ```

3. **Apply to Cluster**:
   ```bash
   ./bin/my-platform generate -f instances/my-app.yaml | kubectl apply -f -
   ```

## Recent Enhancements

### Phase 1: DSL Advanced Features

Three major features were added to the DSL:

### Phase 2: Overlay System and Testing Framework

Two major features were added:

### Phase 3: Resource References

One major feature was added:

1. **Array Indexing**
   - Access array elements: `$(.spec.items[0])`
   - Variable indices: `$(.spec.items[.spec.index])`
   - Nested arrays: `$(.spec.pods[0].containers[1])`

2. **Arithmetic Operations**
   - Basic operations: `+`, `-`, `*`, `/`, `%`
   - Parentheses for precedence: `$((.a + .b) * .c)`
   - Type conversion and error handling

3. **String Concatenation**
   - Multiple elements: `$(.prefix + "-" + .name)`
   - Auto type-to-string conversion
   - Efficient string building

**Files Modified**:
- `pkg/dsl/parser.go` - Enhanced parsing
- `pkg/dsl/evaluator.go` - New evaluation logic
- `pkg/dsl/dsl_test.go` - Comprehensive tests
- `docs/dsl-reference.md` - Updated docs
- `examples/advanced-dsl-features.md` - Examples

1. **Native Kustomize Integration**
   - Uses `sigs.k8s.io/kustomize/api` directly
   - Flexible overlay paths (directory or kustomization.yaml file)
   - Full kustomize feature support
   - Auto-generates overlay structure in new projects

2. **Comprehensive Testing Framework**
   - Test framework package (`pkg/testing/`)
   - Unit tests for all packages (55% average coverage)
   - Integration test suite
   - Golden file testing support
   - CI/CD pipeline with GitHub Actions
   - Makefile test targets

**Files Created**:
- `pkg/overlay/kustomize.go` - Kustomize engine
- `pkg/overlay/kustomize_test.go` - Overlay tests
- `pkg/testing/framework.go` - Test framework
- `pkg/testing/expectations.go` - Validation helpers
- `pkg/testing/golden.go` - Golden file support
- `pkg/hydrator/hydrator_test.go` - Hydrator tests
- `pkg/validation/validator_test.go` - Validation tests
- `pkg/scaffold/project_test.go` - Scaffold tests
- `test/integration/` - Integration test suite
- `.github/workflows/test.yml` - CI/CD pipeline
- `docs/overlay-guide.md` - Overlay documentation
- `docs/testing-guide.md` - Testing documentation

1. **Resource References**
   - Syntax: `$(resource(apiVersion, kind, name).path.to.field)`
   - Two-pass hydration processing
   - Cross-resource field access
   - Supports expressions in names
   - Array indexing in field paths

**Files Created/Modified**:
- `pkg/dsl/parser.go` - Added ResourceRef parsing
- `pkg/dsl/evaluator.go` - Resource registry and evaluation
- `pkg/hydrator/hydrator.go` - Two-pass processing
- `pkg/hydrator/registry.go` - Resource registry (new)
- `pkg/dsl/resource_ref_test.go` - Tests (new)
- `docs/dsl-reference.md` - Resource reference documentation
- `examples/resource-references-example.yaml` - Example (new)

## Key Design Decisions

### 1. Client-Side vs Server-Side

**Decision**: Client-side generation
**Rationale**: 
- No cluster access required
- Easier to integrate with CI/CD
- No ongoing maintenance of controllers
- Simpler security model

### 2. DSL Syntax

**Decision**: `$()` syntax instead of `{{ }}`
**Rationale**:
- More YAML-native appearance
- Clear distinction from Go templates
- Easier to parse unambiguously
- Better IDE support potential

### 3. Type Safety

**Decision**: Go structs with kubebuilder markers
**Rationale**:
- Compile-time type checking
- Familiar to Kubernetes developers
- Reuses existing tooling (controller-gen)
- OpenAPI schema generation

### 4. Two-Component Architecture

**Decision**: Framework CLI + Project Binary
**Rationale**:
- Separation of concerns
- Each project is independent
- Easy distribution to users
- Familiar pattern (like kubebuilder)

## Common Patterns

### Pattern 1: Simple Abstraction

**Use Case**: Standardize web service deployments

**Go Struct**:
```go
type WebServiceSpec struct {
    Image    string `json:"image"`
    Replicas int32  `json:"replicas"`
    Port     int32  `json:"port"`
}
```

**Template**:
```yaml
resources:
  - apiVersion: apps/v1
    kind: Deployment
    # ... deployment spec
  - apiVersion: v1
    kind: Service
    # ... service spec
```

### Pattern 2: Conditional Features

**Use Case**: Optional high availability

**Go Struct**:
```go
type WebServiceSpec struct {
    // ... other fields
    EnableHA bool `json:"enableHA,omitempty"`
}
```

**Template**:
```yaml
spec:
  $if(.spec.enableHA):
    affinity:
      podAntiAffinity:
        # ... HA configuration
```

### Pattern 3: Dynamic Lists

**Use Case**: Multiple containers

**Go Struct**:
```go
type Container struct {
    Name  string `json:"name"`
    Image string `json:"image"`
    Port  int32  `json:"port"`
}

type MultiContainerSpec struct {
    Containers []Container `json:"containers"`
}
```

**Template**:
```yaml
containers:
  $for(container in .spec.containers):
    - name: $(container.name)
      image: $(container.image)
      ports:
      - containerPort: $(container.port)
```

### Pattern 4: Calculated Values

**Use Case**: Dynamic scaling

**Go Struct**:
```go
type ScalableSpec struct {
    BaseReplicas int32 `json:"baseReplicas"`
    ScaleFactor  int32 `json:"scaleFactor"`
}
```

**Template**:
```yaml
spec:
  replicas: $(.spec.baseReplicas * .spec.scaleFactor)
```

## Testing

### Unit Tests

**DSL Tests** (`pkg/dsl/dsl_test.go`):
- Array indexing (4 test cases)
- Arithmetic operations (6 test cases)
- String concatenation (3 test cases)
- Combined features (2 test cases)

**Run Tests**:
```bash
go test -v ./pkg/dsl/
```

### Integration Testing

Create test projects and verify:
1. Project scaffolding
2. API generation
3. Template hydration
4. Validation

## Build & Development

### Build Framework
```bash
make build          # Build krm-sdk binary
make test           # Run tests
make fmt            # Format code
make vet            # Run go vet
```

### Create Example Project
```bash
./bin/krm-sdk init test-platform --domain test.example.com
cd test-platform
../bin/krm-sdk create api --group test --version v1alpha1 --kind MyService
make generate
make build
```

## Troubleshooting

### Common Issues

1. **"not in a KRM project directory"**
   - Ensure you're in a directory with a PROJECT file
   - Run `krm-sdk init` first

2. **"template not found"**
   - Check template file exists: `api/{version}/{kind}_template.yaml`
   - Verify filename uses snake_case

3. **"validation failed"**
   - Check instance matches Go struct definition
   - Verify kubebuilder markers are correct
   - Run `make generate` after changing types

4. **"failed to parse expression"**
   - Check DSL syntax: `$(expression)` not `{{expression}}`
   - Verify paths start with `.`
   - Check for balanced parentheses

## Future Enhancements

### Planned Features

1. **Overlay System**: Full kustomize-style overlay implementation
2. **Apply Command**: Direct cluster application via client-go
3. **Composition**: Examples of abstractions referencing abstractions
4. **Testing Framework**: Unit and integration test helpers

### Potential DSL Enhancements

1. **Operator Precedence**: Proper precedence rules for arithmetic
2. **Array Slicing**: Support `[start:end]` syntax
3. **Nested Functions**: Allow `$(func1(func2(.path)))`
4. **Boolean Operators**: `&&`, `||`, `!`
5. **Ternary Operator**: `$(condition ? true : false)`
6. **Array Methods**: `.length`, `.first`, `.last`

## Resources

### Documentation
- [README.md](../README.md) - Project overview
- [Getting Started](../docs/getting-started.md) - Tutorial
- [DSL Reference](../docs/dsl-reference.md) - Complete DSL syntax
- [Example Project](../examples/my-platform/EXAMPLE.md) - Working example
- [Advanced DSL](../examples/advanced-dsl-features.md) - New features

### Code Entry Points
- Framework CLI: `cmd/krm-sdk/main.go`
- DSL Parser: `pkg/dsl/parser.go`
- DSL Evaluator: `pkg/dsl/evaluator.go`
- Hydrator: `pkg/hydrator/hydrator.go`
- Validator: `pkg/validation/validator.go`

### External Dependencies
- Cobra: CLI framework
- Controller-tools: CRD generation
- K8s API machinery: Types and validation
- YAML libraries: Parsing and serialization

## Contributing Guidelines

### Code Style
- Follow Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Write tests for new features

### Adding DSL Features
1. Update parser to recognize new syntax
2. Add new expression type if needed
3. Implement evaluator logic
4. Add comprehensive tests
5. Update documentation
6. Add examples

### Project Structure
- Keep packages focused and cohesive
- Avoid circular dependencies
- Use interfaces for testability
- Document public APIs

## Contact & Support

- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Documentation**: See `docs/` directory

---

**Last Updated**: 2024
**Framework Version**: v0.1.0 (development)
**Go Version**: 1.21+
**Status**: Active Development

