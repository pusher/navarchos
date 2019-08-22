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

var _ = Describe("NodeRollout Status Suite", func() {
	var c client.Client
	var m utils.Matcher

	var nodeRollout *navarchosv1alpha1.NodeRollout
	var result *Result

	const timeout = time.Second * 5
	const consistentlyTimeout = time.Second

	BeforeEach(func() {
		var err error
		c, err = client.New(cfg, client.Options{})
		Expect(err).NotTo(HaveOccurred())
		m = utils.Matcher{Client: c}

		nodeRollout = utils.ExampleNodeRollout.DeepCopy()
		m.Create(nodeRollout).Should(Succeed())

		result = &Result{}
	})

	AfterEach(func() {
		utils.DeleteAll(cfg, timeout,
			&navarchosv1alpha1.NodeRolloutList{},
		)
	})

	Context("UpdateStatus", func() {
		var updateErr error

		JustBeforeEach(func() {
			updateErr = UpdateStatus(c, nodeRollout, result)
		})

		Context("when the phase is set in the Result", func() {
			var phase navarchosv1alpha1.NodeRolloutPhase

			BeforeEach(func() {
				phase = navarchosv1alpha1.RolloutPhaseInProgress
				Expect(nodeRollout.Status.Phase).ToNot(Equal(phase))
				result.Phase = &phase
			})

			PIt("updates the phase in the status", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("Phase", Equal(phase)))
			})

			PIt("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when no existing ReplacementsCreated is set", func() {
			var replacementsCreated []string

			BeforeEach(func() {
				replacementsCreated = []string{"example-master-1", "example-master-2", "example-worker-1", "example-worker-2"}
				Expect(nodeRollout.Status.ReplacementsCreated).To(BeEmpty())
				result.ReplacementsCreated = replacementsCreated
			})

			PIt("sets the ReplacementsCreated field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCreated", Equal(replacementsCreated)))
			})

			PIt("sets the ReplacementsCreatedCount field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCreatedCount", Equal(len(replacementsCreated))))
			})

			PIt("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when an existing ReplacementsCreated is set", func() {
			var replacementsCreated []string
			var existingReplacementsCreated []string

			BeforeEach(func() {
				// Set up the existing expected state
				existingReplacementsCreated = []string{"example-master-1", "example-worker-1"}
				m.Update(nodeRollout, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeRollout)
					nr.Status.ReplacementsCreated = existingReplacementsCreated
					nr.Status.ReplacementsCreatedCount = len(existingReplacementsCreated)
					return nr
				}, timeout).Should(Succeed())

				replacementsCreated = []string{"example-master-1", "example-master-2", "example-worker-1", "example-worker-2"}
				result.ReplacementsCreated = replacementsCreated
			})

			It("does not update the ReplacementsCreated field", func() {
				m.Consistently(nodeRollout, consistentlyTimeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCreated", Equal(existingReplacementsCreated)))
			})

			It("does not update the ReplacementsCreatedCount field", func() {
				m.Consistently(nodeRollout, consistentlyTimeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCreatedCount", Equal(len(existingReplacementsCreated))))
			})

			PIt("returns an error", func() {
				Expect(updateErr.Error()).To(Equal("cannot update ReplacementsCreated, field is immutable once set"))
			})
		})

		Context("when no existing ReplacementsCompleted is set", func() {
			var replacementsCompleted []string

			BeforeEach(func() {
				replacementsCompleted = []string{"example-master-1", "example-master-2", "example-worker-1", "example-worker-2"}
				Expect(nodeRollout.Status.ReplacementsCompleted).To(BeEmpty())
				result.ReplacementsCompleted = replacementsCompleted
			})

			PIt("sets the ReplacementsCompleted field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCompleted", Equal(replacementsCompleted)))
			})

			PIt("sets the ReplacementsCompletedCount field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCompletedCount", Equal(len(replacementsCompleted))))
			})

			PIt("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when an existing ReplacementsCompleted is set", func() {
			var replacementsCompleted []string
			var existingReplacementsCompleted []string
			var expectedReplacementsCompleted []string

			BeforeEach(func() {
				// Set up the existing expected state
				existingReplacementsCompleted = []string{"example-master-1", "example-worker-1"}
				m.Update(nodeRollout, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeRollout)
					nr.Status.ReplacementsCompleted = existingReplacementsCompleted
					nr.Status.ReplacementsCompletedCount = len(existingReplacementsCompleted)
					return nr
				}, timeout).Should(Succeed())

				replacementsCompleted = []string{"example-master-2", "example-worker-2"}
				result.ReplacementsCompleted = replacementsCompleted

				expectedReplacementsCompleted = append(replacementsCompleted, existingReplacementsCompleted...)
			})

			PIt("joins the new and existing ReplacementsCompleted field", func() {
				m.Eventually(nodeRollout, timeout).Should(
					utils.WithNodeRolloutStatusField("ReplacementsCompleted", ConsistOf(expectedReplacementsCompleted)),
				)
			})

			PIt("updates the ReplacementsCompletedCount field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCompletedCount", Equal(len(expectedReplacementsCompleted))))
			})

			PIt("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when the ReplacementsCompletedError is not set in the Result", func() {
			PIt("updates the status condition", func() {
				m.Eventually(nodeRollout, timeout).Should(
					utils.WithNodeRolloutStatusField("Conditions",
						ContainElement(SatisfyAll(
							utils.WithNodeRolloutConditionField("Type", Equal(navarchosv1alpha1.ReplacementsCreatedType)),
							utils.WithNodeRolloutConditionField("Status", Equal(corev1.ConditionTrue)),
							utils.WithNodeRolloutConditionField("Reason", BeEmpty()),
							utils.WithNodeRolloutConditionField("Message", BeEmpty()),
						)),
					),
				)
			})

			PIt("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when the ReplacementsCompletedError is set in the Result", func() {
			BeforeEach(func() {
				result.ReplacementsCompletedError = errors.New("error creating replacements")
				result.ReplacementsCompletedReason = "CompletedErrorReason"
			})

			PIt("updates the status condition", func() {
				m.Eventually(nodeRollout, timeout).Should(
					utils.WithNodeRolloutStatusField("Conditions",
						ContainElement(SatisfyAll(
							utils.WithNodeRolloutConditionField("Type", Equal(navarchosv1alpha1.ReplacementsCreatedType)),
							utils.WithNodeRolloutConditionField("Status", Equal(corev1.ConditionFalse)),
							utils.WithNodeRolloutConditionField("Reason", Equal(result.ReplacementsCompletedReason)),
							utils.WithNodeRolloutConditionField("Message", Equal(result.ReplacementsCompletedError.Error())),
						)),
					),
				)
			})

			PIt("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("ReplacementsCompletedError and ReplacementsCompletedReason must be set together", func() {
			Context("if only ReplacementsCompleteError is set", func() {
				BeforeEach(func() {
					result.ReplacementsCompletedError = errors.New("error")
				})

				PIt("causes an error", func() {
					Expect(updateErr).ToNot(BeNil())
				})
			})

			Context("if only ReplacementsCompleteReason is set", func() {
				BeforeEach(func() {
					result.ReplacementsCompletedReason = "test"
				})

				PIt("causes an error", func() {
					Expect(updateErr).ToNot(BeNil())
				})
			})

			Context("if both are set", func() {
				BeforeEach(func() {
					result.ReplacementsCompletedError = errors.New("error")
					result.ReplacementsCompletedReason = "test"
				})

				PIt("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("if neither are set", func() {
				PIt("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})
		})
	})
})
