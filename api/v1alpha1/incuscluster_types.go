/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IncusClusterSpec defines the desired state of IncusCluster.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type IncusCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IncusClusterSpec   `json:"spec,omitempty"`
	Status IncusClusterStatus `json:"status,omitempty"`
}

type IncusClusterSpec struct {
	Network string `json:"network,omitempty"`
}

type IncusClusterStatus struct {
	// Conditions represent the latest available observations of the cluster's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
type IncusClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IncusCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IncusCluster{}, &IncusClusterList{})
}
