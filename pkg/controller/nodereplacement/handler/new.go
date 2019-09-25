package handler

import (
	"context"
	"fmt"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// handleNew handles a NodeReplacement in the New phase
func (h *NodeReplacementHandler) handleNew(instance *navarchosv1alpha1.NodeReplacement) (*status.Result, error) {
	proceed, reason := h.shouldRequeueReplacement(instance)
	if !proceed {
		return &status.Result{
			Requeue:       true,
			RequeueReason: reason,
		}, nil
	}

	node, exists, err := h.getNode(instance)
	if err != nil {
		return &status.Result{}, fmt.Errorf("error getting node: %v", err)
	}
	if !exists {
		completedPhase := navarchosv1alpha1.ReplacementPhaseCompleted
		completedTime := metav1.Now()

		return &status.Result{
			Phase:               &completedPhase,
			CompletionTimestamp: &completedTime,
		}, nil
	}

	err = h.cordonNode(node)
	if err != nil {
		return &status.Result{}, fmt.Errorf("error cordoning node: %v", err)
	}

	result := &status.Result{}
	result.NodePods, result.IgnoredPods, err = h.getPodsToEvict(node)
	if err != nil {
		return result, fmt.Errorf("error listing pods on node %s: %v", node.GetName(), err)
	}

	inProgress := navarchosv1alpha1.ReplacementPhaseInProgress
	result.Phase = &inProgress

	return result, nil
}

// shouldRequeueReplacement determines if a replacement should be processed. If the
// replacement passed should be processed it returns true along with an empty
// reason string, otherwise it returns false with a reason
func (h *NodeReplacementHandler) shouldRequeueReplacement(instance *navarchosv1alpha1.NodeReplacement) (bool, string) {
	replacements := &navarchosv1alpha1.NodeReplacementList{}
	err := h.client.List(context.Background(), replacements)
	if err != nil {
		return false, fmt.Sprintf("failed to list NodeReplacements: %v", err)
	}

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
	node, updated := addTaint(node, &corev1.Taint{
		Key:    "node.kubernetes.io/unschedulable",
		Effect: corev1.TaintEffect("NoSchedule"),
	})
	if !updated {
		return nil
	}

	err := h.client.Update(context.Background(), node)

	return err
}

// addTaint tries to add a taint to annotations list. Returns a new copy of the
// updated Node and true if something was updated false otherwise. When
// determining if the taint already exists only the key:effect are checked, the
// value and time added are disregarded
func addTaint(node *corev1.Node, taint *corev1.Taint) (*corev1.Node, bool) {
	newNode := node.DeepCopy()
	nodeTaints := newNode.Spec.Taints

	var newTaints []corev1.Taint
	for i := range nodeTaints {
		if taint.MatchTaint(&nodeTaints[i]) {
			// break early, taint already exists
			return newNode, false
		}
		// perserve the previous taints§
		newTaints = append(newTaints, nodeTaints[i])

	}

	newTaints = append(newTaints, *taint)
	newNode.Spec.Taints = newTaints

	return newNode, true
}

// getPodsToEvict lists the pods present on a node. It returns a []string
// consisting of all pods on the node and a []PodReason consisitng of all pods
// that are to be ignored
func (h *NodeReplacementHandler) getPodsToEvict(node *corev1.Node) ([]string, []navarchosv1alpha1.PodReason, error) {
	podList := &corev1.PodList{}
	err := h.client.List(context.Background(), podList, client.MatchingField("spec.nodeName", node.GetName()))
	if err != nil {
		return []string{}, []navarchosv1alpha1.PodReason{}, err
	}

	nodePods := []string{}
	ignoredPods := []navarchosv1alpha1.PodReason{}
	for _, pod := range podList.Items {
		nodePods = append(nodePods, pod.GetName())

		ownerRefs := pod.GetOwnerReferences()
		for _, ref := range ownerRefs {
			if ref.Kind == "DaemonSet" {
				ignoredPods = append(ignoredPods, navarchosv1alpha1.PodReason{Name: pod.GetName(), Reason: "pod owned by a DaemonSet"})
			}
		}
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

	// if the node UID has changed, it has already been replaced
	if node.GetUID() != instance.Spec.NodeUID {
		return nil, false, nil
	}

	return node, true, nil
}
