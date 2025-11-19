package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// APIConfig holds configuration for API scaffolding
type APIConfig struct {
	Group   string
	Version string
	Kind    string
	Verbose bool
}

// APIScaffolder handles API type scaffolding
type APIScaffolder struct {
	config APIConfig
}

// NewAPIScaffolder creates a new API scaffolder
func NewAPIScaffolder(config APIConfig) *APIScaffolder {
	return &APIScaffolder{config: config}
}

// Scaffold creates a new API type
func (s *APIScaffolder) Scaffold() error {
	// Check if we're in a project directory
	if _, err := os.Stat("PROJECT"); err != nil {
		return fmt.Errorf("not in a KRM project directory (PROJECT file not found)")
	}

	// Read project config
	projectData, err := os.ReadFile("PROJECT")
	if err != nil {
		return fmt.Errorf("failed to read PROJECT file: %w", err)
	}

	// Parse domain from PROJECT file
	domain := ""
	for _, line := range strings.Split(string(projectData), "\n") {
		if strings.HasPrefix(line, "domain:") {
			domain = strings.TrimSpace(strings.TrimPrefix(line, "domain:"))
			break
		}
	}

	if domain == "" {
		return fmt.Errorf("domain not found in PROJECT file")
	}

	apiDir := filepath.Join("api", s.config.Version)

	// Create API directory if it doesn't exist
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return fmt.Errorf("failed to create API directory: %w", err)
	}

	// Generate files
	snakeName := ToSnakeCase(s.config.Kind)

	files := map[string]string{
		filepath.Join(apiDir, snakeName+"_types.go"):       s.generateTypesFile(domain),
		filepath.Join(apiDir, snakeName+"_template.yaml"):  s.generateTemplateFile(),
		filepath.Join("config/samples", snakeName+".yaml"): s.generateSampleFile(domain),
	}

	for filename, content := range files {
		if s.config.Verbose {
			fmt.Printf("Creating file: %s\n", filename)
		}
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	// Update register.go to include new type
	if err := s.updateRegister(apiDir); err != nil {
		return fmt.Errorf("failed to update register.go: %w", err)
	}

	return nil
}

func (s *APIScaffolder) generateTypesFile(domain string) string {
	return fmt.Sprintf(`package %s

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// %sSpec defines the desired state of %s
type %sSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS HERE
	// Add your fields with kubebuilder validation markers
	
	// Example:
	// +kubebuilder:validation:MinLength=1
	// Name string `+"`json:\"name\"`"+`
}

// %sStatus defines the observed state of %s
type %sStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS HERE
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// %s is the Schema for the %s API
type %s struct {
	metav1.TypeMeta   `+"`json:\",inline\"`"+`
	metav1.ObjectMeta `+"`json:\"metadata,omitempty\"`"+`

	Spec   %sSpec   `+"`json:\"spec,omitempty\"`"+`
	Status %sStatus `+"`json:\"status,omitempty\"`"+`
}

// +kubebuilder:object:root=true

// %sList contains a list of %s
type %sList struct {
	metav1.TypeMeta `+"`json:\",inline\"`"+`
	metav1.ListMeta `+"`json:\"metadata,omitempty\"`"+`
	Items           []%s `+"`json:\"items\"`"+`
}

func init() {
	SchemeBuilder.Register(&%s{}, &%sList{})
}
`, s.config.Version, s.config.Kind, s.config.Kind, s.config.Kind,
		s.config.Kind, s.config.Kind, s.config.Kind,
		s.config.Kind, ToLowerPlural(s.config.Kind), s.config.Kind,
		s.config.Kind, s.config.Kind,
		s.config.Kind, s.config.Kind, s.config.Kind,
		s.config.Kind, s.config.Kind, s.config.Kind)
}

func (s *APIScaffolder) generateTemplateFile() string {
	return fmt.Sprintf(`# Hydration template for %s
# This template defines how a %s instance expands into Kubernetes resources

resources:
  # Example: Generate a Deployment
  - apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: $(.metadata.name)
      namespace: $(.metadata.namespace)
      labels:
        app: $(.metadata.name)
        managed-by: %s
    spec:
      replicas: 1
      selector:
        matchLabels:
          app: $(.metadata.name)
      template:
        metadata:
          labels:
            app: $(.metadata.name)
        spec:
          containers:
          - name: main
            image: nginx:latest
            # Add more configuration based on your spec fields
            # Example: image: $(.spec.image)

  # Example: Generate a Service
  - apiVersion: v1
    kind: Service
    metadata:
      name: $(.metadata.name)
      namespace: $(.metadata.namespace)
      labels:
        app: $(.metadata.name)
    spec:
      selector:
        app: $(.metadata.name)
      ports:
      - port: 80
        targetPort: 80
        # Example: port: $(.spec.port)

# You can add more resources as needed
# Supported DSL features:
#   - Variable substitution: $(.path.to.field)
#   - Conditionals: $if(.spec.condition):
#   - Loops: $for(item in .spec.items):
#   - Functions: $(lower(.metadata.name))
`, s.config.Kind, s.config.Kind, ToLowerPlural(s.config.Kind))
}

func (s *APIScaffolder) generateSampleFile(domain string) string {
	return fmt.Sprintf(`apiVersion: %s/%s
kind: %s
metadata:
  name: %s-sample
  namespace: default
spec:
  # Add your spec fields here
  # Example:
  # name: example
  # replicas: 3
`, domain, s.config.Version, s.config.Kind, ToSnakeCase(s.config.Kind))
}

func (s *APIScaffolder) updateRegister(apiDir string) error {
	registerPath := filepath.Join(apiDir, "register.go")

	// Read existing register.go if it exists
	content, err := os.ReadFile(registerPath)
	if err != nil {
		// File doesn't exist, create it
		return nil
	}

	// Check if the type is already registered
	if strings.Contains(string(content), "&"+s.config.Kind+"{") {
		// Already registered
		return nil
	}

	// Update the addKnownTypes function to include the new type
	// This is a simple approach - in production you'd want more sophisticated parsing
	updated := strings.Replace(string(content),
		"func addKnownTypes(scheme *runtime.Scheme) error {\n\tscheme.AddKnownTypes(SchemeGroupVersion",
		fmt.Sprintf("func addKnownTypes(scheme *runtime.Scheme) error {\n\tscheme.AddKnownTypes(SchemeGroupVersion,\n\t\t&%s{},\n\t\t&%sList{}",
			s.config.Kind, s.config.Kind),
		1)

	return os.WriteFile(registerPath, []byte(updated), 0644)
}

// ToSnakeCase converts a string to snake_case
func ToSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// ToLowerPlural converts a Kind name to lowercase plural
func ToLowerPlural(s string) string {
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, "s") {
		return lower + "es"
	}
	return lower + "s"
}
