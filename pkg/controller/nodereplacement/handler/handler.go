package handler

import (
	"time"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Options are used to configure the NodeReplacementHandler
type Options struct {
	// EvictionGracePeriod determins how long the controller should attempt to
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

// HandleNew performs the business logic of a New NodeReplacement and returns
// information in a Result
func (h *NodeReplacementHandler) HandleNew(instance *navarchosv1alpha1.NodeReplacement) *status.Result {
	return &status.Result{}
}

// HandleInProgress performs the business logic of an InProgress NodeReplacemen
// and returns information in a Result
func (h *NodeReplacementHandler) HandleInProgress(instance *navarchosv1alpha1.NodeReplacement) *status.Result {
	return &status.Result{}
}
