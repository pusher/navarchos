package handler

import (
	"context"
	"fmt"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/noderollout/status"
)

// handleInProgress handles a NodeRollout in the 'InProgress' phase. It checks
// the number of completed NodeReplacements, and updates the result. If all
// replacements are completed it updates the status of the NodeReplacement to
// 'Complete'
func (h *NodeRolloutHandler) handleInProgress(instance *navarchosv1alpha1.NodeRollout) *status.Result {
	result := &status.Result{}

	nodeReplacementList := &navarchosv1alpha1.NodeReplacementList{}
	err := h.client.List(context.Background(), nodeReplacementList)
	if err != nil {
		result.ReplacementsCreatedError = fmt.Errorf("failed to list NodeReplacements: %v", err)
		return result
	}

	completed := completedNodeReplacements(nodeReplacementList.Items)
	result.ReplacementsCompleted = completed

	if len(completed) == len(nodeReplacementList.Items) {
		completedPhase := navarchosv1alpha1.RolloutPhaseCompleted
		result.Phase = &completedPhase
	}
	return result
}

// completedNodeReplacements takes a slice of replacements and returns a list of the nodes'
// names with the replacement phase set to completed
func completedNodeReplacements(replacements []navarchosv1alpha1.NodeReplacement) []string {
	var completedList = []string{}

	for _, replacement := range replacements {
		if replacement.Status.Phase == navarchosv1alpha1.ReplacementPhaseCompleted {
			completedList = append(completedList, replacement.Spec.NodeName)
		}
	}
	return completedList
}
