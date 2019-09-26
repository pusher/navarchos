package handler

import (
	"fmt"
	"time"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Options are used to configure the NodeReplacementHandler
type Options struct {
	// EvictionGracePeriod determines how long the controller should attempt to
	// evict a pod before marking it a failed eviction
	EvictionGracePeriod *time.Duration
}

// Complete defaults any values that are not explicitly set
func (o *Options) Complete() {
	if o.EvictionGracePeriod == nil {
		grace := 30 * time.Second
		o.EvictionGracePeriod = &grace
	}
}

// NodeReplacementHandler handles the business logic within the NodeReplacement controller.
type NodeReplacementHandler struct {
	client              client.Client
	evictionGracePeriod time.Duration
}

// NewNodeReplacementHandler creates a new NodeReplacementHandler
func NewNodeReplacementHandler(c client.Client, opts *Options) *NodeReplacementHandler {
	opts.Complete()
	return &NodeReplacementHandler{
		client:              c,
		evictionGracePeriod: *opts.EvictionGracePeriod,
	}
}

// Handle performs the business logic of a NodeReplacement and returns
// information in a Result. The use of fallthrough is to ensure that one
// instance of a NodeReplacement can be handled in full without interruption
func (h *NodeReplacementHandler) Handle(instance *navarchosv1alpha1.NodeReplacement) (*status.Result, error) {
	var result = &status.Result{}
	var err error

	switch instance.Status.Phase {
	default:
		newPhase := navarchosv1alpha1.ReplacementPhaseNew
		// Update status before starting next phase. This updates the instance
		// phase too, it is mutated in place...
		err = status.UpdateStatus(h.client, instance, &status.Result{
			Phase: &newPhase,
		})
		if err != nil {
			// This is an API error which means other errors are also likely,
			// bail and requeue We will reattempt the status update anyway where
			// this is called
			return result, fmt.Errorf("error updating status: %v", err)
		}

		fallthrough // This is important, we want one instance to be handled to completion without a requeue if possible
	case navarchosv1alpha1.ReplacementPhaseNew:
		result, err = h.handleNew(instance)
		if err != nil {
			return result, err
		}
		if result.Requeue {
			return result, nil
		}

		// Update status before starting next phase
		err = status.UpdateStatus(h.client, instance, result)
		if err != nil {
			return result, fmt.Errorf("error updating status: %v", err)
		}

		fallthrough // This is important, we want one instance to be handled to completion without a requeue if possible
	case navarchosv1alpha1.ReplacementPhaseInProgress:
		result, err = h.handleInProgress(instance)
		if err != nil {
			return result, err
		}
		// Nothing left to do
		return result, nil
	}
}

func (h *NodeReplacementHandler) handleInProgress(instance *navarchosv1alpha1.NodeReplacement) (*status.Result, error) {
	return &status.Result{}, nil
}
