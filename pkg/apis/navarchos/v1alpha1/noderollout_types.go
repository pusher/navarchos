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
)

// NodeRolloutSpec defines the desired state of NodeRollout
type NodeRolloutSpec struct {
	// NodeSelectors uses label selectors to select a group of nodes.
	// The priority set on the label selector will be passed to the NodeReplacement.
	// The highest priority of any matching LabelSelector will be used,
	NodeSelectors []NodeLabelSelector `json:"nodeSelectors,omitempty"`

	// NodeNames allows specific nodes to be requested for replacement by name.
	// The priority set on the name will be passed to the NodeReplacement.
	// NodeName priorities always override NodeSelector priorities.
	NodeNames []NodeName `json:"nodeNames,omitempty"`
}

// NodeLabelSelector adds a ReplacementSpec field to the metav1.LabelSelector
type NodeLabelSelector struct {
	metav1.LabelSelector `json:",inline"`
	ReplacementSpec      ReplacementSpec `json:"replacement,omitempty"`
}

// NodeName pairs a Name with ReplacementSpec
type NodeName struct {
	Name            string          `json:"name"`
	ReplacementSpec ReplacementSpec `json:"replacement,omitempty"`
}

// NodeRolloutPhase determines the phase in which the NodeRollout currently is
type NodeRolloutPhase string

// The following RolloutPhases enumerate all possible NodeRolloutPhases
const (
	RolloutPhaseNew        NodeRolloutPhase = "New"
	RolloutPhaseInProgress NodeRolloutPhase = "InProgress"
	RolloutPhaseCompleted  NodeRolloutPhase = "Completed"
)

// NodeRolloutStatus defines the observed state of NodeRollout
type NodeRolloutStatus struct {
	// Phase is used to determine which phase of the replacement cycle a Rollout
	// is currently in.
	Phase NodeRolloutPhase `json:"phase"`

	// ReplacementsCreated lists the names of all NodeReplacements created by the
	// controller for this NodeRollout.
	ReplacementsCreated []string `json:"replacementsCreated,omitempty"`

	// ReplacementsCreatedCount is the count of ReplacementsCreated.
	// This is used for printing in kubectl.
	ReplacementsCreatedCount int `json:"replacementsCreatedCount,omitempty"`

	// ReplacementsCompleted lists the names of all NodeReplacements that have
	// successfully replaced their node.
	ReplacementsCompleted []string `json:"replacementsCompleted,omitempty"`

	// ReplacementsCompletedCount is the count of ReplacementsCompleted.
	// This is used for printing in kubectl.
	ReplacementsCompletedCount int `json:"replacementsCompletedCount,omitempty"`

	// CompletionTimestamp is a timestamp for when the rollout has completed
	CompletionTimestamp metav1.Time `json:"completionTimestamp,omitempty"`

	// Conditions gives detailed condition information about the NodeRollout
	Conditions []NodeRolloutCondition `json:"conditions,omitempty"`
}

// NodeRolloutConditionType is the type of a NodeRolloutCondition
type NodeRolloutConditionType string

const (
	// ReplacementsCreatedType refers to whether the controller successfully
	// created all of the required NodeRollouts
	ReplacementsCreatedType NodeRolloutConditionType = "ReplacementsCreated"

	// ReplacementsInProgressType refers to whether the controller is currently
	// processing replacements
	ReplacementsInProgressType NodeRolloutConditionType = "ReplacementsInProgress"
)

// NodeRolloutConditionReason represents a valid condition reason for a NodeRollout
type NodeRolloutConditionReason string

// NodeRolloutCondition is a status condition for a NodeRollout
type NodeRolloutCondition struct {
	// Type of this condition
	Type NodeRolloutConditionType `json:"type"`

	// Status of this condition
	Status corev1.ConditionStatus `json:"status"`

	// LastUpdateTime of this condition
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty"`

	// LastTransitionTime of this condition
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// Reason for the current status of this condition
	Reason NodeRolloutConditionReason `json:"reason,omitempty"`

	// Message associated with this condition
	Message string `json:"message,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// NodeRollout is the Schema for the noderollouts API
// +k8s:openapi-gen=true
type NodeRollout struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeRolloutSpec   `json:"spec,omitempty"`
	Status NodeRolloutStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// NodeRolloutList contains a list of NodeRollout
type NodeRolloutList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeRollout `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeRollout{}, &NodeRolloutList{})
}
