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

package handler

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	"github.com/pusher/navarchos/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Handler suite", func() {
	var m utils.Matcher
	var h *NodeReplacementHandler
	var result *status.Result

	var nodeReplacement *navarchosv1alpha1.NodeReplacement
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	var workerNode1 *corev1.Node
	var workerNode2 *corev1.Node
	var pod1 *corev1.Pod
	var pod2 *corev1.Pod
	var pod3 *corev1.Pod
	var pod4 *corev1.Pod

	const timeout = time.Second * 5
	const consistentlyTimeout = time.Second

	var newPod = func(name string, node *corev1.Node) *corev1.Pod {
		pod := utils.ExamplePod.DeepCopy()
		pod.Name = name
		pod.Spec.NodeName = node.Name
		return pod
	}

	var setPodRunning = func(obj utils.Object) utils.Object {
		pod, _ := obj.(*corev1.Pod)
		pod.Status.Phase = corev1.PodRunning
		return pod
	}

	var setPodSucceeded = func(obj utils.Object) utils.Object {
		pod, _ := obj.(*corev1.Pod)
		pod.Status.Phase = corev1.PodSucceeded
		return pod
	}

	BeforeEach(func() {
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c, err := client.New(cfg, client.Options{})
		Expect(err).ToNot(HaveOccurred())
		m = utils.Matcher{Client: c}

		stopMgr, mgrStopped = StartTestManager(mgr)

		// Create a node to act as owners for the NodeReplacements created
		workerNode1 = utils.ExampleNodeWorker1.DeepCopy()
		workerNode2 = utils.ExampleNodeWorker2.DeepCopy()
		m.Create(workerNode1).Should(Succeed())
		m.Create(workerNode2).Should(Succeed())

		// Create some pods attached to the nodes
		pod1 = newPod("pod-1", workerNode1)
		pod2 = newPod("pod-2", workerNode1)
		pod3 = newPod("pod-3", workerNode1)
		pod4 = newPod("pod-4", workerNode2)
		m.Create(pod1).Should(Succeed())
		m.Create(pod2).Should(Succeed())
		m.Create(pod3).Should(Succeed())
		m.Create(pod4).Should(Succeed())
		m.UpdateStatus(pod1, setPodRunning, timeout).Should(Succeed())
		m.UpdateStatus(pod2, setPodRunning, timeout).Should(Succeed())
		m.UpdateStatus(pod3, setPodRunning, timeout).Should(Succeed())
		m.UpdateStatus(pod4, setPodRunning, timeout).Should(Succeed())

		nodeReplacement = utils.ExampleNodeReplacement.DeepCopy()
		nodeReplacement.SetOwnerReferences([]metav1.OwnerReference{utils.GetOwnerReferenceForNode(workerNode1)})
		m.Create(nodeReplacement).Should(Succeed())
	})

	AfterEach(func() {
		close(stopMgr)
		mgrStopped.Wait()

		m.UpdateStatus(pod1, setPodSucceeded, timeout).Should(Succeed())
		m.UpdateStatus(pod2, setPodSucceeded, timeout).Should(Succeed())
		m.UpdateStatus(pod3, setPodSucceeded, timeout).Should(Succeed())
		m.UpdateStatus(pod4, setPodSucceeded, timeout).Should(Succeed())

		utils.DeleteAll(cfg, timeout,
			&navarchosv1alpha1.NodeReplacementList{},
			&corev1.NodeList{},
			&corev1.PodList{},
		)

		m.Eventually(&corev1.PodList{}, timeout).Should(utils.WithListItems(BeEmpty()))
	})

	Context("when the Handler function is called on a New NodeReplacement", func() {
		JustBeforeEach(func() {
			result = h.Handle(nodeReplacement)
		})
	})

	Context("when the Handler function is called on an InProgress NodeReplacement", func() {
		BeforeEach(func() {
			// Set the NodeReplacement as we expect it to be at this point
			m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
				nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
				nr.Status.Phase = navarchosv1alpha1.ReplacementPhaseInProgress
				nr.Status.NodePods = []string{}
				nr.Status.NodePodsCount = len(nr.Status.NodePods)
				return nr
			}, timeout).Should(Succeed())
			Expect(nodeReplacement.Status.Phase).To(Equal(navarchosv1alpha1.ReplacementPhaseInProgress))
		})

		JustBeforeEach(func() {
			result = h.Handle(nodeReplacement)
		})
	})

	Context("when the Handler function is called on a Completed NodeReplacement", func() {
		BeforeEach(func() {
			// Set the NodeReplacement as we expect it to be at this point
			m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
				nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
				nr.Status.Phase = navarchosv1alpha1.ReplacementPhaseCompleted
				nr.Status.NodePods = []string{}
				nr.Status.NodePodsCount = len(nr.Status.NodePods)
				nr.Status.EvictedPods = nr.Status.NodePods
				nr.Status.EvictedPodsCount = len(nr.Status.EvictedPods)
				return nr
			}, timeout).Should(Succeed())
			Expect(nodeReplacement.Status.Phase).To(Equal(navarchosv1alpha1.ReplacementPhaseCompleted))
		})

		JustBeforeEach(func() {
			result = h.Handle(nodeReplacement)
		})
	})

	Context("when the Handler function is called on a Failed NodeReplacement", func() {
		BeforeEach(func() {
			// Set the NodeReplacement as we expect it to be at this point
			m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
				nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
				nr.Status.Phase = navarchosv1alpha1.ReplacementPhaseFailed
				nr.Status.NodePods = []string{}
				nr.Status.NodePodsCount = len(nr.Status.NodePods)
				nr.Status.EvictedPods = []string{}
				nr.Status.EvictedPodsCount = len(nr.Status.EvictedPods)
				nr.Status.FailedPods = []navarchosv1alpha1.PodReason{}
				nr.Status.FailedPodsCount = len(nr.Status.FailedPods)
				return nr
			}, timeout).Should(Succeed())
			Expect(nodeReplacement.Status.Phase).To(Equal(navarchosv1alpha1.ReplacementPhaseFailed))
		})

		JustBeforeEach(func() {
			result = h.Handle(nodeReplacement)
		})
	})
})
