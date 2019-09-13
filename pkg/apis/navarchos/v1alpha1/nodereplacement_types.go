/*
Copyright 2019 Pusher Ltd.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// NodeReplacementSpec defines the desired state of NodeReplacement
type NodeReplacementSpec struct {
	ReplacementSpec ReplacementSpec `json:"replacement,omitempty"`

	// NodeName should match the Name of the Node this NodeReplacement intends to
	// replace.
	NodeName string `json:"nodeName,omitempty"`

	// NodeUID should match the UID of the Node this NodeReplacement intends to
	// replace.
	NodeUID types.UID `json:"nodeUID,omitempty"`
}

// ReplacementSpec contains configuration for the replacement of the Node
// reference in the NodeReplacement
type ReplacementSpec struct {
	// Priority determines the priority of this NodeReplacement.
	// Higher priorities should be replaced sooner.
	Priority *int `json:"priority,omitempty"`
}

// NodeReplacementPhase determines the phase in which the NodeRollout currently is
type NodeReplacementPhase string

// The following ReplacementPhases enumerate all possible NodeReplacementPhases
const (
	ReplacementPhaseNew        NodeReplacementPhase = "New"
	ReplacementPhaseInProgress NodeReplacementPhase = "InProgress"
	ReplacementPhaseCompleted  NodeReplacementPhase = "Completed"
)

// NodeReplacementStatus defines the observed state of NodeReplacement
type NodeReplacementStatus struct {
	// Phase is used to determine which phase of the replacement cycle a Replacement
	// is currently in.
	Phase NodeReplacementPhase `json:"phase"`

	// NodePods lists all pods on the node when the  controller cordoned it.
	NodePods []string `json:"nodePods,omitempty"`

	// NodePodsCount is the count of NodePods.
	NodePodsCount int `json:"nodePodsCount,omitempty"`

	// EvictedPods lists all pods successfully evicted by the controller.
	EvictedPods []string `json:"evictedPods,omitempty"`

	// EvictedPodsCount is the count of EvictedPods
	EvictedPodsCount int `json:"evictedPodsCount,omitempty"`

	// IgnoredPods lists all pods not being evicted by the controller.
	// This should contain daemonset pods at the minimum.
	IgnoredPods []PodReason `json:"ignoredPods,omitempty"`

	// IgnoredPodsCount is the count of IgnoredPods.
	IgnoredPodsCount int `json:"ignoredPodsCount,omitempty"`

	// FailedPods lists all pods the controller has failed to evict.
	FailedPods []PodReason `json:"failedPods,omitempty"`

	// FailedPodsCount is the count of FailedPods.
	FailedPodsCount int `json:"failedPodsCount,omitempty"`

	// CompletionTimestamp is a timestamp for when the replacement has completed
	CompletionTimestamp *metav1.Time `json:"completionTimestamp,omitempty"`

	// Conditions gives detailed condition information about the NodeReplacement
	Conditions []NodeReplacementCondition `json:"conditions,omitempty"`
}

// PodReason is used to add details to a Pods eviction status
type PodReason struct {
	// Name is the name of the pod
	Name string `json:"name"`

	// Reason is the message to display to the user as to why this Pod is ignored/failed
	Reason string `json:"reason"`
}

// NodeReplacementConditionType is the type of a NodeRolloutCondition
type NodeReplacementConditionType string

const (
	// NodeCordonedType refers to whether the controller successfully managed to
	// cordon the node.
	NodeCordonedType NodeReplacementConditionType = "NodeCordoned"
)

// NodeReplacementConditionReason represents a valid condition reason for a NodeReplacement
type NodeReplacementConditionReason string

// NodeReplacementCondition is a status condition for a NodeReplacement
type NodeReplacementCondition struct {
	// Type of this condition
	Type NodeReplacementConditionType `json:"type"`

	// Status of this condition
	Status corev1.ConditionStatus `json:"status"`

	// LastUpdateTime of this condition
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastTransitionTime of this condition
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason for the current status of this condition
	Reason NodeReplacementConditionReason `json:"reason,omitempty"`

	// Message associated with this condition
	Message string `json:"message,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// NodeReplacement is the Schema for the nodereplacements API
// +k8s:openapi-gen=true
type NodeReplacement struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeReplacementSpec   `json:"spec,omitempty"`
	Status NodeReplacementStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// NodeReplacementList contains a list of NodeReplacement
type NodeReplacementList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeReplacement `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeReplacement{}, &NodeReplacementList{})
}
