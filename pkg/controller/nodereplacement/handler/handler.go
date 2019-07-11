package handler

import (
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeReplacementHandler handles the business logic within the NodeReplacement controller.
type NodeReplacementHandler struct {
	client client.Client
}

// NewNodeReplacementHandler creates a new NodeReplacementHandler
func NewNodeReplacementHandler(c client.Client) *NodeReplacementHandler {
	return &NodeReplacementHandler{client: c}
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
