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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type IncusMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IncusMachineSpec   `json:"spec,omitempty"`
	Status IncusMachineStatus `json:"status,omitempty"`
}

type IncusMachineSpec struct {
	// Node configuration for the VM
	Image     string `json:"image"`
	CPUs      int    `json:"cpus"`
	MemoryMiB int    `json:"memoryMiB"`
	// RootDiskSizeGiB is the size of the root disk in gibibytes. If 0, the default from the image/profile is used.
	// +optional
	RootDiskSizeGiB int `json:"rootDiskSizeGiB,omitempty"`
}

type IncusMachineStatus struct {
	// Conditions represent the latest available observations of the machine's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// InstanceID is the name of the Incus VM instance
	InstanceID string `json:"instanceId,omitempty"`
}

// +kubebuilder:object:root=true
type IncusMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IncusMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IncusMachine{}, &IncusMachineList{})
}
