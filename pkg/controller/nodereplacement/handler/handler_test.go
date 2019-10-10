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
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	"github.com/pusher/navarchos/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Handler suite", func() {
	var m utils.Matcher
	var h *NodeReplacementHandler
	var opts *Options
	var result *status.Result
	var handleErr error

	var nodeReplacement *navarchosv1alpha1.NodeReplacement
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}
	var stopPodGC chan struct{}

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
		pod.SetOwnerReferences([]metav1.OwnerReference{
			{
				APIVersion:         "apps/v1",
				Kind:               "ReplicaSet",
				Name:               "example",
				UID:                "qwertyuiop",
				Controller:         boolPtr(true),
				BlockOwnerDeletion: boolPtr(false),
			},
		})
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

	var startPodGC = func(m utils.Matcher) chan struct{} {
		close := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			for {
				select {
				case <-close:
					return
				default:
					podList := &corev1.PodList{}
					m.List(podList).Should(Succeed())
					for _, pod := range podList.Items {
						if pod.DeletionTimestamp != nil && pod.Status.Phase != corev1.PodSucceeded {
							m.UpdateStatus(&pod, setPodSucceeded, timeout).Should(Succeed())
							// Since we have no GC to check that the deletion requirements are met,
							// we will mock the GC here
							err := m.Client.Delete(context.Background(), &pod, client.GracePeriodSeconds(0))
							Expect(err).ToNot(HaveOccurred())
						}
					}
				}
			}
		}()
		return close
	}

	BeforeEach(func() {
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c, err := client.New(cfg, client.Options{})
		Expect(err).ToNot(HaveOccurred())
		m = utils.Matcher{Client: c}

		stopPodGC = startPodGC(m)
		stopMgr, mgrStopped = StartTestManager(mgr)

		grace := 1 * time.Second
		drain := 5 * time.Second
		opts = &Options{
			EvictionGracePeriod: &grace,
			DrainTimeout:        &drain,
			Config:              mgr.GetConfig(),
		}

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
		nodeReplacement.Spec.NodeUID = workerNode1.GetUID()
		nodeReplacement.Spec.NodeName = workerNode1.GetName()
		m.Create(nodeReplacement).Should(Succeed())
	})

	AfterEach(func() {
		close(stopMgr)
		mgrStopped.Wait()

		close(stopPodGC)

		pods := &corev1.PodList{}
		m.List(pods).Should(Succeed())
		for _, pod := range pods.Items {
			m.UpdateStatus(&pod, setPodSucceeded, timeout).Should(Succeed())
		}

		utils.DeleteAll(cfg, timeout,
			&navarchosv1alpha1.NodeReplacementList{},
			&corev1.NodeList{},
			&corev1.PodList{},
			&appsv1.DaemonSetList{},
			&policyv1beta1.PodDisruptionBudgetList{},
		)

		m.Eventually(&corev1.PodList{}, timeout).Should(utils.WithListItems(BeEmpty()))
	})

	JustBeforeEach(func() {
		h = NewNodeReplacementHandler(m.Client, opts)
	})

	Context("when the Handler is called on a New NodeReplacement", func() {
		JustBeforeEach(func() {
			result, handleErr = h.Handle(nodeReplacement)
		})

		Context("if a another NodeReplacement is higher priority", func() {
			BeforeEach(func() {
				highPriorityNR := utils.ExampleNodeReplacement.DeepCopy()
				highPriorityNR.SetName("high-priority")
				highPriorityNR.Spec.ReplacementSpec.Priority = intPtr(10)
				highPriorityNR.SetOwnerReferences([]metav1.OwnerReference{utils.GetOwnerReferenceForNode(workerNode2)})
				m.Create(highPriorityNR).Should(Succeed())
				m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
					nr.Spec.ReplacementSpec.Priority = intPtr(0)
					return nr
				}, timeout).Should(Succeed())
				m.Eventually(nodeReplacement, timeout).Should(utils.WithField("Spec.ReplacementSpec.Priority", Equal(intPtr(0))))
				Expect(*highPriorityNR.Spec.ReplacementSpec.Priority).To(BeNumerically(">", *nodeReplacement.Spec.ReplacementSpec.Priority))
			})

			It("requeues the NodeReplacement", func() {
				Expect(result.Requeue).To(BeTrue())
				Expect(result.RequeueReason).To(Equal("NodeReplacement \"high-priority\" has a higher priority"))
			})

			It("does not set the Result NodePods field", func() {
				Expect(result.NodePods).To(BeEmpty())
			})

			It("should not return an error", func() {
				Expect(handleErr).ToNot(HaveOccurred())
			})
		})

		Context("if a another NodeReplacement is the same priority", func() {
			BeforeEach(func() {
				samePriorityNR := utils.ExampleNodeReplacement.DeepCopy()
				samePriorityNR.SetName("same-priority")
				samePriorityNR.Spec.ReplacementSpec.Priority = intPtr(10)
				samePriorityNR.SetOwnerReferences([]metav1.OwnerReference{utils.GetOwnerReferenceForNode(workerNode2)})
				m.Create(samePriorityNR).Should(Succeed())
				m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
					nr.Spec.ReplacementSpec.Priority = intPtr(10)
					return nr
				}, timeout).Should(Succeed())
				m.Eventually(nodeReplacement, timeout).Should(utils.WithField("Spec.ReplacementSpec.Priority", Equal(intPtr(10))))
				Expect(*samePriorityNR.Spec.ReplacementSpec.Priority).To(BeNumerically("==", *nodeReplacement.Spec.ReplacementSpec.Priority))
			})

			It("does not requeue the NodeReplacement", func() {
				Expect(result.Requeue).To(BeFalse())
				Expect(result.RequeueReason).To(BeEmpty())
			})

			It("sets the Status NodePods field to contain a list of pods to evict", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithField("Status.NodePods",
					ConsistOf(
						"pod-1",
						"pod-2",
						"pod-3",
					),
				))
			})

			It("sets the Status Phase field to InProgress", func() {
				inProgress := navarchosv1alpha1.ReplacementPhaseInProgress
				m.Eventually(nodeReplacement, timeout).Should(utils.WithField("Status.Phase",
					Equal(inProgress),
				))
			})

			It("should not return an error", func() {
				Expect(handleErr).ToNot(HaveOccurred())
			})
		})

		Context("if a another NodeReplacement is in Phase InProgress", func() {
			BeforeEach(func() {
				highPriorityNR := utils.ExampleNodeReplacement.DeepCopy()
				highPriorityNR.SetName("in-progress")
				highPriorityNR.Status.Phase = navarchosv1alpha1.ReplacementPhaseInProgress
				highPriorityNR.SetOwnerReferences([]metav1.OwnerReference{utils.GetOwnerReferenceForNode(workerNode2)})
				m.Create(highPriorityNR).Should(Succeed())
			})

			It("requeues the NodeReplacement", func() {
				Expect(result.Requeue).To(BeTrue())
				Expect(result.RequeueReason).To(Equal("NodeReplacement \"in-progress\" is already in-progress"))
			})

			It("does not set the Result NodePods field", func() {
				Expect(result.NodePods).To(BeEmpty())
			})

			It("should not return an error", func() {
				Expect(handleErr).ToNot(HaveOccurred())
			})
		})

		It("should cordon the node", func() {
			m.Eventually(workerNode1, timeout).Should(utils.WithField("Spec.Unschedulable", BeTrue()))
			m.Eventually(workerNode1, timeout).Should(utils.WithField("Spec.Taints",
				ContainElement(SatisfyAll(
					utils.WithField("Effect", Equal(corev1.TaintEffect("NoSchedule"))),
					utils.WithField("Key", Equal("node.kubernetes.io/unschedulable")),
				)),
			))
		})

		It("should list all Pods in the Status NodePods field", func() {
			m.Eventually(nodeReplacement, timeout).Should(utils.WithField("Status.NodePods",
				ConsistOf(
					"pod-1",
					"pod-2",
					"pod-3",
				),
			))
		})

		Context("when a pod is owned by a DeamonSet", func() {
			BeforeEach(func() {
				ds := utils.ExampleDaemonSet.DeepCopy()
				m.Create(ds).Should(Succeed())
				m.Update(pod1, func(obj utils.Object) utils.Object {
					pod, _ := obj.(*corev1.Pod)
					pod.SetOwnerReferences([]metav1.OwnerReference{utils.GetOwnerReferenceForDaemonSet(ds)})
					return pod
				}, timeout).Should(Succeed())
			})

			It("should ignore the DaemonSet managed Pod", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithField("Status.IgnoredPods",
					ConsistOf(
						navarchosv1alpha1.PodReason{Name: "pod-1", Reason: "pod owned by a DaemonSet"},
					),
				))
			})

			It("should not return an error", func() {
				Expect(handleErr).ToNot(HaveOccurred())
			})
		})
	})

	Context("when the Handler is called on an InProgress NodeReplacement", func() {
		BeforeEach(func() {
			// Set the NodeReplacement as we expect it to be at this point
			m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
				nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
				nr.Status.Phase = navarchosv1alpha1.ReplacementPhaseInProgress
				nr.Status.NodePods = []string{"pod-1", "pod-2", "pod-3"}
				nr.Status.NodePodsCount = len(nr.Status.NodePods)
				return nr
			}, timeout).Should(Succeed())
			Expect(nodeReplacement.Status.Phase).To(Equal(navarchosv1alpha1.ReplacementPhaseInProgress))
		})

		// Since HandleInProgress could take some time, we set a timeout
		JustBeforeEach(func(done Done) {
			result, handleErr = h.Handle(nodeReplacement)
			close(done)
		}, 2*timeout.Seconds())

		It("evicts all pods in the NodePods list", func() {
			for _, pod := range []*corev1.Pod{pod1, pod2, pod3} {
				m.Get(pod, timeout).ShouldNot(Succeed())
			}
		})

		It("does not evict pods not listed in the NodePods list", func() {
			m.Consistently(pod4, consistentlyTimeout).Should(utils.WithField("ObjectMeta.DeletionTimestamp", BeNil()))
		})

		It("adds evicted pods to the Result EvictedPods field", func() {
			Expect(result.EvictedPods).To(ConsistOf("pod-1", "pod-2", "pod-3"))
		})

		It("does not add any pods to the Result FailedPods field", func() {
			Expect(result.FailedPods).To(BeEmpty())
		})

		PIt("deletes the node", func() {
			m.Get(workerNode1, timeout).ShouldNot(Succeed())
		})

		It("should not return an error", func() {
			Expect(handleErr).ToNot(HaveOccurred())
		})

		Context("if a Pod has already been evicted", func() {
			BeforeEach(func() {
				nodeReplacement.Status.NodePods = append(nodeReplacement.Status.NodePods, "evicted-pod")
				nodeReplacement.Status.EvictedPods = append(nodeReplacement.Status.EvictedPods, "evicted-pod")
			})

			It("does not list it in the Result EvictedPods field", func() {
				Expect(result.EvictedPods).ToNot(ContainElement(Equal("evicted-pod")))
			})

			It("does not list it in the Result FailedPods field", func() {
				Expect(result.FailedPods).To(BeEmpty())
			})

			It("should not return an error", func() {
				Expect(handleErr).ToNot(HaveOccurred())
			})
		})

		PContext("when a Pod Disruption Budget blocks eviction of a pod", func() {
			var pdb *policyv1beta1.PodDisruptionBudget
			BeforeEach(func() {
				pdb = utils.ExamplePodDisruptionBudget.DeepCopy()
				m.Create(pdb).Should(Succeed())
				m.Update(pod1, func(obj utils.Object) utils.Object {
					pod, _ := obj.(*corev1.Pod)
					// Ensure the Pod matches the PDB LabelSelector
					labels := pod.GetLabels()
					if labels == nil {
						labels = make(map[string]string)
					}
					for k, v := range pdb.Spec.Selector.MatchLabels {
						labels[k] = v
					}
					pod.SetLabels(labels)
					return pod
				}, timeout).Should(Succeed())
			})

			Context("permanently", func() {
				It("fails the eviction of the Pod", func() {
					Expect(result.FailedPods).To(ConsistOf(
						navarchosv1alpha1.PodReason{
							Name:   "pod-1",
							Reason: "evicting pod blocked by disruption budget",
						},
					))
				})

				It("does not delete the node", func() {
					m.Consistently(workerNode1).Should(utils.WithField("ObjectMeta.DeletionTimestamp", BeNil()))
				})

				It("should return an error", func() {
					Expect(handleErr).To(MatchError(Equal("failure evicting pods")))
				})
			})

			Context("temporarily", func() {
				BeforeEach(func() {
					// Ensure we update the PDB while the handler is running
					go func() {
						defer GinkgoRecover()
						time.Sleep(2 * time.Second)
						m.Update(pdb, func(obj utils.Object) utils.Object {
							p, _ := obj.(*policyv1beta1.PodDisruptionBudget)
							one := intstr.FromInt(1)
							p.Spec.MaxUnavailable = &one
							return p
						}, timeout-2*time.Second).Should(Succeed())
					}()
				})

				It("retries the eviction until it passes", func() {
					Expect(result.FailedPods).To(BeEmpty())
					Expect(result.EvictedPods).To(ContainElement(Equal("pod-1")))
				})

				It("should not return an error", func() {
					Expect(handleErr).ToNot(HaveOccurred())
				})
			})
		})
	})

	Context("when the Handler is called on a Completed NodeReplacement", func() {
		BeforeEach(func() {
			// Set the NodeReplacement as we expect it to be at this point
			m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
				nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
				nr.Status.Phase = navarchosv1alpha1.ReplacementPhaseCompleted
				return nr
			}, timeout).Should(Succeed())
			Expect(nodeReplacement.Status.Phase).To(Equal(navarchosv1alpha1.ReplacementPhaseCompleted))

		})

		It("should not return an error", func() {
			Expect(handleErr).ToNot(HaveOccurred())
		})

		It("should remain in the completed phase", func() {
			m.Consistently(nodeReplacement, consistentlyTimeout).Should(utils.WithField("Status.Phase", Equal(navarchosv1alpha1.ReplacementPhaseCompleted)))
		})

	})
})

func intPtr(i int) *int {
	return &i
}
