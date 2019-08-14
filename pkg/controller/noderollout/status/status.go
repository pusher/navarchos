package status

import (
	"context"
	"fmt"
	"reflect"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatus merges the status in the existing instance with the information
// provided in the handler. Result and then updates the instance if there is any
// difference between the new and updated status
func UpdateStatus(c client.Client, instance *navarchosv1alpha1.NodeRollout, result *Result) error {
	status := instance.Status

	setPhase(&status, result)

	err := setReplacementsCreated(&status, result)
	if err != nil {
		return err
	}

	setReplacementsCompleted(&status, result)
	err = setCondition(&status, navarchosv1alpha1.ReplacementsCreatedType, result.ReplacementsCompletedError, result.ReplacementsCompletedReason)
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

// setPhase sets the phase when it is set in the Result
func setPhase(status *navarchosv1alpha1.NodeRolloutStatus, result *Result) {
	if result.Phase != nil {
		status.Phase = *result.Phase
	}
}

// setReplacementsCreated sets the ReplacementsCreated, provided it has not been
// set before
func setReplacementsCreated(status *navarchosv1alpha1.NodeRolloutStatus, result *Result) error {
	if status.ReplacementsCreated != nil && result.ReplacementsCreated != nil {
		return fmt.Errorf("cannot update ReplacementsCreated, field is immutable once set")
	}

	if status.ReplacementsCreated == nil && result.ReplacementsCreated != nil {
		status.ReplacementsCreated = result.ReplacementsCreated
		status.ReplacementsCreatedCount = len(result.ReplacementsCreated)
	}

	return nil
}

// setReplacementsCompleted sets the ReplacementsCompleted, if it has not been
// set before it is added. If it has been set before the two are appended
func setReplacementsCompleted(status *navarchosv1alpha1.NodeRolloutStatus, result *Result) {
	if status.ReplacementsCompleted != nil && result.ReplacementsCompleted != nil {
		status.ReplacementsCompleted = append(status.ReplacementsCompleted, result.ReplacementsCompleted...)
		status.ReplacementsCompletedCount = len(status.ReplacementsCompleted)
	}

	if status.ReplacementsCompleted == nil && result.ReplacementsCompleted != nil {
		status.ReplacementsCompleted = result.ReplacementsCompleted
		status.ReplacementsCompletedCount = len(result.ReplacementsCompleted)
	}

}

// newNodeRolloutCondition creates a new condition NodeRolloutCondition
func newNodeRolloutCondition(condType navarchosv1alpha1.NodeRolloutConditionType, status v1.ConditionStatus, reason string, message string) *navarchosv1alpha1.NodeRolloutCondition {
	return &navarchosv1alpha1.NodeRolloutCondition{
		Type:               condType,
		Status:             status,
		LastUpdateTime:     metav1.Now(),
		LastTransitionTime: metav1.Now(),
		Reason:             string(reason),
		Message:            message,
	}
}

// getNodeRolloutCondition returns the condition with the provided type
func getNodeRolloutCondition(status navarchosv1alpha1.NodeRolloutStatus, condType navarchosv1alpha1.NodeRolloutConditionType) *navarchosv1alpha1.NodeRolloutCondition {
	for i := range status.Conditions {
		c := status.Conditions[i]
		if c.Type == condType {
			return &c
		}
	}
	return nil
}

// setNodeRolloutCondition updates the NodeRollout to include the provided condition. If the condition that
// we are about to add already exists and has the same status and reason then we are not going to update
func setNodeRolloutCondition(status *navarchosv1alpha1.NodeRolloutStatus, condition navarchosv1alpha1.NodeRolloutCondition) {
	currentCond := getNodeRolloutCondition(*status, condition.Type)
	if currentCond != nil && currentCond.Status == condition.Status && currentCond.Reason == condition.Reason {
		return
	}
	// Do not update lastTransitionTime if the status of the condition doesn't change
	if currentCond != nil && currentCond.Status == condition.Status {
		condition.LastTransitionTime = currentCond.LastTransitionTime
	}
	newConditions := filterOutCondition(status.Conditions, condition.Type)
	status.Conditions = append(newConditions, condition)
}

// filterOutCondition returns a new slice of NodeRollout conditions without conditions with the provided types
func filterOutCondition(conditions []navarchosv1alpha1.NodeRolloutCondition, condType navarchosv1alpha1.NodeRolloutConditionType) []navarchosv1alpha1.NodeRolloutCondition {
	var newConditions []navarchosv1alpha1.NodeRolloutCondition
	for _, c := range conditions {
		if c.Type == condType {
			continue
		}
		newConditions = append(newConditions, c)
	}
	return newConditions
}

func setCondition(status *navarchosv1alpha1.NodeRolloutStatus, condType navarchosv1alpha1.NodeRolloutConditionType, condErr error, reason string) error {
	if (condErr == nil) != (reason == "") {
		return fmt.Errorf("Either ReplacementsCompletedError or ReplacementsCompletedReason is not set")
	}
	if condErr != nil {
		// Error for condition , set condition appropriately
		cond := newNodeRolloutCondition(
			condType,
			v1.ConditionFalse,
			reason,
			condErr.Error(),
		)
		setNodeRolloutCondition(status, *cond)
		return nil
	}

	// No error for condition, set condition appropriately
	cond := newNodeRolloutCondition(
		condType,
		v1.ConditionTrue,
		reason,
		"",
	)
	setNodeRolloutCondition(status, *cond)

	return nil
}
