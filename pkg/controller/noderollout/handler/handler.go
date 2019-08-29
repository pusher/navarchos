package handler

import (
	"context"
	"fmt"
	"time"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/noderollout/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metalabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

type nodeReplacementSpec struct {
	node            corev1.Node
	replacementSpec navarchosv1alpha1.NodeReplacementSpec
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

// handleNew handles a NodeRollout in the 'New' phase
func (h *NodeRolloutHandler) handleNew(instance *navarchosv1alpha1.NodeRollout) *status.Result {
	result := &status.Result{}
	nodes := &corev1.NodeList{}
	err := h.client.List(context.Background(), nodes)
	if err != nil {
		result.ReplacementsCompletedError = err
		return result
	}

	var nodeReplacementMap map[string]nodeReplacementSpec
	nodeReplacementMap = make(map[string]nodeReplacementSpec)

	nodeReplacementMap, err = filterNodeSelectors(nodes, instance.Spec.NodeSelectors, nodeReplacementMap)
	if err != nil {
		result.ReplacementsCompletedError = err
		return result
	}

	nodeReplacementMap = filterNodeNames(nodes, instance.Spec.NodeNames, nodeReplacementMap)

	// create the NodeReplacements
	for _, spec := range nodeReplacementMap {
		nodeReplacement := createNodeReplacementFromSpec(spec.replacementSpec, instance, &spec.node)
		err := h.client.Create(context.Background(), &nodeReplacement)
		if err != nil {
			result.ReplacementsCompletedError = err
			return result
		}
		result.ReplacementsCreated = append(result.ReplacementsCreated, spec.replacementSpec.NodeName)
	}
	newPhase := navarchosv1alpha1.RolloutPhaseInProgress
	result.Phase = &newPhase

	return result
}

// filterNodeSelectors filters the list of all nodes. If a nodes labels match it
// adds the node to the nodeMap
func filterNodeSelectors(nodes *corev1.NodeList, selectors []navarchosv1alpha1.NodeLabelSelector, nodeMap map[string]nodeReplacementSpec) (map[string]nodeReplacementSpec, error) {
	for _, node := range nodes.Items {
		labels := metalabels.Set(node.GetLabels())
		for _, nls := range selectors {
			selector, err := metav1.LabelSelectorAsSelector(&nls.LabelSelector)
			if err != nil {
				return nil, err
			}
			if selector.Matches(labels) {
				nodeMap[node.GetName()] = nodeReplacementSpec{
					node: node,
					replacementSpec: navarchosv1alpha1.NodeReplacementSpec{
						ReplacementSpec: nls.ReplacementSpec,
						NodeName:        node.GetName(),
						NodeUID:         node.GetUID(),
					},
				}
			}

		}
	}
	return nodeMap, nil
}

// filterNodeNames filters the list of all nodes. If a nodes name matches one
// provided it adds the node to the nodeMap
func filterNodeNames(nodes *corev1.NodeList, nodeNames []navarchosv1alpha1.NodeName, nodeMap map[string]nodeReplacementSpec) map[string]nodeReplacementSpec {
	for _, node := range nodes.Items {
		for _, selectedName := range nodeNames {
			if node.GetName() == selectedName.Name {
				nodeMap[node.GetName()] = nodeReplacementSpec{
					node: node,
					replacementSpec: navarchosv1alpha1.NodeReplacementSpec{
						ReplacementSpec: selectedName.ReplacementSpec,
						NodeName:        node.GetName(),
						NodeUID:         node.GetUID(),
					},
				}
			}
		}
	}
	return nodeMap
}

func createNodeReplacementFromSpec(spec navarchosv1alpha1.NodeReplacementSpec, rolloutOwner *navarchosv1alpha1.NodeRollout, nodeOwner *corev1.Node) navarchosv1alpha1.NodeReplacement {
	// gvk := schema.GroupVersionKind{
	// 	Group:   "navarchos.pusher.com",
	// 	Version: "v1alpha1",
	// 	Kind:    "NodeReplacement",
	// }

	nodeReplacement := navarchosv1alpha1.NodeReplacement{
		TypeMeta: metav1.TypeMeta{
			APIVersion: navarchosv1alpha1.SchemeGroupVersion.String(),
			Kind:       "NodeReplacement",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", spec.NodeName),
			OwnerReferences: []metav1.OwnerReference{
				newOwnerRef(rolloutOwner, rolloutOwner.GroupVersionKind(), true, true),
				newOwnerRef(nodeOwner, nodeOwner.GroupVersionKind(), false, false),
			},
		},
		Spec:   spec,
		Status: navarchosv1alpha1.NodeReplacementStatus{},
	}
	return nodeReplacement
}

// newOwnerRef creates an OwnerReference pointing to the given owner.
func newOwnerRef(owner metav1.Object, gvk schema.GroupVersionKind, isController bool, blockOwnerDeletion bool) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               owner.GetName(),
		UID:                owner.GetUID(),
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	}
}
func (h *NodeRolloutHandler) handleInProgress(instance *navarchosv1alpha1.NodeRollout) *status.Result {
	return &status.Result{}
}
func (h *NodeRolloutHandler) handleCompleted(instance *navarchosv1alpha1.NodeRollout) *status.Result {
	return &status.Result{}
}
