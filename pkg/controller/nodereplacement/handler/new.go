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

	proceed, reason := shouldProcess(instance, replacements)
	if !proceed {
		result.Requeue = true
		result.RequeueReason = reason
		return result, nil
	}

	node, exists, err := h.getNode(instance)
	if err != nil {
		return &status.Result{}, fmt.Errorf("error getting node: %v", err)
	}
	if !exists {
		completedPhase := navarchosv1alpha1.ReplacementPhaseCompleted
		completedTime := metav1.Now()

		result.Phase = &completedPhase
		result.CompletionTimestamp = &completedTime

		return result, nil
	}

	err = h.cordonNode(node)
	if err != nil {
		return &status.Result{}, fmt.Errorf("error cordoning node: %v", err)
	}

	result = &status.Result{}
	result.NodePods, result.IgnoredPods, err = h.processPods(node)
	if err != nil {
		return &status.Result{}, fmt.Errorf("error listing pods on node %s: %v", node.GetName(), err)
	}

	inProgress := navarchosv1alpha1.ReplacementPhaseInProgress
	result.Phase = &inProgress

	return result, nil
}

// shouldProcess determines if a replacement should be processed. If the
// replacement passed should be processed it returns true along with an empty
// reason string, otherwise it returns false with a reason
func shouldProcess(instance *navarchosv1alpha1.NodeReplacement, replacements *navarchosv1alpha1.NodeReplacementList) (bool, string) {
	for _, replacement := range replacements.Items {
		if *replacement.Spec.ReplacementSpec.Priority > *instance.Spec.ReplacementSpec.Priority {
			reason := fmt.Sprintf("NodeReplacement \"%s\" has a higher priority", replacement.GetName())
			return false, reason
		}
		if replacement.Status.Phase == navarchosv1alpha1.ReplacementPhaseInProgress {
			reason := fmt.Sprintf("NodeReplacement \"%s\" is already in-progress", replacement.GetName())
			return false, reason
		}
	}

	return true, ""
}

// cordonNode cordons a node
func (h *NodeReplacementHandler) cordonNode(node *corev1.Node) error {
	node.Spec.Unschedulable = true
	node, _, err := taints.AddOrUpdateTaint(node, &corev1.Taint{
		Key:    "node.kubernetes.io/unschedulable",
		Effect: corev1.TaintEffect("NoSchedule"),
	})
	if err != nil {
		return err
	}

	err = h.client.Update(context.Background(), node)

	return err
}

// processPods processes the pods present on a node. It returns a []string
// consisting of all pods on the node and a []PodReason consisitng of all pods
// that are ignored
func (h *NodeReplacementHandler) processPods(node *corev1.Node) ([]string, []navarchosv1alpha1.PodReason, error) {
	podList := &corev1.PodList{}
	err := h.client.List(context.Background(), podList, client.MatchingField("spec.nodeName", node.GetName()))
	if err != nil {
		return []string{}, []navarchosv1alpha1.PodReason{}, err
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

	return nodePods, ignoredPods, nil
}

// getNode gets the node specified in a NodeReplacement. If it does not exist it
// returns false, otherwise it returns the node and a true bool
func (h *NodeReplacementHandler) getNode(instance *navarchosv1alpha1.NodeReplacement) (*corev1.Node, bool, error) {
	node := &corev1.Node{}
	err := h.client.Get(context.Background(), client.ObjectKey{
		Name: instance.Spec.NodeName,
	}, node)

	if err != nil {
		if errors.IsNotFound(err) {
			return nil, false, nil
		}

		return nil, false, err
	}

	// if  the node UID has changed, it has already been replaced
	if node.GetUID() != instance.Spec.NodeUID {
		return nil, false, nil
	}

	return node, true, nil
}
