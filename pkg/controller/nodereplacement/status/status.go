package status

import (
	"fmt"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UpdateStatus merges the status in the existing instance with the information
// provided in the handler.Result and then updates the instance if there is any
// difference between the new and updated status
func UpdateStatus(c client.Client, instance *navarchosv1alpha1.NodeReplacement, result *Result) error {
	return fmt.Errorf("Method not implemented")
}
