package handler

import (
	"context"
	"fmt"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/noderollout/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// handleCompleted handles a NodeRollout in the 'Completed' phase. It checks to
// see if the rollout is older than the cutoff defined as h.maxAge, if it is it
// deletes the rollout
func (h *NodeRolloutHandler) handleCompleted(instance *navarchosv1alpha1.NodeRollout) (*status.Result, error) {
	result := &status.Result{}
	cutoff := metav1.NewTime(metav1.Now().Add(-h.maxAge))

	if instance.Status.CompletionTimestamp != nil && instance.Status.CompletionTimestamp.Before(&cutoff) {
		err := h.client.Delete(context.Background(), instance)
		if err != nil {
			// todo: expose prometheus metric
			return nil, fmt.Errorf("error deleting resource: %v", err)
		}
	}

	return result, nil
}
