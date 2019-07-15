/*
Copyright 2018 Pusher Ltd.

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

package utils

import (
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetOwnerReferenceForNode constructs an owner reference for the Node given
func GetOwnerReferenceForNode(node *corev1.Node) metav1.OwnerReference {
	f := false
	return metav1.OwnerReference{
		APIVersion:         "v1",
		Kind:               "Node",
		Name:               node.Name,
		UID:                node.UID,
		Controller:         &f,
		BlockOwnerDeletion: &f,
	}
}

// GetOwnerReferenceForNodeRollout constructs an owner reference for the NodeRollout given
func GetOwnerReferenceForNodeRollout(nr *navarchosv1alpha1.NodeRollout) metav1.OwnerReference {
	t := true
	return metav1.OwnerReference{
		APIVersion:         "navarchos.pusher.com/v1alpha1",
		Kind:               "NodeRollout",
		Name:               nr.Name,
		UID:                nr.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}
}
