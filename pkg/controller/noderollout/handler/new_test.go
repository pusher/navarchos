package handler

import (
	"fmt"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("when handling new NodeRollouts", func() {
	var masterNode1 *corev1.Node
	var masterNode2 *corev1.Node
	var workerNode1 *corev1.Node
	var workerNode2 *corev1.Node

	var replacementSpec1 navarchosv1alpha1.ReplacementSpec
	var replacementSpec2 navarchosv1alpha1.ReplacementSpec

	var rollout *navarchosv1alpha1.NodeRollout

	var filteredNodes map[string]nodeReplacementSpec

	BeforeEach(func() {
		masterNode1 = utils.ExampleNodeMaster1.DeepCopy()
		masterNode2 = utils.ExampleNodeMaster2.DeepCopy()
		workerNode1 = utils.ExampleNodeWorker1.DeepCopy()
		workerNode2 = utils.ExampleNodeWorker2.DeepCopy()

		replacementSpec1 = navarchosv1alpha1.ReplacementSpec{
			Priority: intPtr(10),
		}

		replacementSpec2 = navarchosv1alpha1.ReplacementSpec{
			Priority: intPtr(20),
		}

		rollout = utils.ExampleNodeRollout.DeepCopy()

		filteredNodes = make(map[string]nodeReplacementSpec)

	})

	Context("filterNodeSelectors", func() {
		var selectors []navarchosv1alpha1.NodeLabelSelector
		var nodes *corev1.NodeList
		var filterError error

		BeforeEach(func() {
			nodes = &corev1.NodeList{
				Items: []corev1.Node{
					*masterNode1,
					*masterNode2,
					*workerNode1,
					*workerNode2,
				},
			}
		})

		JustBeforeEach(func() {
			filteredNodes, filterError = filterNodeSelectors(nodes, selectors, filteredNodes)
		})

		var AssertReturnsMatchingNodes = func() {
			It("only returns nodes that match the selectors", func() {
				Expect(filteredNodes).To(SatisfyAll(
					HaveKeyWithValue(
						masterNode1.GetName(),
						newNodeReplacementSpec(*masterNode1, replacementSpec1),
					),
					HaveKeyWithValue(
						masterNode2.GetName(),
						newNodeReplacementSpec(*masterNode2, replacementSpec1),
					),
					Not(HaveKey(workerNode1.GetName())),
					Not(HaveKey(workerNode2.GetName())),
				))

			})

			It("does not throw an error", func() {
				Expect(filterError).To(BeNil())
			})
		}

		Context("when using MatchLabels", func() {
			BeforeEach(func() {
				selectors = []navarchosv1alpha1.NodeLabelSelector{
					{
						LabelSelector: metav1.LabelSelector{
							MatchLabels: map[string]string{"node-role.kubernetes.io/master": "true"},
						},

						ReplacementSpec: replacementSpec1,
					},
				}
			})

			AssertReturnsMatchingNodes()
		})

		Context("when using MatchLabels", func() {
			BeforeEach(func() {
				selectors = []navarchosv1alpha1.NodeLabelSelector{
					{
						LabelSelector: metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "node-role.kubernetes.io/master",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{"true"},
								},
							},
						},

						ReplacementSpec: replacementSpec1,
					},
				}
			})

			AssertReturnsMatchingNodes()

		})
	})

	Context("filterNodeNames", func() {
		var nodeNames []navarchosv1alpha1.NodeName
		var nodes *corev1.NodeList

		BeforeEach(func() {
			nodes = &corev1.NodeList{
				Items: []corev1.Node{
					*masterNode1,
					*masterNode2,
					*workerNode1,
					*workerNode2,
				},
			}
		})

		JustBeforeEach(func() {
			filteredNodes = filterNodeNames(nodes, nodeNames, filteredNodes)
		})

		var AssertReturnsMatchingNodes = func() {
			It("only returns nodes that match the node names", func() {
				Expect(filteredNodes).To(SatisfyAll(
					HaveKeyWithValue(
						masterNode1.GetName(),
						newNodeReplacementSpec(*masterNode1, replacementSpec1),
					),
					HaveKeyWithValue(
						masterNode2.GetName(),
						newNodeReplacementSpec(*masterNode2, replacementSpec2),
					),
					Not(HaveKey(workerNode1.GetName())),
					Not(HaveKey(workerNode2.GetName())),
				))

			})
		}

		Context("when using MatchLabels", func() {
			BeforeEach(func() {
				nodeNames = []navarchosv1alpha1.NodeName{
					{
						Name:            masterNode1.GetName(),
						ReplacementSpec: replacementSpec1,
					},
					{
						Name:            masterNode2.GetName(),
						ReplacementSpec: replacementSpec2,
					},
				}
			})

			AssertReturnsMatchingNodes()
		})
	})

	Context("filterReplacementsByOwner", func() {
		var replacements []navarchosv1alpha1.NodeReplacement
		var nodeReplacementList *navarchosv1alpha1.NodeReplacementList
		var replacement1 navarchosv1alpha1.NodeReplacement
		var replacement2 navarchosv1alpha1.NodeReplacement

		BeforeEach(func() {
			replacement1 = *utils.ExampleNodeReplacement.DeepCopy()
			replacement2 = *utils.ExampleNodeReplacement.DeepCopy()

			replacement1.SetOwnerReferences(
				[]metav1.OwnerReference{utils.GetOwnerReferenceForNodeRollout(rollout)},
			)

			nodeReplacementList = &navarchosv1alpha1.NodeReplacementList{
				Items: []navarchosv1alpha1.NodeReplacement{
					replacement1,
					replacement2,
				},
			}
		})

		JustBeforeEach(func() {
			replacements = filterReplacementsByOwner(nodeReplacementList, rollout)
		})

		It("only returns replacements that are owned by the rollout", func() {
			Expect(replacements).To(SatisfyAll(
				ContainElement(replacement1),
				Not(ContainElement(replacement2)),
			))
		})

	})

	Context("createNodeReplacements", func() {
		const timeout = time.Second * 5
		const consistentlyTimeout = time.Second

		var m utils.Matcher
		var h *NodeRolloutHandler
		var opts *Options
		var mgrStopped *sync.WaitGroup
		var stopMgr chan struct{}

		var outputChan <-chan replacementCreationResult
		var createErr error

		BeforeEach(func() {
			mgr, err := manager.New(cfg, manager.Options{
				MetricsBindAddress: "0",
			})
			Expect(err).NotTo(HaveOccurred())

			c := mgr.GetClient()
			m = utils.Matcher{Client: c}
			opts = &Options{}

			stopMgr, mgrStopped = StartTestManager(mgr)

			h = NewNodeRolloutHandler(c, opts)
			m.Create(rollout).Should(Succeed())
			m.Get(rollout, timeout).Should(Succeed())

			m.Create(masterNode1).Should(Succeed())
			m.Get(masterNode1, timeout).Should(Succeed())
			m.Create(masterNode2).Should(Succeed())
			m.Get(masterNode2, timeout).Should(Succeed())
			m.Create(workerNode1).Should(Succeed())
			m.Get(workerNode1, timeout).Should(Succeed())
			m.Create(workerNode2).Should(Succeed())
			m.Get(workerNode2, timeout).Should(Succeed())

			filteredNodes = map[string]nodeReplacementSpec{
				masterNode1.GetName(): newNodeReplacementSpec(*masterNode1, replacementSpec1),
				workerNode1.GetName(): newNodeReplacementSpec(*workerNode1, replacementSpec2),
				masterNode2.GetName(): newNodeReplacementSpec(*masterNode2, replacementSpec1),
				workerNode2.GetName(): newNodeReplacementSpec(*workerNode2, replacementSpec2),
			}
		})

		AfterEach(func() {
			close(stopMgr)
			mgrStopped.Wait()

			utils.DeleteAll(cfg, timeout,
				&navarchosv1alpha1.NodeRolloutList{},
				&navarchosv1alpha1.NodeReplacementList{},
				&corev1.NodeList{},
			)
		})

		JustBeforeEach(func(done Done) {
			outputChan, createErr = h.createNodeReplacements(filteredNodes, rollout)
			close(done)
		}, 5)

		// checkForNodeReplacement checks if a NodeReplacement exists with the given
		// name, an owner reference pointing to the given node, and the given priority
		var checkForNodeReplacement = func(owner *corev1.Node, priority int) {
			nrList := &navarchosv1alpha1.NodeReplacementList{}

			m.Eventually(nrList, timeout).Should(utils.WithField("Items", ContainElement(SatisfyAll(
				utils.WithField("Spec.ReplacementSpec.Priority", Equal(&priority)),
				utils.WithField("Spec.NodeName", Equal(owner.GetName())),
				utils.WithField("Spec.NodeUID", Equal(owner.GetUID())),
				utils.WithField("ObjectMeta.OwnerReferences", SatisfyAll(
					ContainElement(Equal(utils.GetOwnerReferenceForNode(owner))),
					ContainElement(Equal(utils.GetOwnerReferenceForNodeRollout(rollout))),
				)),
			))))
		}

		Context("when no existing replacements exist", func() {
			It("creates a NodeReplacement for example-master-1", func() {
				checkForNodeReplacement(masterNode1, 10)
			})

			It("creates a NodeReplacement for example-master-2", func() {
				checkForNodeReplacement(masterNode2, 10)
			})

			It("creates a NodeReplacement for example-worker-1", func() {
				checkForNodeReplacement(workerNode1, 20)
			})

			It("creates a NodeReplacement for example-worker-2", func() {
				checkForNodeReplacement(workerNode2, 20)
			})

			It("should send a successful result for each node", func() {
				Expect(outputChan).To(HaveLen(4))
				output := []replacementCreationResult{}
				for i := 0; i < 4; i++ {
					output = append(output, <-outputChan)
				}
				Expect(output).To(ConsistOf(
					replacementCreationResult{replacementCreated: masterNode1.GetName()},
					replacementCreationResult{replacementCreated: masterNode2.GetName()},
					replacementCreationResult{replacementCreated: workerNode1.GetName()},
					replacementCreationResult{replacementCreated: workerNode2.GetName()},
				))
			})

			It("does not set any error", func() {
				Expect(createErr).To(BeNil())
			})
		})

		Context("when all of the replacements to be created already exist", func() {
			var nodeReplacementFor = func(node *corev1.Node) *navarchosv1alpha1.NodeReplacement {
				return &navarchosv1alpha1.NodeReplacement{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "navarchos.pusher.com/v1alpha1",
						Kind:       "NodeReplacement",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: fmt.Sprintf("%s-", node.Name),
						OwnerReferences: []metav1.OwnerReference{
							utils.GetOwnerReferenceForNode(node),
							utils.GetOwnerReferenceForNodeRollout(rollout),
						},
					},
					Spec: navarchosv1alpha1.NodeReplacementSpec{
						NodeName: node.GetName(),
						NodeUID:  node.GetUID(),
					},
				}
			}

			var masterReplacement1 *navarchosv1alpha1.NodeReplacement
			var masterReplacement2 *navarchosv1alpha1.NodeReplacement
			var workerReplacement1 *navarchosv1alpha1.NodeReplacement
			var workerReplacement2 *navarchosv1alpha1.NodeReplacement

			BeforeEach(func() {
				masterReplacement1 = nodeReplacementFor(masterNode1)
				m.Create(masterReplacement1).Should(Succeed())
				m.Get(masterReplacement1).Should(Succeed())
				masterReplacement2 = nodeReplacementFor(masterNode2)
				m.Create(masterReplacement2).Should(Succeed())
				m.Get(masterReplacement2).Should(Succeed())

				workerReplacement1 = nodeReplacementFor(workerNode1)
				m.Create(workerReplacement1).Should(Succeed())
				m.Get(workerReplacement1).Should(Succeed())
				workerReplacement2 = nodeReplacementFor(workerNode2)
				m.Create(workerReplacement2).Should(Succeed())
				m.Get(workerReplacement2).Should(Succeed())
			})

			It("does not create any new NodeReplacements", func() {
				replacementList := &navarchosv1alpha1.NodeReplacementList{}

				m.Consistently(replacementList, consistentlyTimeout).Should(utils.WithField("Items", ConsistOf(
					*masterReplacement1,
					*masterReplacement2,
					*workerReplacement1,
					*workerReplacement2,
				)))
			})

			It("should send a successful result for each node", func() {
				Expect(outputChan).To(HaveLen(4))
				output := []replacementCreationResult{}
				for i := 0; i < 4; i++ {
					output = append(output, <-outputChan)
				}
				Expect(output).To(ConsistOf(
					replacementCreationResult{replacementCreated: masterNode1.GetName()},
					replacementCreationResult{replacementCreated: masterNode2.GetName()},
					replacementCreationResult{replacementCreated: workerNode1.GetName()},
					replacementCreationResult{replacementCreated: workerNode2.GetName()},
				))
			})

			It("does not set any error", func() {
				Expect(createErr).To(BeNil())
			})
		})

		Context("when some of the replacements to be created already exist", func() {
			var nodeReplacementFor = func(node *corev1.Node) *navarchosv1alpha1.NodeReplacement {
				return &navarchosv1alpha1.NodeReplacement{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "navarchos.pusher.com/v1alpha1",
						Kind:       "NodeReplacement",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: fmt.Sprintf("%s-", node.Name),
						OwnerReferences: []metav1.OwnerReference{
							utils.GetOwnerReferenceForNode(node),
							utils.GetOwnerReferenceForNodeRollout(rollout),
						},
					},
					Spec: navarchosv1alpha1.NodeReplacementSpec{
						NodeName: node.GetName(),
						NodeUID:  node.GetUID(),
					},
				}
			}

			var masterReplacement1 *navarchosv1alpha1.NodeReplacement
			var masterReplacement2 *navarchosv1alpha1.NodeReplacement

			BeforeEach(func() {
				masterReplacement1 = nodeReplacementFor(masterNode1)
				m.Create(masterReplacement1).Should(Succeed())
				m.Get(masterReplacement1).Should(Succeed())
				masterReplacement2 = nodeReplacementFor(masterNode2)
				m.Create(masterReplacement2).Should(Succeed())
				m.Get(masterReplacement2).Should(Succeed())
			})

			It("does not create new NodeReplacements for the masters", func() {
				replacementList := &navarchosv1alpha1.NodeReplacementList{}

				m.Consistently(replacementList, consistentlyTimeout).Should(utils.WithField("Items", SatisfyAll(
					ContainElement(*masterReplacement1),
					ContainElement(*masterReplacement2),
				)))
			})

			It("creates a NodeReplacement for example-worker-1", func() {
				checkForNodeReplacement(workerNode1, 20)
			})

			It("creates a NodeReplacement for example-worker-2", func() {
				checkForNodeReplacement(workerNode2, 20)
			})

			It("should send a successful result for each node", func() {
				Expect(outputChan).To(HaveLen(4))
				output := []replacementCreationResult{}
				for i := 0; i < 4; i++ {
					output = append(output, <-outputChan)
				}
				Expect(output).To(ConsistOf(
					replacementCreationResult{replacementCreated: masterNode1.GetName()},
					replacementCreationResult{replacementCreated: masterNode2.GetName()},
					replacementCreationResult{replacementCreated: workerNode1.GetName()},
					replacementCreationResult{replacementCreated: workerNode2.GetName()},
				))
			})

			It("does not set any error", func() {
				Expect(createErr).To(BeNil())
			})
		})

		Context("when replacements exist with a different owner", func() {
			var newRollout *navarchosv1alpha1.NodeRollout

			var masterReplacement1 *navarchosv1alpha1.NodeReplacement
			var masterReplacement2 *navarchosv1alpha1.NodeReplacement
			var workerReplacement1 *navarchosv1alpha1.NodeReplacement
			var workerReplacement2 *navarchosv1alpha1.NodeReplacement

			var nodeReplacementFor = func(node *corev1.Node) *navarchosv1alpha1.NodeReplacement {
				return &navarchosv1alpha1.NodeReplacement{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "navarchos.pusher.com/v1alpha1",
						Kind:       "NodeReplacement",
					},
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: fmt.Sprintf("%s-", node.Name),
						OwnerReferences: []metav1.OwnerReference{
							utils.GetOwnerReferenceForNode(node),
							utils.GetOwnerReferenceForNodeRollout(newRollout),
						},
					},
					Spec: navarchosv1alpha1.NodeReplacementSpec{
						NodeName: node.GetName(),
						NodeUID:  node.GetUID(),
					},
				}
			}
			BeforeEach(func() {
				newRollout = utils.ExampleNodeRollout.DeepCopy()
				newRollout.ObjectMeta.SetName("example-2")

				m.Create(newRollout).Should(Succeed())
				m.Get(newRollout, timeout).Should(Succeed())

				masterReplacement1 = nodeReplacementFor(masterNode1)
				masterReplacement2 = nodeReplacementFor(masterNode2)
				workerReplacement1 = nodeReplacementFor(workerNode1)
				workerReplacement2 = nodeReplacementFor(workerNode2)

				m.Create(masterReplacement1).Should(Succeed())
				m.Get(masterReplacement1).Should(Succeed())
				m.Create(masterReplacement2).Should(Succeed())
				m.Get(masterReplacement2).Should(Succeed())
				m.Create(workerReplacement1).Should(Succeed())
				m.Get(workerReplacement1).Should(Succeed())
				m.Create(workerReplacement2).Should(Succeed())
				m.Get(workerReplacement2).Should(Succeed())

			})
			It("creates a NodeReplacement for example-master-1", func() {
				checkForNodeReplacement(masterNode1, 10)
			})

			It("creates a NodeReplacement for example-master-2", func() {
				checkForNodeReplacement(masterNode2, 10)
			})

			It("creates a NodeReplacement for example-worker-1", func() {
				checkForNodeReplacement(workerNode1, 20)
			})

			It("creates a NodeReplacement for example-worker-2", func() {
				checkForNodeReplacement(workerNode2, 20)
			})

			It("should send a successful result for each node", func() {
				Expect(outputChan).To(HaveLen(4))
				output := []replacementCreationResult{}
				for i := 0; i < 4; i++ {
					output = append(output, <-outputChan)
				}
				Expect(output).To(ConsistOf(
					replacementCreationResult{replacementCreated: masterNode1.GetName()},
					replacementCreationResult{replacementCreated: masterNode2.GetName()},
					replacementCreationResult{replacementCreated: workerNode1.GetName()},
					replacementCreationResult{replacementCreated: workerNode2.GetName()},
				))
			})

			It("does not set any error", func() {
				Expect(createErr).To(BeNil())
			})
		})

	})

})
