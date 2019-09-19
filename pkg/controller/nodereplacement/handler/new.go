package handler

import (
	"context"
	"fmt"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/util/taints"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// handleNew handles a NodeReplacement in the New phase
func (h *NodeReplacementHandler) handleNew(instance *navarchosv1alpha1.NodeReplacement) (*status.Result, error) {
	result := &status.Result{}
	replacements := &navarchosv1alpha1.NodeReplacementList{}
	err := h.client.List(context.Background(), replacements)
	if err != nil {
		return &status.Result{}, err
	}

	for _, replacement := range replacements.Items {
		if *replacement.Spec.ReplacementSpec.Priority > *instance.Spec.ReplacementSpec.Priority {
			result.Requeue = true
			result.RequeueReason = fmt.Sprintf("NodeReplacement \"%s\" has a higher priority", replacement.GetName())
			return result, nil
		}
		if replacement.Status.Phase == navarchosv1alpha1.ReplacementPhaseInProgress {
			result.Requeue = true
			result.RequeueReason = fmt.Sprintf("NodeReplacement \"%s\" is already in-progress", replacement.GetName())
			return result, nil
		}
	}

	node := &corev1.Node{}

	err = h.client.Get(context.Background(), client.ObjectKey{
		Name: instance.Spec.NodeName,
	}, node)
	if err != nil {
		if errors.IsNotFound(err) {
			// node no longer exists, mark as completed
			completedPhase := navarchosv1alpha1.ReplacementPhaseCompleted
			completedTime := metav1.Now()

			result.Phase = &completedPhase
			result.CompletionTimestamp = &completedTime

			return result, nil
		}

		return &status.Result{}, err
	}

	if node.GetUID() != instance.Spec.NodeUID {
		// node no longer exists, mark as completed
		completedPhase := navarchosv1alpha1.ReplacementPhaseCompleted
		completedTime := metav1.Now()

		result.Phase = &completedPhase
		result.CompletionTimestamp = &completedTime

		return result, nil
	}

	// cordon the node
	node.Spec.Unschedulable = true
	node, _, err = taints.AddOrUpdateTaint(node, &corev1.Taint{
		Key:    "node.kubernetes.io/unschedulable",
		Effect: corev1.TaintEffect("NoSchedule"),
	})
	err = h.client.Update(context.Background(), node)
	if err != nil {
		return &status.Result{}, err
	}

	podList := &corev1.PodList{}
	err = h.client.List(context.Background(), podList, client.MatchingField("spec.nodeName", node.GetName()))
	if err != nil {
		return &status.Result{}, err
	}

	nodePods := []string{}
	ignoredPods := []navarchosv1alpha1.PodReason{}
	for _, pod := range podList.Items {

		ownerRefs := pod.GetOwnerReferences()
		for _, ref := range ownerRefs {
			if ref.Kind == "DaemonSet" {
				ignoredPods = append(ignoredPods, navarchosv1alpha1.PodReason{Name: pod.GetName(), Reason: "pod owned by a DaemonSet"})
			}
		}

		nodePods = append(nodePods, pod.GetName())
	}

	result.NodePods = nodePods
	result.IgnoredPods = ignoredPods

	inProgress := navarchosv1alpha1.ReplacementPhaseInProgress
	result.Phase = &inProgress

	return result, nil
}
