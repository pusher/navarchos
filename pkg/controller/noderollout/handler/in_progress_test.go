package handler

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/test/utils"
)

var _ = Describe("when handling in progress NodeRollouts", func() {

	var replacement1 *navarchosv1alpha1.NodeReplacement
	var replacement2 *navarchosv1alpha1.NodeReplacement
	var replacement3 *navarchosv1alpha1.NodeReplacement

	BeforeEach(func() {
		replacement1 = utils.ExampleNodeReplacement.DeepCopy()
		replacement2 = utils.ExampleNodeReplacement.DeepCopy()
		replacement3 = utils.ExampleNodeReplacement.DeepCopy()

		replacement1.SetName("replacement1")
		replacement2.SetName("replacement2")
		replacement3.SetName("replacement3")

		replacement1.Spec.NodeName = "replacement1node"
		replacement2.Spec.NodeName = "replacement2node"
		replacement3.Spec.NodeName = "replacement3node"
	})

	Context("completedNodeReplacements", func() {
		var replacements []navarchosv1alpha1.NodeReplacement
		var output []string

		JustBeforeEach(func() {
			output = completedNodeReplacements(replacements)
		})

		Context("when no replacements are in the completed phase", func() {
			BeforeEach(func() {
				replacements = []navarchosv1alpha1.NodeReplacement{*replacement1, *replacement2, *replacement3}
			})

			It("returns an empty slice", func() {
				Expect(output).To(BeEmpty())
			})
		})

		Context("when only replacement1 is completed", func() {
			var updatedReplacement1 navarchosv1alpha1.NodeReplacement
			BeforeEach(func() {
				updatedReplacement1 = *replacement1
				updatedReplacement1.Status.Phase = navarchosv1alpha1.ReplacementPhaseCompleted
				replacements = []navarchosv1alpha1.NodeReplacement{updatedReplacement1, *replacement2, *replacement3}
			})

			It("returns replacement1's node only", func() {
				Expect(output).To(ConsistOf(replacement1.Spec.NodeName))
			})
		})

		Context("when all replacements are completed", func() {
			var updatedReplacement1 navarchosv1alpha1.NodeReplacement
			var updatedReplacement2 navarchosv1alpha1.NodeReplacement
			var updatedReplacement3 navarchosv1alpha1.NodeReplacement
			BeforeEach(func() {
				updatedReplacement1 = *replacement1
				updatedReplacement2 = *replacement2
				updatedReplacement3 = *replacement3

				updatedReplacement1.Status.Phase = navarchosv1alpha1.ReplacementPhaseCompleted
				updatedReplacement2.Status.Phase = navarchosv1alpha1.ReplacementPhaseCompleted
				updatedReplacement3.Status.Phase = navarchosv1alpha1.ReplacementPhaseCompleted
				replacements = []navarchosv1alpha1.NodeReplacement{updatedReplacement1, updatedReplacement2, updatedReplacement3}
			})

			It("returns the nodes from replacements 1,2 and 3", func() {
				Expect(output).To(ConsistOf(replacement1.Spec.NodeName, replacement2.Spec.NodeName, replacement3.Spec.NodeName))
			})
		})
	})
})
