package handler

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/test/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("in progress replacement handler", func() {
	var m utils.Matcher
	var h *NodeReplacementHandler
	var opts *Options

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

		grace := 5 * time.Second
		opts = &Options{
			EvictionGracePeriod: &grace,
		}

		// Create nodes to act as owners for the NodeReplacements created
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
		nodeReplacement.Spec.ReplacementSpec.Priority = intPtr(0)
		nodeReplacement.Spec.NodeUID = workerNode1.GetUID()
		nodeReplacement.Spec.NodeName = workerNode1.GetName()
		m.Create(nodeReplacement).Should(Succeed())
	})

	AfterEach(func() {
		close(stopMgr)
		mgrStopped.Wait()

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

	Context("addCompletedLabel", func() {
		var labelErr error
		JustBeforeEach(func() {
			m.Consistently(workerNode1, timeout).ShouldNot(utils.WithField("ObjectMeta.Labels", HaveKey("navarchos.pusher.com/drain-completed")))
			labelErr = h.addCompletedLabel(workerNode1.GetName())
		})

		It("adds a completed timestamp as a label", func() {
			m.Eventually(workerNode1, timeout).Should(utils.WithField("ObjectMeta.Labels", HaveKey("navarchos.pusher.com/drain-completed")))
		})

		It("does not set an error", func() {
			Expect(labelErr).ToNot(HaveOccurred())
		})
	})

	Context("parsePodName", func() {
		var podName string
		var parseErr error

		Context("when the error contains one double quoted substring", func() {
			JustBeforeEach(func() {
				podName, parseErr = parsePodName("error when evicting pod \"pod-1\" (will retry after 5s): Cannot evict pod as it would violate the pod's disruption budget.")
			})

			It("correctly parses the pod name", func() {
				Expect(podName).To(Equal("pod-1"))
			})

			It("does not set an error", func() {
				Expect(parseErr).ToNot(HaveOccurred())
			})
		})

		Context("when the error does not contain one double quoted substring", func() {
			JustBeforeEach(func() {
				podName, parseErr = parsePodName("error when evicting pod pod-1 (will retry after 5s): Cannot evict pod as it would violate the pod's disruption budget.")
			})

			It("does not return a parsed pod name", func() {
				Expect(podName).To(Equal(""))
			})

			It("does set an error", func() {
				Expect(parseErr.Error()).To(ContainSubstring("failed to parse error"))
			})
		})

	})

	Context("buildPodReasonsFromMap", func() {
		var podReasons []navarchosv1alpha1.PodReason
		var input map[string]string

		JustBeforeEach(func() {
			input = map[string]string{
				"pod-1": "pdb disruption",
				"pod-2": "global timeout",
			}
			podReasons = buildPodReasonsFromMap(input)
		})

		It("returns the input map formatted as PodReasons", func() {
			Expect(podReasons).To(ConsistOf([]navarchosv1alpha1.PodReason{
				{
					Name:   "pod-1",
					Reason: "pdb disruption",
				},
				{
					Name:   "pod-2",
					Reason: "global timeout",
				},
			}))
		})
	})

	Context("threadsafeEvictedPods", func() {
		var evictedPods threadsafeEvictedPods
		var podList []string

		Context("when handling concurrent writes", func() {
			JustBeforeEach(func() {
				var wg sync.WaitGroup

				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func(i int, wait *sync.WaitGroup) {
						evictedPods.writePod(fmt.Sprintf("pod_%d", i))
						wait.Done()
					}(i, &wg)
				}
				wg.Wait()

				podList = evictedPods.readPods()
			})

			It("does not drop a write", func() {
				testList := []string{}
				for i := 0; i < 10; i++ {
					testList = append(testList, fmt.Sprintf("pod_%d", i))
				}
				Expect(podList).To(ConsistOf(testList))
			})
		})
	})

	Context("threadsafeErrWriter", func() {
		var errWriter threadsafeErrWriter
		var errMap map[string]string

		Context("when handling concurrent writes with no evictedPods supplied", func() {
			JustBeforeEach(func() {
				var wg sync.WaitGroup

				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func(i int, wait *sync.WaitGroup) {
						str := fmt.Sprintf("unfortunately \"pod_%d\" could not be evicted, goodbye.\n", i)
						errWriter.Write([]byte(str))
						wait.Done()
					}(i, &wg)
				}
				wg.Wait()

				errMap = errWriter.ReadErrorMap([]string{})
			})

			It("does not drop a write", func() {
				testMap := map[string]string{}
				for i := 0; i < 10; i++ {
					testMap[fmt.Sprintf("pod_%d", i)] = fmt.Sprintf("unfortunately \"pod_%d\" could not be evicted, goodbye.\n", i)
				}
				Expect(errMap).To(Equal(testMap))
			})
		})

		Context("when handling concurrent writes with evictedPods supplied", func() {
			JustBeforeEach(func() {
				var wg sync.WaitGroup

				for i := 0; i < 10; i++ {
					wg.Add(1)
					go func(i int, wait *sync.WaitGroup) {
						str := fmt.Sprintf("unfortunately \"pod_%d\" could not be evicted, goodbye.\n", i)
						errWriter.Write([]byte(str))
						wait.Done()
					}(i, &wg)
				}
				wg.Wait()

				errMap = errWriter.ReadErrorMap([]string{"pod_0"})
			})

			It("does not drop a write", func() {
				testMap := map[string]string{}
				for i := 1; i < 10; i++ {
					testMap[fmt.Sprintf("pod_%d", i)] = fmt.Sprintf("unfortunately \"pod_%d\" could not be evicted, goodbye.\n", i)
				}
				Expect(errMap).To(Equal(testMap))
			})
		})
	})
})
