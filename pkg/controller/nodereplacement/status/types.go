package status

import (
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Result is used as the basis to updating the status of the NodeReplacement.
// It contains information gathered during a single run of the reconcile loop.
type Result struct {
	// This represents the Phase of the NodeReplacement that the status should
	// be set to when updating the status.  If Phase == nil, don't update the
	// Phase, else, overwrite it.
	Phase *navarchosv1alpha1.NodeReplacementPhase

	// CompletionTimestamp is a timestamp for when the rollout has completed
	CompletionTimestamp *metav1.Time

	// This allows the Handler to requeue the object before starting if there is
	// a higher priority NodeReplacement to reconcile
	Requeue bool

	// This should contain a message saying why the NodeReplacement is being
	// Requeued
	RequeueReason string

	// This should contain any error the controller had cordoning the node.
	NodeCordonError error

	// This should contain a short description of the type of error for
	// cordoning the node
	NodeCordonReason navarchosv1alpha1.NodeReplacementConditionReason

	// This should list all Pods on the Node at the time the controller cordons
	// the node.  This should be set on the first pass of the controller only.
	NodePods []string

	// This should be a list of any newly evicted Pods.  This list will be
	// merged with the existing status list.
	EvictedPods []string

	// This should list any Pods not being evicted by the controller.  This
	// should at least contain any DaemonSet pods.
	IgnoredPods []navarchosv1alpha1.PodReason

	// This should be a list of any currently unevictable Pods.  This list will
	// replace the existing status list.
	FailedPods []navarchosv1alpha1.PodReason
}
