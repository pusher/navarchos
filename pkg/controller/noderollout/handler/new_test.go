package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("when handling new NodeRollouts", func() {
	var masterNode1 *corev1.Node
	var masterNode2 *corev1.Node
	var workerNode1 *corev1.Node
	var workerNode2 *corev1.Node

	var replacementSpec1 navarchosv1alpha1.ReplacementSpec
	var replacementSpec2 navarchosv1alpha1.ReplacementSpec

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
	})
	Context("filterNodeSelectors", func() {
		var selectors []navarchosv1alpha1.NodeLabelSelector
		var nodes *corev1.NodeList
		var filteredNodes map[string]nodeReplacementSpec
		var filterError error

		BeforeEach(func() {
			filteredNodes = make(map[string]nodeReplacementSpec)
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
		var filteredNodes map[string]nodeReplacementSpec

		BeforeEach(func() {
			filteredNodes = make(map[string]nodeReplacementSpec)
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
		var rollout *navarchosv1alpha1.NodeRollout
		var replacement1 navarchosv1alpha1.NodeReplacement
		var replacement2 navarchosv1alpha1.NodeReplacement

		BeforeEach(func() {
			rollout = utils.ExampleNodeRollout.DeepCopy()
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
})
