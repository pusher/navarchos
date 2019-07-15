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

package utils

import (
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func intPtr(i int) *int {
	return &i
}

// ExampleNodeRollout represents an example NodeRollout for use in tests
var ExampleNodeRollout = &navarchosv1alpha1.NodeRollout{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example",
	},
	Spec: navarchosv1alpha1.NodeRolloutSpec{
		NodeSelectors: []navarchosv1alpha1.PriorityLabelSelector{
			{
				Priority: intPtr(15),
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/master": "true",
					},
				},
			},
			{
				Priority: intPtr(5),
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/worker": "true",
					},
				},
			},
		},
		NodeNames: []navarchosv1alpha1.PriorityName{
			{
				Name:     "example-master-1",
				Priority: intPtr(20),
			},
			{
				Name:     "example-worker-1",
				Priority: intPtr(10),
			},
		},
	},
	Status: navarchosv1alpha1.NodeRolloutStatus{
		Phase: navarchosv1alpha1.RolloutPhaseNew,
	},
}

// ExampleNodeMaster1 is an example Node for use in tests
var ExampleNodeMaster1 = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-master-1",
		Labels: map[string]string{
			"node-role.kubernetes.io/master": "true",
		},
	},
}

// ExampleNodeMaster2 is an example Node for use in tests
var ExampleNodeMaster2 = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-master-2",
		Labels: map[string]string{
			"node-role.kubernetes.io/master": "true",
		},
	},
}

// ExampleNodeWorker1 is an example Node for use in tests
var ExampleNodeWorker1 = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-worker-1",
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "true",
		},
	},
}

// ExampleNodeWorker2 is an example Node for use in tests
var ExampleNodeWorker2 = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-worker-2",
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "true",
		},
	},
}

// ExampleNodeReplacement represents an example NodeReplacement for use in tests
var ExampleNodeReplacement = &navarchosv1alpha1.NodeReplacement{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example",
	},
}
