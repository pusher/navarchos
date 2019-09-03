package status

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/test/utils"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("NodeReplacement Status Suite", func() {
	var c client.Client
	var m utils.Matcher

	var nodeReplacement *navarchosv1alpha1.NodeReplacement
	var result *Result

	const timeout = time.Second * 5
	const consistentlyTimeout = time.Second

	BeforeEach(func() {
		var err error
		c, err = client.New(cfg, client.Options{})
		Expect(err).NotTo(HaveOccurred())
		m = utils.Matcher{Client: c}

		nodeReplacement = utils.ExampleNodeReplacement.DeepCopy()
		m.Create(nodeReplacement).Should(Succeed())

		result = &Result{}
	})

	AfterEach(func() {
		utils.DeleteAll(cfg, timeout,
			&navarchosv1alpha1.NodeReplacementList{},
		)
	})

	Context("UpdateStatus", func() {
		var updateErr error

		JustBeforeEach(func() {
			updateErr = UpdateStatus(c, nodeReplacement, result)
		})

		Context("when the phase is set in the Result", func() {
			var phase navarchosv1alpha1.NodeReplacementPhase

			BeforeEach(func() {
				phase = navarchosv1alpha1.ReplacementPhaseInProgress
				Expect(nodeReplacement.Status.Phase).ToNot(Equal(phase))
				result.Phase = &phase
			})

			It("updates the phase in the status", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("Phase", Equal(phase)))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when no existing NodePods is set", func() {
			var nodePods []string

			BeforeEach(func() {
				nodePods = []string{"example-pod-1", "example-pod-2", "example-pod-3", "example-pod-4"}
				Expect(nodeReplacement.Status.NodePods).To(BeEmpty())
				result.NodePods = nodePods
			})

			It("sets the NodePods field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("NodePods", Equal(nodePods)))
			})

			It("sets the NodePodsCount field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("NodePodsCount", Equal(len(nodePods))))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when an existing NodePods is set", func() {
			var nodePods []string
			var existingNodePods []string

			BeforeEach(func() {
				// Set up the existing expected state
				existingNodePods = []string{"example-pod-1", "example-pod-3"}
				m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
					nr.Status.NodePods = existingNodePods
					nr.Status.NodePodsCount = len(existingNodePods)
					return nr
				}, timeout).Should(Succeed())

				nodePods = []string{"example-pod-1", "example-pod-2", "example-pod-3", "example-pod-4"}
				result.NodePods = nodePods
			})

			It("does not update the NodePods field", func() {
				m.Consistently(nodeReplacement, consistentlyTimeout).Should(utils.WithNodeReplacementStatusField("NodePods", Equal(existingNodePods)))
			})

			It("does not update the NodePodsCount field", func() {
				m.Consistently(nodeReplacement, consistentlyTimeout).Should(utils.WithNodeReplacementStatusField("NodePodsCount", Equal(len(existingNodePods))))
			})

			It("returns an error", func() {
				Expect(updateErr.Error()).To(Equal("cannot update NodePods, field is immutable once set"))
			})
		})

		Context("when no existing EvictedPods is set", func() {
			var evictedPods []string

			BeforeEach(func() {
				evictedPods = []string{"example-pod-1", "example-pod-2", "example-pod-3", "example-pod-4"}
				Expect(nodeReplacement.Status.EvictedPods).To(BeEmpty())
				result.EvictedPods = evictedPods
			})

			It("sets the EvictedPods field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("EvictedPods", Equal(evictedPods)))
			})

			It("sets the EvictedPodsCount field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("EvictedPodsCount", Equal(len(evictedPods))))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when an existing EvictedPods is set", func() {
			var evictedPods []string
			var existingEvictedPods []string
			var expectedEvictedPods []string

			BeforeEach(func() {
				// Set up the existing expected state
				existingEvictedPods = []string{"example-pod-1", "example-pod-3"}
				m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
					nr.Status.EvictedPods = existingEvictedPods
					nr.Status.EvictedPodsCount = len(existingEvictedPods)
					return nr
				}, timeout).Should(Succeed())

				// Introduce some duplication, this implicitly tests for de-duplication.
				evictedPods = []string{"example-pod-2", "example-pod-4", "example-pod-1"}
				result.EvictedPods = evictedPods
				expectedEvictedPods = []string{"example-pod-2", "example-pod-4", "example-pod-1", "example-pod-3"}
			})

			It("joins the new and existing EvictedPods field", func() {
				m.Eventually(nodeReplacement, timeout).Should(
					utils.WithNodeReplacementStatusField("EvictedPods", ConsistOf(expectedEvictedPods)),
				)
			})

			It("updates the EvictedPodsCount field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("EvictedPodsCount", Equal(len(expectedEvictedPods))))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when no existing IgnoredPods is set", func() {
			var ignoredPods []navarchosv1alpha1.PodReason

			BeforeEach(func() {
				ignoredPods = []navarchosv1alpha1.PodReason{
					{Name: "example-pod-1", Reason: "reason-1"},
					{Name: "example-pod-2", Reason: "reason-2"},
					{Name: "example-pod-3", Reason: "reason-3"},
					{Name: "example-pod-4", Reason: "reason-4"},
				}
				Expect(nodeReplacement.Status.IgnoredPods).To(BeEmpty())
				result.IgnoredPods = ignoredPods
			})

			It("sets the IgnoredPods field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("IgnoredPods", Equal(ignoredPods)))
			})

			It("sets the IgnoredPodsCount field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("IgnoredPodsCount", Equal(len(ignoredPods))))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when an existing IgnoredPods is set", func() {
			var ignoredPods []navarchosv1alpha1.PodReason
			var existingIgnoredPods []navarchosv1alpha1.PodReason

			BeforeEach(func() {
				// Set up the existing expected state
				existingIgnoredPods = []navarchosv1alpha1.PodReason{
					{Name: "example-pod-1", Reason: "reason-1"},
					{Name: "example-pod-3", Reason: "reason-3"},
				}
				m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
					nr.Status.IgnoredPods = existingIgnoredPods
					nr.Status.IgnoredPodsCount = len(existingIgnoredPods)
					return nr
				}, timeout).Should(Succeed())

				ignoredPods = []navarchosv1alpha1.PodReason{
					{Name: "example-pod-1", Reason: "reason-1"},
					{Name: "example-pod-2", Reason: "reason-2"},
					{Name: "example-pod-3", Reason: "reason-3"},
					{Name: "example-pod-4", Reason: "reason-4"},
				}
				result.IgnoredPods = ignoredPods
			})

			It("does not update the IgnoredPods field", func() {
				m.Consistently(nodeReplacement, consistentlyTimeout).Should(utils.WithNodeReplacementStatusField("IgnoredPods", Equal(existingIgnoredPods)))
			})

			It("does not update the IgnoredPodsCount field", func() {
				m.Consistently(nodeReplacement, consistentlyTimeout).Should(utils.WithNodeReplacementStatusField("IgnoredPodsCount", Equal(len(existingIgnoredPods))))
			})

			It("returns an error", func() {
				Expect(updateErr.Error()).To(Equal("cannot update IgnoredPods, field is immutable once set"))
			})
		})

		Context("when no existing FailedPods is set", func() {
			var failedPods []navarchosv1alpha1.PodReason

			BeforeEach(func() {
				failedPods = []navarchosv1alpha1.PodReason{
					{Name: "example-pod-1", Reason: "reason-1"},
					{Name: "example-pod-2", Reason: "reason-2"},
					{Name: "example-pod-3", Reason: "reason-3"},
					{Name: "example-pod-4", Reason: "reason-4"},
				}
				Expect(nodeReplacement.Status.FailedPods).To(BeEmpty())
				result.FailedPods = failedPods
			})

			It("sets the FailedPods field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("FailedPods", Equal(failedPods)))
			})

			It("sets the FailedPodsCount field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("FailedPodsCount", Equal(len(failedPods))))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when an existing FailedPods is set", func() {
			var failedPods []navarchosv1alpha1.PodReason
			var existingFailedPods []navarchosv1alpha1.PodReason

			BeforeEach(func() {
				// Set up the existing expected state
				existingFailedPods = []navarchosv1alpha1.PodReason{
					{Name: "example-pod-1", Reason: "reason-1"},
					{Name: "example-pod-3", Reason: "reason-3"},
				}
				m.Update(nodeReplacement, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeReplacement)
					nr.Status.FailedPods = existingFailedPods
					nr.Status.FailedPodsCount = len(existingFailedPods)
					return nr
				}, timeout).Should(Succeed())

				failedPods = []navarchosv1alpha1.PodReason{
					{Name: "example-pod-2", Reason: "reason-2"},
					{Name: "example-pod-4", Reason: "reason-4"},
				}
				result.FailedPods = failedPods
			})

			It("updates the FailedPods field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("FailedPods", Equal(failedPods)))
			})

			It("updates the FailedPodsCount field", func() {
				m.Eventually(nodeReplacement, timeout).Should(utils.WithNodeReplacementStatusField("FailedPodsCount", Equal(len(failedPods))))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when the NodeCordonError is not set in the Result", func() {
			Context("and NodeCordonReason is set", func() {
				BeforeEach(func() {
					result.NodeCordonReason = "NodeCordoned"
				})

				It("adds the status condition with Status True", func() {
					m.Eventually(nodeReplacement, timeout).Should(
						utils.WithNodeReplacementStatusField("Conditions",
							ContainElement(SatisfyAll(
								utils.WithNodeReplacementConditionField("Type", Equal(navarchosv1alpha1.NodeCordonedType)),
								utils.WithNodeReplacementConditionField("Status", Equal(corev1.ConditionTrue)),
								utils.WithNodeReplacementConditionField("Reason", Equal(navarchosv1alpha1.NodeReplacementConditionReason("NodeCordoned"))),
								utils.WithNodeReplacementConditionField("Message", BeEmpty()),
							)),
						),
					)
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("and NodeCordonReason is not set", func() {
				It("should not add a status condition", func() {
					m.Eventually(nodeReplacement, timeout).Should(
						utils.WithNodeReplacementStatusField("Conditions",
							Not(ContainElement(
								utils.WithNodeReplacementConditionField("Type", Equal(navarchosv1alpha1.NodeCordonedType)),
							)),
						),
					)
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})
		})

		Context("when the NodeCordonError is set in the Result", func() {
			BeforeEach(func() {
				result.NodeCordonError = errors.New("error creating replacements")
				result.NodeCordonReason = "CompletedErrorReason"
			})

			It("updates the status condition", func() {
				m.Eventually(nodeReplacement, timeout).Should(
					utils.WithNodeReplacementStatusField("Conditions",
						ContainElement(SatisfyAll(
							utils.WithNodeReplacementConditionField("Type", Equal(navarchosv1alpha1.NodeCordonedType)),
							utils.WithNodeReplacementConditionField("Status", Equal(corev1.ConditionFalse)),
							utils.WithNodeReplacementConditionField("Reason", Equal(result.NodeCordonReason)),
							utils.WithNodeReplacementConditionField("Message", Equal(result.NodeCordonError.Error())),
						)),
					),
				)
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("NodeCordonError implies NodeCordonReason must be set", func() {
			Context("if only NodeCordonError is set", func() {
				BeforeEach(func() {
					result.NodeCordonError = errors.New("error")
				})

				It("causes an error", func() {
					Expect(updateErr).ToNot(BeNil())
				})
			})

			Context("if only NodeCordonReason is set", func() {
				BeforeEach(func() {
					result.NodeCordonReason = "test"
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("if both are set", func() {
				BeforeEach(func() {
					result.NodeCordonError = errors.New("error")
					result.NodeCordonReason = "test"
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("if neither are set", func() {
				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})
		})
	})
})
