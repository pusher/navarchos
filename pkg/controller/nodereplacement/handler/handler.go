package handler

import (
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Options are used to configure the NodeReplacementHandler
type Options struct {
	// EvictionGracePeriod determines how long the controller should attempt to
	// evict a pod before marking it a failed eviction
	EvictionGracePeriod *time.Duration

	// DrainTimeout determines how long the controller should attempt to drain a
	// node before timing out. Zero means infinite
	DrainTimeout *time.Duration

	// IgnoreAllDaemonSets instructs the controller to ignore all DaemonSet
	// managed pods. Defaults true
	IgnoreAllDaemonSets *bool

	// DeleteLocalData instructs the controller to delete local data belonging
	// to pods (emptyDir). Defaults true
	DeleteLocalData *bool

	// ForcePodDeletion instructs the controller to continue even if there are
	// pods not managed by a ReplicationController, ReplicaSet, Job, DaemonSet
	// or StatefulSet. Defaults false
	ForcePodDeletion *bool

	// Config is used to construct a kubernetes client
	Config *rest.Config

	// k8sClient is the typed client interface for all standard groups in
	// Kubernetes
	k8sClient kubernetes.Interface
}

// Complete defaults any values that are not explicitly set
func (o *Options) Complete() {
	if o.EvictionGracePeriod == nil {
		grace := 30 * time.Second
		o.EvictionGracePeriod = &grace
	}
	if o.DrainTimeout == nil {
		timeout := 15 * time.Minute
		o.DrainTimeout = &timeout
	}
	if o.IgnoreAllDaemonSets == nil {
		o.IgnoreAllDaemonSets = boolPtr(true)
	}
	if o.DeleteLocalData == nil {
		o.DeleteLocalData = boolPtr(true)
	}
	if o.ForcePodDeletion == nil {
		o.ForcePodDeletion = boolPtr(false)
	}
	if o.Config != nil {
		o.k8sClient = kubernetes.NewForConfigOrDie(o.Config)
	}
}

// NodeReplacementHandler handles the business logic within the NodeReplacement controller.
type NodeReplacementHandler struct {
	client              client.Client
	k8sClient           kubernetes.Interface
	evictionGracePeriod time.Duration
	drainTimeout        time.Duration
	ignoreAllDaemonSets bool
	deleteLocalData     bool
	forcePodDeletion    bool
}

// NewNodeReplacementHandler creates a new NodeReplacementHandler
func NewNodeReplacementHandler(c client.Client, opts *Options) *NodeReplacementHandler {
	opts.Complete()
	return &NodeReplacementHandler{
		client:              c,
		k8sClient:           opts.k8sClient,
		evictionGracePeriod: *opts.EvictionGracePeriod,
		drainTimeout:        *opts.DrainTimeout,
		ignoreAllDaemonSets: *opts.IgnoreAllDaemonSets,
		deleteLocalData:     *opts.DeleteLocalData,
		forcePodDeletion:    *opts.ForcePodDeletion,
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
	case navarchosv1alpha1.ReplacementPhaseCompleted:
		return &status.Result{}, nil
	}
}

func boolPtr(b bool) *bool {
	return &b
}
