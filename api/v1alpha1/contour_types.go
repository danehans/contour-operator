// Copyright Project Contour Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// OwningContourLabel is the owner reference label used for a Contour
	// created by the operator.
	OwningContourLabel = "contour.operator.projectcontour.io/owning-contour"
)

// +kubebuilder:object:root=true

// Contour is the Schema for the contours API.
// +kubebuilder:subresource:status
type Contour struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ContourSpec   `json:"spec,omitempty"`
	Status ContourStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ContourList contains a list of Contour
type ContourList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Contour `json:"items"`
}

// ContourSpec defines the desired state of Contour.
type ContourSpec struct {
	// replicas is the desired number of Contour replicas. If unset,
	// defaults to 2.
	//
	// +kubebuilder:default=2
	// +kubebuilder:validation:Minimum=0
	Replicas int32 `json:"replicas,omitempty"`

	// namespace defines the schema of a Contour namespace.
	// See each field for additional details.
	//
	// +kubebuilder:default={name: "projectcontour", removeOnDeletion: false}
	Namespace NamespaceSpec `json:"namespace,omitempty"`
}

// NamespaceSpec defines the schema of a Contour namespace.
type NamespaceSpec struct {
	// name is the name of the namespace to run Contour and dependant
	// resources. If unset, defaults to "projectcontour".
	//
	// +kubebuilder:default=projectcontour
	Name string `json:"name,omitempty"`

	// removeOnDeletion will remove the namespace when the Contour is
	// deleted. If set to True, deletion will not occur if any of the
	// following conditions exist:
	//
	// 1. The Contour namespace is "default", "kube-system" or the
	//    contour-operator's namespace.
	//
	// 2. Another Contour exists in the namespace.
	//
	// +kubebuilder:default=false
	RemoveOnDeletion bool `json:"removeOnDeletion,omitempty"`
}

const (
	// Available indicates that the contour is running and available.
	ContourAvailableConditionType = "Available"

	// Progressing indicates that the contour is actively transitioning
	// from one steady state to another, e.g. propagating a configuration
	// change. Progressing should not be reported when reconciling a
	// previously known state.
	ContourProgressingConditionType = "Progressing"

	// Degraded indicates that the contour's current state does not
	// match its desired state over a period of time resulting in a
	// lower quality of service. Degraded state represents persistent
	// observation of a condition. As a result, a contour should not
	// oscillate in and out of the Degraded state. A contour may be
	// Available even if its Degraded. For example, a contour may
	// desire 2 running Contour pods, but 1 pod is crash-looping.
	// The contour is Available but Degraded because it may have a
	// lower quality of service. A contour may be Progressing but not
	// Degraded because the transition from one state to another does
	// not persist over a long enough period to report Degraded.
	// A contour should not report Degraded during the course of a normal
	// upgrade. A contour may report Degraded in response to a persistent
	// infrastructure failure that requires administrator intervention.
	// For example, if a cluster node is unhealthy and the cluster does
	// not contain enough nodes to run the desired number of Contour
	// replicas.
	ContourDegradedConditionType = "Degraded"

	// Unknown indicates that the contour's current state can not be
	// determined.
	ContourUnknownConditionType = "Unknown"
)

// ContourStatus defines the observed state of Contour.
type ContourStatus struct {
	// availableReplicas is the number of observed available Contour
	// replicas according to the deployment.
	AvailableReplicas int32 `json:"availableReplicas"`

	// availablePods is the number of observed available Envoy pods.
	AvailablePods int32 `json:"availablePods"`

	// availableSecrets is a list of available TLS secrets for the contour
	// represented as namespace/secret name.
	AvailableSecrets []string `json:"availableSecrets"`

	// Conditions represent the observations of a contour's current state.
	// Known condition types are "Available", "Progressing", and "Degraded".
	// Reference the condition types for additional details.
	//
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Contour{}, &ContourList{})
}
