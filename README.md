# KRM SDK - Client-Side Kubernetes Resource Model Framework

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

KRM SDK is a framework for building client-side Kubernetes abstractions. Think of it as **kubebuilder for client-side hydrators** - it provides the same developer experience as kubebuilder but for generating Kubernetes resources client-side instead of running controllers server-side.

## 🎯 What Problem Does This Solve?

Platform teams often need to provide simplified abstractions for application teams. Traditional approaches have limitations:

- **Helm**: Text templates are error-prone and hard to validate
- **Kustomize**: Limited abstraction capabilities
- **Server-side Operators**: Require cluster access and ongoing maintenance
- **Plain YAML**: Repetitive and error-prone

**KRM SDK** provides a better way: type-safe, validated, client-side abstractions that generate standard Kubernetes resources.

## ✨ Key Features

- 🔒 **Type-Safe APIs**: Define abstractions using Go structs with compile-time safety
- ✅ **Validation**: Kubebuilder markers enforce constraints before generation
- 📝 **Clean DSL**: YAML-native syntax with `$()` expressions (no Go text templates)
- 🔄 **Multi-Resource**: One abstraction expands to multiple K8s resources
- 🎨 **Overlays**: Native Kustomize integration for environment-specific customization
- 🔗 **Resource References**: Cross-resource field access within templates
- 📦 **Scaffolding**: Generate new projects and APIs with simple commands
- 🧪 **Testing Framework**: Comprehensive testing utilities and CI/CD

## 🚀 Quick Start

### Install the Framework CLI

```bash
go install github.com/yourusername/krm-sdk/cmd/krm-sdk@latest
```

### Create Your First Project

```bash
# Initialize a new project
krm-sdk init my-platform --domain platform.mycompany.com
cd my-platform

# Create a WebService abstraction
krm-sdk create api --group platform --version v1alpha1 --kind WebService

# Edit the generated files to add your fields and hydration logic
# - api/v1alpha1/web_service_types.go
# - api/v1alpha1/web_service_template.yaml

# Generate code and build
make generate
make build
```

### Use Your Abstraction

Create an instance (`instances/my-app.yaml`):

```yaml
apiVersion: platform.mycompany.com/v1alpha1
kind: WebService
metadata:
  name: my-app
  namespace: production
spec:
  image: nginx:1.25
  replicas: 3
  port: 80
  enableHA: true
```

Generate Kubernetes resources:

```bash
# Generate to stdout
./bin/my-platform generate -f instances/my-app.yaml

# Generate to directory
./bin/my-platform generate -f instances/my-app.yaml -o output/

# Validate first
./bin/my-platform validate -f instances/my-app.yaml

# Apply to cluster
./bin/my-platform generate -f instances/my-app.yaml | kubectl apply -f -
```

## 📖 Documentation

- **[Getting Started Guide](docs/getting-started.md)** - Step-by-step tutorial
- **[DSL Reference](docs/dsl-reference.md)** - Complete DSL syntax guide
- **[Example Project](examples/my-platform/EXAMPLE.md)** - Full working example

## 🏗️ Architecture

### Two-Component System

1. **Framework CLI** (`krm-sdk`): Scaffolding tool (like `kubebuilder init`)
2. **Project Binary**: Each project compiles its own hydrator (like a custom controller)

```
┌─────────────────────────────────────────────────────────────┐
│                      Platform Team                          │
│                                                             │
│  1. krm-sdk init my-platform                               │
│  2. krm-sdk create api --kind WebService                   │
│  3. Define Go structs + validation                         │
│  4. Define hydration templates                             │
│  5. make build → ./bin/my-platform                         │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    Application Teams                        │
│                                                             │
│  1. Write simple YAML (WebService instance)                │
│  2. ./bin/my-platform generate -f instance.yaml            │
│  3. Get full Deployment + Service + more                   │
│  4. kubectl apply -f output/                               │
└─────────────────────────────────────────────────────────────┘
```

## 🎨 DSL Syntax

The framework uses a clean, YAML-native DSL:

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
      preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchLabels:
              app: $(.metadata.name)
```

### Loops
```yaml
$for(container in .spec.containers):
  - name: $(container.name)
    image: $(container.image)
    ports:
      $for(port in container.ports):
        - containerPort: $(port)
```

### Functions
```yaml
labels:
  app: $(lower(.metadata.name))
  version: $(trim(.spec.version))
  hash: $(sha256(.spec.image))
```

### Resource References
```yaml
# Reference other resources in the same template
serviceIP: $(resource("v1", "Service", "my-app").spec.clusterIP)
servicePort: $(resource("v1", "Service", "my-app").spec.ports[0].port)
secretName: $(resource("v1", "Secret", .metadata.name + "-secret").metadata.name)
```

## 📊 Comparison with Other Tools

| Feature | KRM SDK | Kubebuilder | Helm | kpt |
|---------|---------|-------------|------|-----|
| **Client-side** | ✅ | ❌ | ✅ | ✅ |
| **Type-safe APIs** | ✅ | ✅ | ❌ | Partial |
| **Scaffolding** | ✅ | ✅ | ❌ | ❌ |
| **Go structs** | ✅ | ✅ | ❌ | ❌ |
| **Validation** | ✅ OpenAPI | ✅ Webhooks | ✅ JSON Schema | ✅ |
| **Templating** | ✅ DSL | ❌ | ✅ Go templates | ✅ Functions |
| **No cluster access needed** | ✅ | ❌ | ✅ | ✅ |
| **Composition** | ✅ | ❌ | ✅ | ✅ |

## 🔧 Project Structure

### Framework Repository (this project)
```
krm-sdk/
├── cmd/krm-sdk/          # Framework CLI
├── pkg/
│   ├── scaffold/         # Project scaffolding
│   ├── dsl/             # DSL parser & evaluator
│   ├── hydrator/        # Hydration engine
│   ├── validation/      # OpenAPI validation
│   └── cli/             # CLI runtime for projects
├── examples/            # Example projects
└── docs/               # Documentation
```

### Generated Project Structure
```
my-platform/
├── api/v1alpha1/
│   ├── webservice_types.go        # Go struct + validation
│   └── webservice_template.yaml   # Hydration template
├── cmd/my-platform/              # Project binary
├── config/crd/                   # Generated CRDs
├── instances/                    # Instance files
└── overlays/                     # Environment overlays
```

## 🎯 Use Cases

### Internal Developer Platforms
Create simplified abstractions for your organization:
- `WebService` - Standardized web app deployment
- `Database` - Managed database instances
- `CronJob` - Scheduled tasks with monitoring
- `StatefulApp` - Stateful applications with storage

### Multi-Tenancy
Generate tenant-specific resources with consistent policies:
- Network policies
- Resource quotas
- RBAC rules
- Service mesh configuration

### GitOps Workflows
Generate manifests for GitOps tools:
- Commit abstraction instances to Git
- CI generates full manifests
- GitOps tool (Flux/ArgoCD) applies to cluster
- Full audit trail and rollback capability

## 🤝 Contributing

Contributions are welcome! Areas where we'd love help:

- Additional DSL functions
- More example abstractions
- Documentation improvements
- Bug reports and fixes
- Feature requests

## 📝 License

Apache 2.0 - See [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

Inspired by:
- **Kubebuilder** - Project structure and scaffolding patterns
- **Helm** - Templating and packaging concepts
- **kpt** - KRM function pipeline ideas
- **Timoni** - CUE-based configuration approach
- **Kustomize** - Overlay and patching patterns

## 📬 Contact

- Issues: [GitHub Issues](https://github.com/yourusername/krm-sdk/issues)
- Discussions: [GitHub Discussions](https://github.com/yourusername/krm-sdk/discussions)

---

**Built with ❤️ for platform engineers who want type-safe, validated, client-side Kubernetes abstractions.**

