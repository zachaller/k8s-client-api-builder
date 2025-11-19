package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WebServiceSpec defines the desired state of WebService
type WebServiceSpec struct {
	// Image is the container image to deploy
	// +kubebuilder:validation:MinLength=1
	Image string `json:"image"`
	
	// Replicas is the number of pod replicas
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas,omitempty"`
	
	// Port is the container port to expose
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
	
	// EnableIngress enables ingress creation
	// +kubebuilder:default=false
	EnableIngress bool `json:"enableIngress,omitempty"`
	
	// Domain is the domain for the ingress (required if enableIngress is true)
	// +optional
	Domain string `json:"domain,omitempty"`
}

// WebServiceStatus defines the observed state of WebService
type WebServiceStatus struct {
	// INSERT ADDITIONAL STATUS FIELDS HERE
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced

// WebService is the Schema for the webservices API
type WebService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebServiceSpec   `json:"spec,omitempty"`
	Status WebServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebServiceList contains a list of WebService
type WebServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WebService{}, &WebServiceList{})
}
