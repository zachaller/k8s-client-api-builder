package validation

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

// Validator validates instances against CRD schemas
type Validator struct {
	crdDir  string
	schemas map[string]*apiextensionsv1.CustomResourceValidation
	verbose bool
}

// NewValidator creates a new validator
func NewValidator(crdDir string, verbose bool) *Validator {
	return &Validator{
		crdDir:  crdDir,
		schemas: make(map[string]*apiextensionsv1.CustomResourceValidation),
		verbose: verbose,
	}
}

// ValidationResult contains validation results
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// LoadSchemas loads CRD schemas from the CRD directory
func (v *Validator) LoadSchemas() error {
	if v.crdDir == "" {
		v.crdDir = "config/crd"
	}
	
	if _, err := os.Stat(v.crdDir); os.IsNotExist(err) {
		return fmt.Errorf("CRD directory not found: %s", v.crdDir)
	}
	
	files, err := ioutil.ReadDir(v.crdDir)
	if err != nil {
		return fmt.Errorf("failed to read CRD directory: %w", err)
	}
	
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}
		
		path := filepath.Join(v.crdDir, file.Name())
		if v.verbose {
			fmt.Printf("Loading CRD schema: %s\n", path)
		}
		
		if err := v.loadCRD(path); err != nil {
			return fmt.Errorf("failed to load CRD %s: %w", path, err)
		}
	}
	
	return nil
}

// loadCRD loads a single CRD file
func (v *Validator) loadCRD(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	
	var crd apiextensionsv1.CustomResourceDefinition
	if err := yaml.Unmarshal(data, &crd); err != nil {
		return fmt.Errorf("failed to parse CRD: %w", err)
	}
	
	// Extract validation schema for each version
	for _, version := range crd.Spec.Versions {
		key := fmt.Sprintf("%s/%s/%s", crd.Spec.Group, version.Name, crd.Spec.Names.Kind)
		
		if version.Schema != nil && version.Schema.OpenAPIV3Schema != nil {
			v.schemas[key] = &apiextensionsv1.CustomResourceValidation{
				OpenAPIV3Schema: version.Schema.OpenAPIV3Schema,
			}
			
			if v.verbose {
				fmt.Printf("Loaded schema for: %s\n", key)
			}
		}
	}
	
	return nil
}

// Validate validates an instance against its CRD schema
func (v *Validator) Validate(instance map[string]interface{}) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}
	
	// Extract metadata
	apiVersion, ok := instance["apiVersion"].(string)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "missing or invalid 'apiVersion' field")
		return result, nil
	}
	
	kind, ok := instance["kind"].(string)
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "missing or invalid 'kind' field")
		return result, nil
	}
	
	// Build schema key
	key := fmt.Sprintf("%s/%s", apiVersion, kind)
	
	schema, ok := v.schemas[key]
	if !ok {
		// Try to load schemas if not already loaded
		if len(v.schemas) == 0 {
			if err := v.LoadSchemas(); err != nil {
				return nil, fmt.Errorf("failed to load schemas: %w", err)
			}
			schema, ok = v.schemas[key]
		}
		
		if !ok {
			return nil, fmt.Errorf("schema not found for %s", key)
		}
	}
	
	// Validate against OpenAPI schema
	if schema.OpenAPIV3Schema != nil {
		u := &unstructured.Unstructured{Object: instance}
		
		validator, _, err := validation.NewSchemaValidator(schema.OpenAPIV3Schema)
		if err != nil {
			return nil, fmt.Errorf("failed to create validator: %w", err)
		}
		
		errs := validation.ValidateCustomResource(field.NewPath(""), u.Object, validator)
		if len(errs) > 0 {
			result.Valid = false
			for _, err := range errs {
				result.Errors = append(result.Errors, err.Error())
			}
		}
	}
	
	return result, nil
}

// ValidateFile validates an instance from a file
func (v *Validator) ValidateFile(path string) (*ValidationResult, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	var instance map[string]interface{}
	if err := yaml.Unmarshal(data, &instance); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	
	return v.Validate(instance)
}

