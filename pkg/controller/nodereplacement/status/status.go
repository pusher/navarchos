package status

import (
	"context"
	"fmt"
	"reflect"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatus merges the status in the existing instance with the information
// provided in the handler.Result and then updates the instance if there is any
// difference between the new and updated status
func UpdateStatus(c client.Client, instance *navarchosv1alpha1.NodeReplacement, result *Result) error {
	status := instance.Status

	setPhase(&status, result)

	err := setNodePods(&status, result)
	if err != nil {
		return err
	}

	setEvictedPods(&status, result)

	err = setIgnoredPods(&status, result)
	if err != nil {
		return err
	}

	setFailedPods(&status, result)

	err = setCondition(&status, navarchosv1alpha1.NodeCordonedType, result.NodeCordonError, result.NodeCordonReason)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(status, instance.Status) {
		copy := instance.DeepCopy()
		copy.Status = status

		err := c.Update(context.TODO(), copy)
		if err != nil {
			return fmt.Errorf("error updating status: %v", err)
		}
	}

	return nil
}

// setPhase sets the phase when it is set in the result
func setPhase(status *navarchosv1alpha1.NodeReplacementStatus, result *Result) {
	if result.Phase != nil {
		status.Phase = *result.Phase
	}
}

// setNodePods sets the NodePods field, provided it has not been set before
func setNodePods(status *navarchosv1alpha1.NodeReplacementStatus, result *Result) error {
	if status.NodePods != nil && result.NodePods != nil {
		return fmt.Errorf("cannot update NodePods, field is immutable once set")
	}

	if status.NodePods == nil && result.NodePods != nil {
		status.NodePods = result.NodePods
		status.NodePodsCount = len(result.NodePods)
	}

	return nil

}

// setEvictedPods sets the EvictedPods field. If the field was not previously
// set it is added, otherwise the new pods are appended to the previous ones
func setEvictedPods(status *navarchosv1alpha1.NodeReplacementStatus, result *Result) {
	if status.EvictedPods != nil && result.EvictedPods != nil {
		status.EvictedPods = append(status.EvictedPods, result.EvictedPods...)
		status.EvictedPodsCount = len(status.EvictedPods)
	}

	if status.EvictedPods == nil && result.EvictedPods != nil {
		status.EvictedPods = result.EvictedPods
		status.EvictedPodsCount = len(result.EvictedPods)
	}
}

// setIgnoredPods sets the NodePods field, provided it has not been set before
func setIgnoredPods(status *navarchosv1alpha1.NodeReplacementStatus, result *Result) error {
	if status.IgnoredPods != nil && result.IgnoredPods != nil {
		return fmt.Errorf("cannot update IgnoredPods, field is immutable once set")
	}

	if status.IgnoredPods == nil && result.IgnoredPods != nil {
		status.IgnoredPods = result.IgnoredPods
		status.IgnoredPodsCount = len(result.IgnoredPods)
	}

	return nil
}

// setFailedPods sets the FailedPods field if it is set in the result
func setFailedPods(status *navarchosv1alpha1.NodeReplacementStatus, result *Result) {
	if result.FailedPods != nil {
		status.FailedPods = result.FailedPods
		status.FailedPodsCount = len(result.FailedPods)
	}
}

// newNodeReplacementCondition creates a new condition NodeReplacementCondition
func newNodeReplacementCondition(condType navarchosv1alpha1.NodeReplacementConditionType, status corev1.ConditionStatus, reason string, message string) *navarchosv1alpha1.NodeReplacementCondition {
	return &navarchosv1alpha1.NodeReplacementCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
}

// getNodeReplacementCondition returns the condition with the provided type
func getNodeReplacementCondition(status navarchosv1alpha1.NodeReplacementStatus, condType navarchosv1alpha1.NodeReplacementConditionType) *navarchosv1alpha1.NodeReplacementCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// setNodeReplacementCondition updates the NodeReplacement to include the
// provided condition. If the condition that we are about to add already exists
// and has the same status and reason then we are not going to update
func setNodeReplacementCondition(status *navarchosv1alpha1.NodeReplacementStatus, condition navarchosv1alpha1.NodeReplacementCondition) {
	currentCond := getNodeReplacementCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't
	// change
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of NodeReplacement conditions without
// conditions with the provided type
func filterOutCondition(conditions []navarchosv1alpha1.NodeReplacementCondition, condType navarchosv1alpha1.NodeReplacementConditionType) []navarchosv1alpha1.NodeReplacementCondition {
	var newConditions []navarchosv1alpha1.NodeReplacementCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

func setCondition(status *navarchosv1alpha1.NodeReplacementStatus, condType navarchosv1alpha1.NodeReplacementConditionType, condErr error, reason string) error {
	if (condErr == nil) != (reason == "") {
		return fmt.Errorf("either NodeCordonError or NodeCordonReason is not set")
	}
	if condErr != nil {
		// Error for condition , set condition appropriately
		cond := newNodeReplacementCondition(
			condType,
			corev1.ConditionFalse,
			reason,
			condErr.Error(),
		)
		setNodeReplacementCondition(status, *cond)
		return nil
	}

	// No error for condition, set condition appropriately
	cond := newNodeReplacementCondition(
		condType,
		corev1.ConditionTrue,
		reason,
		"",
	)
	setNodeReplacementCondition(status, *cond)

	return nil
}
