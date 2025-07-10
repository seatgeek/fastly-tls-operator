/*
Copyright 2025 SeatGeek.
*/

package v1alpha1

import (
	"github.com/seatgeek/k8s-reconciler-generic/apiobjects"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FastlyCertificateSyncSpec defines the desired state of FastlyCertificateSync.
type FastlyCertificateSyncSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Reconciliation of individual resources may be suspended by setting this flag.
	Suspend bool `json:"suspend,omitempty" yaml:"suspend,omitempty"`

	// The name of the Certificate resource to sync
	CertificateName string `json:"certificateName,omitempty" yaml:"certificateName,omitempty"`

	// The list of TLS configuration IDs to sync
	TLSConfigurationIds []string `json:"tlsConfigurationIds,omitempty" yaml:"tlsConfigurationIds,omitempty"`
}

// FastlyCertificateSyncStatus defines the observed state of FastlyCertificateSync.
type FastlyCertificateSyncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	apiobjects.SubjectStatus `json:",inline" yaml:",inline"`

	Ready      bool               `json:"ready" yaml:"ready"`
	Conditions []metav1.Condition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.ready"
// +kubebuilder:printcolumn:name="Suspended",type="boolean",JSONPath=".spec.suspend"

// FastlyCertificateSync is the Schema for the fastlycertificatesyncs API.
type FastlyCertificateSync struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   FastlyCertificateSyncSpec   `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status FastlyCertificateSyncStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FastlyCertificateSyncList contains a list of FastlyCertificateSync.
type FastlyCertificateSyncList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Items           []FastlyCertificateSync `json:"items" yaml:"items"`
}

func (in *FastlyCertificateSync) IsSuspended() bool {
	return in.Spec.Suspend
}

func init() {
	SchemeBuilder.Register(&FastlyCertificateSync{}, &FastlyCertificateSyncList{})
}
