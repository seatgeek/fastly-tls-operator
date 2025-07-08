package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FastlyCertificateSyncSpec defines the desired state of FastlyCertificateSync.
type FastlyCertificateSyncSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of FastlyCertificateSync. Edit fastlycertificatesync_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// FastlyCertificateSyncStatus defines the observed state of FastlyCertificateSync.
type FastlyCertificateSyncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FastlyCertificateSync is the Schema for the fastlycertificatesyncs API.
type FastlyCertificateSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FastlyCertificateSyncSpec   `json:"spec,omitempty"`
	Status FastlyCertificateSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FastlyCertificateSyncList contains a list of FastlyCertificateSync.
type FastlyCertificateSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FastlyCertificateSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FastlyCertificateSync{}, &FastlyCertificateSyncList{})
}
