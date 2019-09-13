package status

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

			It("updates the phase in the status", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("Phase", Equal(phase)))
			})

			It("does not cause an error", func() {
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

			It("sets the ReplacementsCreated field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCreated", Equal(replacementsCreated)))
			})

			It("sets the ReplacementsCreatedCount field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCreatedCount", Equal(len(replacementsCreated))))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when an existing ReplacementsCreated is set and ReplacementsCreated is set in  the Result", func() {
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

			It("returns an error", func() {
				Expect(updateErr).ToNot(BeNil())
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

			It("sets the ReplacementsCompleted field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCompleted", Equal(replacementsCompleted)))
			})

			It("sets the ReplacementsCompletedCount field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCompletedCount", Equal(len(replacementsCompleted))))
			})

			It("does not cause an error", func() {
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

				// Introduce some duplication, this implicitly tests for de-duplication.
				replacementsCompleted = []string{"example-master-2", "example-worker-2", "example-master-1"}
				result.ReplacementsCompleted = replacementsCompleted
				expectedReplacementsCompleted = []string{"example-master-2", "example-worker-2", "example-master-1", "example-worker-1"}

			})

			It("joins the new and existing ReplacementsCompleted field", func() {
				m.Eventually(nodeRollout, timeout).Should(
					utils.WithNodeRolloutStatusField("ReplacementsCompleted", ConsistOf(expectedReplacementsCompleted)),
				)
			})

			It("updates the ReplacementsCompletedCount field", func() {
				m.Eventually(nodeRollout, timeout).Should(utils.WithNodeRolloutStatusField("ReplacementsCompletedCount", Equal(len(expectedReplacementsCompleted))))
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("when no existing CompletionTimestamp is set", func() {
			var completionTimestamp metav1.Time

			Context("and there is a CompletionTimestamp set in the Result", func() {
				BeforeEach(func() {
					completionTimestamp = metav1.Now()
					Expect(nodeRollout.Status.CompletionTimestamp).To(BeNil())
					result.CompletionTimestamp = &completionTimestamp
				})

				It("sets the CompletionTimestamp field", func() {
					m.Eventually(nodeRollout, timeout).Should(utils.WithField("Status.CompletionTimestamp", Equal(&completionTimestamp)))
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("and there is not a CompletionTimestamp set in the Result", func() {
				BeforeEach(func() {
					Expect(nodeRollout.Status.CompletionTimestamp).To(BeNil())
				})

				It("does not set the CompletionTimestamp", func() {
					m.Consistently(nodeRollout, consistentlyTimeout).Should(utils.WithField("Status.CompletionTimestamp", BeNil()))
				})
			})

		})

		Context("when an existing CompletionTimestamp is set and CompletionTimestamp is set in  the Result", func() {
			var completionTimestamp metav1.Time
			var existingCompletionTimestamp metav1.Time

			BeforeEach(func() {
				// Set up the existing expected state
				existingCompletionTimestamp = metav1.NewTime(metav1.Now().Add(-time.Hour))
				m.Update(nodeRollout, func(obj utils.Object) utils.Object {
					nr, _ := obj.(*navarchosv1alpha1.NodeRollout)
					nr.Status.CompletionTimestamp = &existingCompletionTimestamp
					return nr
				}, timeout).Should(Succeed())

				completionTimestamp = metav1.Now()
				result.CompletionTimestamp = &completionTimestamp
			})

			It("does not update the CompletionTimestamp field", func() {
				m.Consistently(nodeRollout, consistentlyTimeout).Should(utils.WithField("Status.CompletionTimestamp", Equal(&existingCompletionTimestamp)))
			})

			It("returns an error", func() {
				Expect(updateErr).ToNot(BeNil())
				Expect(updateErr.Error()).To(Equal("cannot update CompletionTimestamp, field is immutable once set"))
			})
		})

		Context("when the ReplacementsCreatedError is not set in the Result", func() {
			Context("and ReplacementsCreatedReason is set", func() {
				BeforeEach(func() {
					result.ReplacementsCreatedReason = "ErrorCreatingNodeReplacements"

				})

				It("adds the status condition with Status True", func() {
					m.Eventually(nodeRollout, timeout).Should(
						utils.WithField("Status.Conditions",
							ContainElement(SatisfyAll(
								utils.WithField("Type", Equal(navarchosv1alpha1.ReplacementsCreatedType)),
								utils.WithField("Status", Equal(corev1.ConditionTrue)),
								utils.WithField("Reason", Equal(navarchosv1alpha1.NodeRolloutConditionReason("ErrorCreatingNodeReplacements"))),
								utils.WithField("Message", BeEmpty()),
							)),
						),
					)
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("and ReplacementsCreatedReason is not set", func() {
				It("should not add a status condition", func() {
					m.Consistently(nodeRollout, consistentlyTimeout).Should(
						utils.WithField("Status.Conditions",
							Not(ContainElement(
								utils.WithField("Type", Equal(navarchosv1alpha1.ReplacementsCreatedType)),
							)),
						),
					)
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})
		})

		Context("when the ReplacementsCreatedError is set in the Result", func() {
			BeforeEach(func() {
				result.ReplacementsCreatedError = errors.New("error creating replacements")
				result.ReplacementsCreatedReason = "CreatedErrorReason"
			})

			It("updates the status condition", func() {
				m.Eventually(nodeRollout, timeout).Should(
					utils.WithNodeRolloutStatusField("Conditions",
						ContainElement(SatisfyAll(
							utils.WithNodeRolloutConditionField("Type", Equal(navarchosv1alpha1.ReplacementsCreatedType)),
							utils.WithNodeRolloutConditionField("Status", Equal(corev1.ConditionFalse)),
							utils.WithNodeRolloutConditionField("Reason", Equal(result.ReplacementsCreatedReason)),
							utils.WithNodeRolloutConditionField("Message", Equal(result.ReplacementsCreatedError.Error())),
						)),
					),
				)
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("ReplacementsCreatedError implies ReplacementsCreatedReason must be set", func() {
			Context("if only ReplacementsCompleteError is set", func() {
				BeforeEach(func() {
					result.ReplacementsCreatedError = errors.New("error")
				})

				It("causes an error", func() {
					Expect(updateErr).ToNot(BeNil())
				})
			})

			Context("if only ReplacementsCreatedReason is set", func() {
				BeforeEach(func() {
					result.ReplacementsCreatedReason = "test"
				})

				It("doess not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("if both are set", func() {
				BeforeEach(func() {
					result.ReplacementsCreatedError = errors.New("error")
					result.ReplacementsCreatedReason = "test"
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

		Context("when the ReplacementsInProgressError is not set in the Result", func() {
			Context("and ReplacementsInProgressReason is set", func() {
				BeforeEach(func() {
					result.ReplacementsInProgressReason = "ErrorListingNodes"
				})

				It("adds the status condition with Status True", func() {
					m.Eventually(nodeRollout, timeout).Should(
						utils.WithField("Status.Conditions",
							ContainElement(SatisfyAll(
								utils.WithField("Type", Equal(navarchosv1alpha1.ReplacementsInProgressType)),
								utils.WithField("Status", Equal(corev1.ConditionTrue)),
								utils.WithField("Reason", Equal(navarchosv1alpha1.NodeRolloutConditionReason("ErrorListingNodes"))),
								utils.WithField("Message", BeEmpty()),
							)),
						),
					)
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("and ReplacementsInProgressReason is not set", func() {
				It("should not add a status condition", func() {
					m.Consistently(nodeRollout, consistentlyTimeout).Should(
						utils.WithField("Status.Conditions",
							Not(ContainElement(
								utils.WithField("Type", Equal(navarchosv1alpha1.ReplacementsInProgressType)),
							)),
						),
					)
				})

				It("does not cause an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})
		})

		Context("when the ReplacementsInProgressError is set in the Result", func() {
			BeforeEach(func() {
				result.ReplacementsInProgressError = errors.New("error in progress replacements")
				result.ReplacementsInProgressReason = "InProgressErrorReason"
			})

			It("updates the status condition", func() {
				m.Eventually(nodeRollout, timeout).Should(
					utils.WithNodeRolloutStatusField("Conditions",
						ContainElement(SatisfyAll(
							utils.WithNodeRolloutConditionField("Type", Equal(navarchosv1alpha1.ReplacementsInProgressType)),
							utils.WithNodeRolloutConditionField("Status", Equal(corev1.ConditionFalse)),
							utils.WithNodeRolloutConditionField("Reason", Equal(result.ReplacementsInProgressReason)),
							utils.WithNodeRolloutConditionField("Message", Equal(result.ReplacementsInProgressError.Error())),
						)),
					),
				)
			})

			It("does not cause an error", func() {
				Expect(updateErr).To(BeNil())
			})
		})

		Context("ReplacementsInProgressError implies ReplacementsInProgressReason must be set", func() {
			Context("if only ReplacementsInProgressError is set", func() {
				BeforeEach(func() {
					result.ReplacementsInProgressError = errors.New("error")
				})

				It("causes an error", func() {
					Expect(updateErr).ToNot(BeNil())
				})
			})

			Context("if only ReplacementsInProgressReason is set", func() {
				BeforeEach(func() {
					result.ReplacementsInProgressReason = "test"
				})

				It("does not causes an error", func() {
					Expect(updateErr).To(BeNil())
				})
			})

			Context("if both are set", func() {
				BeforeEach(func() {
					result.ReplacementsInProgressError = errors.New("error")
					result.ReplacementsInProgressReason = "test"
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
