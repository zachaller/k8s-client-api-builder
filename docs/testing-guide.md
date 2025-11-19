# Testing Guide

This guide explains how to test KRM SDK projects and the framework itself.

## Overview

KRM SDK provides a comprehensive testing framework with three levels of testing:

1. **Unit Tests** - Test individual components
2. **Integration Tests** - Test end-to-end workflows
3. **Golden File Tests** - Verify template output

## Testing the Framework

### Running Tests

```bash
# Run all unit tests
make test

# Run integration tests
make test-integration

# Run all tests
make test-all

# Generate coverage report
make coverage
```

### Unit Test Coverage

Current coverage by package:
- `pkg/dsl/` - 51.1%
- `pkg/hydrator/` - 46.7%
- `pkg/overlay/` - 73.7%
- `pkg/scaffold/` - 50.0%
- `pkg/validation/` - 56.3%

### Writing Unit Tests

Example unit test for a new feature:

```go
package mypackage

import (
    "testing"
)

func TestMyFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "TEST",
            wantErr:  false,
        },
        {
            name:    "invalid input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := MyFeature(tt.input)

            if (err != nil) != tt.wantErr {
                t.Errorf("MyFeature() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr && result != tt.expected {
                t.Errorf("MyFeature() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

## Testing Generated Projects

### Test Framework

KRM SDK provides a testing framework for generated projects in `pkg/testing/`.

### Using the Test Framework

```go
package myapi_test

import (
    "testing"
    krmtesting "github.com/yourusername/krm-sdk/pkg/testing"
)

func TestWebServiceGeneration(t *testing.T) {
    framework := krmtesting.NewTestFramework(t)
    defer framework.Cleanup()

    // Initialize project
    err := framework.InitProject("test-project", "test.example.com")
    if err != nil {
        t.Fatalf("init failed: %v", err)
    }

    // Create API
    err = framework.CreateAPI("platform", "v1alpha1", "WebService")
    if err != nil {
        t.Fatalf("create api failed: %v", err)
    }

    // Build project
    err = framework.BuildProject()
    if err != nil {
        t.Fatalf("build failed: %v", err)
    }

    // Generate resources
    resources, err := framework.GenerateResources("instances/my-app.yaml")
    if err != nil {
        t.Fatalf("generation failed: %v", err)
    }

    // Validate output
    err = framework.ValidateOutput(resources, []krmtesting.Expectation{
        krmtesting.ExpectResource("Deployment", 1),
        krmtesting.ExpectResource("Service", 1),
    })
    if err != nil {
        t.Fatalf("validation failed: %v", err)
    }
}
```

### Expectations

The testing framework provides flexible expectations:

```go
// Expect specific number of resources
krmtesting.ExpectResource("Deployment", 1)

// Expect resource with specific name
krmtesting.ExpectResource("Deployment", 1).
    WithName("my-app")

// Expect resource with labels
krmtesting.ExpectResource("Deployment", 1).
    WithLabel("environment", "prod")

// Expect resource with custom checks
krmtesting.ExpectResource("Deployment", 1).
    WithCheck(krmtesting.HasField("spec", "replicas")).
    WithCheck(krmtesting.FieldEquals(5, "spec", "replicas"))
```

### Custom Checks

Create custom validation functions:

```go
func hasProperAffinity(resource map[string]interface{}) error {
    spec, ok := resource["spec"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("spec not found")
    }

    template, ok := spec["template"].(map[string]interface{})
    if !ok {
        return fmt.Errorf("template not found")
    }

    // Check for affinity
    if _, ok := template["affinity"]; !ok {
        return fmt.Errorf("affinity not configured")
    }

    return nil
}

// Use in test
expectation := krmtesting.ExpectResource("Deployment", 1).
    WithCheck(hasProperAffinity)
```

## Golden File Testing

### What are Golden Files?

Golden files are reference outputs that your tests compare against. They're useful for:
- Verifying template output doesn't change unexpectedly
- Documenting expected behavior
- Catching regressions

### Using Golden Files

```go
func TestTemplateOutput(t *testing.T) {
    // Generate output
    output := generateSomeYAML()

    // Compare with golden file
    krmtesting.CompareWithGolden(t, output, "testdata/golden/expected.yaml")
}
```

### Updating Golden Files

When you intentionally change output:

```bash
# Update all golden files
make test-golden-update

# Or with go test directly
go test ./... -update
```

### Golden File Organization

```
testdata/
├── golden/
│   ├── webservice-basic.yaml
│   ├── webservice-ha.yaml
│   ├── webservice-overlay-dev.yaml
│   └── webservice-overlay-prod.yaml
└── inputs/
    ├── webservice-basic.yaml
    └── webservice-ha.yaml
```

## Integration Testing

### End-to-End Tests

Test complete workflows:

```go
func TestCompleteWorkflow(t *testing.T) {
    framework := krmtesting.NewTestFramework(t)
    defer framework.Cleanup()

    // 1. Initialize project
    if err := framework.InitProject("my-platform", "platform.example.com"); err != nil {
        t.Fatalf("init failed: %v", err)
    }

    // 2. Create API
    if err := framework.CreateAPI("platform", "v1alpha1", "WebService"); err != nil {
        t.Fatalf("create api failed: %v", err)
    }

    // 3. Build project
    if err := framework.BuildProject(); err != nil {
        t.Fatalf("build failed: %v", err)
    }

    // 4. Generate resources
    resources, err := framework.GenerateResources("instances/test.yaml")
    if err != nil {
        t.Fatalf("generation failed: %v", err)
    }

    // 5. Validate
    if len(resources) == 0 {
        t.Error("no resources generated")
    }
}
```

### Testing with Overlays

```go
func TestOverlayApplication(t *testing.T) {
    framework := krmtesting.NewTestFramework(t)
    defer framework.Cleanup()

    // ... setup project ...

    // Generate with dev overlay
    devResources, err := framework.GenerateWithOverlay(
        "instances/my-app.yaml",
        "dev",
    )
    if err != nil {
        t.Fatalf("dev overlay failed: %v", err)
    }

    // Generate with prod overlay
    prodResources, err := framework.GenerateWithOverlay(
        "instances/my-app.yaml",
        "prod",
    )
    if err != nil {
        t.Fatalf("prod overlay failed: %v", err)
    }

    // Verify differences
    // Dev should have 1 replica, prod should have 5
}
```

## Best Practices

### 1. Use Table-Driven Tests

```go
tests := []struct {
    name     string
    input    Input
    expected Output
    wantErr  bool
}{
    // Test cases here
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic
    })
}
```

### 2. Clean Up Resources

Always clean up test artifacts:

```go
func TestSomething(t *testing.T) {
    framework := krmtesting.NewTestFramework(t)
    defer framework.Cleanup()  // Always cleanup

    // Test logic
}
```

### 3. Use Subtests

Group related tests:

```go
func TestWebService(t *testing.T) {
    t.Run("basic generation", func(t *testing.T) {
        // Test basic case
    })

    t.Run("with HA enabled", func(t *testing.T) {
        // Test HA case
    })
}
```

### 4. Test Error Cases

Don't just test the happy path:

```go
func TestValidation(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid input", "valid", false},
        {"empty input", "", true},
        {"invalid format", "###", true},
    }
    // ...
}
```

### 5. Use Helpers

Create helper functions for common setup:

```go
func setupTestProject(t *testing.T) *krmtesting.TestFramework {
    t.Helper()

    framework := krmtesting.NewTestFramework(t)
    
    if err := framework.InitProject("test", "test.example.com"); err != nil {
        t.Fatalf("setup failed: %v", err)
    }

    return framework
}
```

## CI/CD Integration

### GitHub Actions

The framework includes a GitHub Actions workflow (`.github/workflows/test.yml`):

```yaml
name: Tests
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make test
  
  integration-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: make test-integration
```

### Running Tests Locally

Before pushing:

```bash
# Run all tests
make test-all

# Check formatting
make fmt
make vet

# Build
make build
```

## Troubleshooting

### Tests Fail with "binary not found"

**Solution**: Build the framework first:
```bash
make build
```

### Tests Fail with "module not found"

**Solution**: Ensure dependencies are installed:
```bash
go mod download
go mod tidy
```

### Golden Files Out of Date

**Solution**: Update golden files:
```bash
make test-golden-update
```

### Integration Tests Timeout

**Solution**: Increase timeout:
```bash
go test ./test/integration/... -timeout 10m
```

## Test Coverage Goals

- **Overall**: > 80%
- **Critical Packages** (dsl, hydrator, overlay): > 70%
- **New Features**: > 80%

### Checking Coverage

```bash
# Generate coverage report
make coverage

# Open in browser
open coverage.html
```

## Example Test Suite

Complete example for a new abstraction:

```go
package v1alpha1_test

import (
    "testing"
    krmtesting "github.com/yourusername/krm-sdk/pkg/testing"
)

func TestDatabaseHydration(t *testing.T) {
    tests := []struct {
        name     string
        instance string
        expected []krmtesting.Expectation
    }{
        {
            name:     "postgres small",
            instance: "testdata/database-postgres-small.yaml",
            expected: []krmtesting.Expectation{
                krmtesting.ExpectResource("StatefulSet", 1).
                    WithCheck(krmtesting.FieldEquals(int64(1), "spec", "replicas")),
                krmtesting.ExpectResource("Service", 1),
                krmtesting.ExpectResource("PersistentVolumeClaim", 1),
            },
        },
        {
            name:     "postgres large",
            instance: "testdata/database-postgres-large.yaml",
            expected: []krmtesting.Expectation{
                krmtesting.ExpectResource("StatefulSet", 1).
                    WithCheck(krmtesting.FieldEquals(int64(3), "spec", "replicas")),
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            framework := krmtesting.NewTestFramework(t)
            defer framework.Cleanup()

            resources, err := framework.GenerateResources(tt.instance)
            if err != nil {
                t.Fatalf("generation failed: %v", err)
            }

            if err := framework.ValidateOutput(resources, tt.expected); err != nil {
                t.Fatalf("validation failed: %v", err)
            }
        })
    }
}
```

## Summary

- ✅ Use `make test` for quick unit tests
- ✅ Use `make test-all` before committing
- ✅ Write tests for new features
- ✅ Use golden files for template output
- ✅ Clean up test artifacts
- ✅ Aim for >80% coverage

Testing ensures your abstractions work correctly and don't break over time!

