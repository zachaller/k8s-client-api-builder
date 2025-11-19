// Package v1alpha1 contains API Schema definitions for the platform v1alpha1 API group
// +kubebuilder:object:generate=true
// +groupName=platform.example.com
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "platform.example.com", Version: "v1alpha1"}

	// SchemeGroupVersion is an alias for GroupVersion (for compatibility)
	SchemeGroupVersion = GroupVersion
)
