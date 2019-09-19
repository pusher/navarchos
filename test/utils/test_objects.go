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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func intPtr(i int) *int {
	return &i
}

var exampleApp = map[string]string{
	"app": "example",
}

// ExampleNodeRollout represents an example NodeRollout for use in tests
var ExampleNodeRollout = &navarchosv1alpha1.NodeRollout{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example",
	},
	Spec: navarchosv1alpha1.NodeRolloutSpec{
		NodeSelectors: []navarchosv1alpha1.NodeLabelSelector{
			{
				ReplacementSpec: navarchosv1alpha1.ReplacementSpec{
					Priority: intPtr(15),
				},
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/master": "true",
					},
				},
			},
			{
				ReplacementSpec: navarchosv1alpha1.ReplacementSpec{
					Priority: intPtr(5),
				},
				LabelSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"node-role.kubernetes.io/worker": "true",
					},
				},
			},
		},
		NodeNames: []navarchosv1alpha1.NodeName{
			{
				Name: "example-master-1",
				ReplacementSpec: navarchosv1alpha1.ReplacementSpec{
					Priority: intPtr(20),
				},
			},
			{
				Name: "example-worker-1",
				ReplacementSpec: navarchosv1alpha1.ReplacementSpec{
					Priority: intPtr(10),
				},
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
			"node-role.kubernetes.io/worker": "false",
		},
	},
}

// ExampleNodeMaster2 is an example Node for use in tests
var ExampleNodeMaster2 = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-master-2",
		Labels: map[string]string{
			"node-role.kubernetes.io/master": "true",
			"node-role.kubernetes.io/worker": "false",
		},
	},
}

// ExampleNodeWorker1 is an example Node for use in tests
var ExampleNodeWorker1 = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-worker-1",
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "true",
			"node-role.kubernetes.io/master": "false",
		},
	},
}

// ExampleNodeWorker2 is an example Node for use in tests
var ExampleNodeWorker2 = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-worker-2",
		Labels: map[string]string{
			"node-role.kubernetes.io/worker": "true",
			"node-role.kubernetes.io/master": "false",
		},
	},
}

// ExampleNodeOther is an example Node for use in tests
var ExampleNodeOther = &corev1.Node{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example-other",
		Labels: map[string]string{
			"node-role.kubernetes.io/other":  "true",
			"node-role.kubernetes.io/master": "false",
			"node-role.kubernetes.io/worker": "false",
		},
	},
}

// ExampleNodeReplacement represents an example NodeReplacement for use in tests
var ExampleNodeReplacement = &navarchosv1alpha1.NodeReplacement{
	ObjectMeta: metav1.ObjectMeta{
		Name: "example",
	},
	Status: navarchosv1alpha1.NodeReplacementStatus{
		Phase: navarchosv1alpha1.ReplacementPhaseNew,
	},
	Spec: navarchosv1alpha1.NodeReplacementSpec{
		ReplacementSpec: navarchosv1alpha1.ReplacementSpec{
			Priority: intPtr(0),
		},
	},
}

// ExamplePod is an example of a Pod for use in test
var ExamplePod = &corev1.Pod{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example-pod",
		Namespace: "default",
	},
	Spec: corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name:  "pause",
				Image: "k8s.gcr.io/pause",
			},
		},
	},
}

// ExampleDaemonSet is an example Daemonset for use in tests
var ExampleDaemonSet = &appsv1.DaemonSet{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example-daemonset",
		Namespace: "default",
		Labels:    exampleApp,
	},
	Spec: appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: exampleApp,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: exampleApp,
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "pause",
						Image: "k8s.gcr.io/pause",
					},
				},
			},
		},
	},
}

var intStr0 = intstr.FromInt(0)

// ExamplePodDisruptionBudget is an example PodDisruptionBudget for use in tests
var ExamplePodDisruptionBudget = policyv1beta1.PodDisruptionBudget{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "example-pdb",
		Namespace: "default",
	},
	Spec: policyv1beta1.PodDisruptionBudgetSpec{
		MaxUnavailable: &intStr0,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"block-eviction": "true",
			},
		},
	},
}
