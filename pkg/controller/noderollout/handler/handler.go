package handler

import (
	"time"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/noderollout/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Options are used to configure the NodeRolloutHandler
type Options struct {
	// MaxAge determines the maximum age a NodeRollout should be before it is
	// garbage collected
	MaxAge *time.Duration
}

// Complete defaults any values that are not explicitly set
func (o *Options) Complete() {
	if o.MaxAge == nil {
		maxAge := 48 * time.Hour
		o.MaxAge = &maxAge
	}
}

// NodeRolloutHandler handles the business logic within the NodeRollout controller.
type NodeRolloutHandler struct {
	client client.Client
	maxAge time.Duration
}

// NewNodeRolloutHandler creates a new NodeRolloutHandler
func NewNodeRolloutHandler(c client.Client, opts *Options) *NodeRolloutHandler {
	opts.Complete()
	return &NodeRolloutHandler{
		client: c,
		maxAge: *opts.MaxAge,
	}
}

// Handle performs the business logic of the NodeRollout and returns information
// in a Result
func (h *NodeRolloutHandler) Handle(instance *navarchosv1alpha1.NodeRollout) *status.Result {
	switch instance.Status.Phase {
	case navarchosv1alpha1.RolloutPhaseNew:
		return h.handleNew(instance)
	case navarchosv1alpha1.RolloutPhaseInProgress:
		return h.handleInProgress(instance)
	case navarchosv1alpha1.RolloutPhaseCompleted:
		return h.handleCompleted(instance)
	default:
		return h.handleNew(instance)
	}
}

func (h *NodeRolloutHandler) handleCompleted(instance *navarchosv1alpha1.NodeRollout) *status.Result {
	return &status.Result{}
}
