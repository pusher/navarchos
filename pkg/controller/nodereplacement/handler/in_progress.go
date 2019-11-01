package handler

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/util/retry"
	"k8s.io/kubectl/pkg/drain"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// threadsafeEvictedPods provides a threadsafe []string. This is used to record
// the succesfully evicted pods through the OnPodDeletedOrEvicted callback.
type threadsafeEvictedPods struct {
	sync.RWMutex
	pods []string
}

func (e *threadsafeEvictedPods) writePod(podName string) {
	e.Lock()
	defer e.Unlock()
	e.pods = append(e.pods, podName)
}

func (e *threadsafeEvictedPods) readPods() []string {
	e.RLock()
	defer e.RUnlock()
	return e.pods
}

// failedPodError implements the Error interface and provides a method to lazily
// retrieve the PodReasons
type failedPodError struct {
	err        error
	failedPods []navarchosv1alpha1.PodReason
}

func (f failedPodError) Error() string {
	return f.err.Error()
}

// PodReasons parses the aggregate error and returns the infzormation as type
// PodReasons
func (f *failedPodError) PodReasons() []navarchosv1alpha1.PodReason {
	if f.failedPods != nil {
		return f.failedPods
	}
	reasons := []navarchosv1alpha1.PodReason{}
	// If the error is not an aggregate, bail..
	e, ok := f.err.(utilerrors.Aggregate)
	if !ok {
		return reasons
	}

	for _, err := range e.Errors() {
		split := strings.Split(err.Error(), "\"")
		// If there is more than one occurance of "" this won't work, bail..
		if len(split) != 2 {
			continue
		}
		reasons = append(reasons, navarchosv1alpha1.PodReason{
			Name:   split[1],
			Reason: err.Error(),
		})
	}
	f.failedPods = reasons

	return reasons
}

// handleInProgress handles a NodeReplacement in the in progress phase. It
// drains the node specified in the replacement and then marks it completed.
func (h *NodeReplacementHandler) handleInProgress(instance *navarchosv1alpha1.NodeReplacement) (*status.Result, error) {
	evictedPods := threadsafeEvictedPods{
		pods: []string{},
	}

	helper := &drain.Helper{
		Client:              h.k8sClient,
		IgnoreAllDaemonSets: h.ignoreAllDaemonSets,
		Timeout:             h.drainTimeout,
		GracePeriodSeconds:  int(h.evictionGracePeriod / time.Second),
		DeleteLocalData:     h.deleteLocalData,
		Force:               h.forcePodDeletion,
		Out:                 os.Stdout,
		ErrOut:              os.Stderr,

		OnPodDeletedOrEvicted: func(pod *corev1.Pod, _ bool) {
			evictedPods.writePod(pod.GetName())
		},
	}

	err := runNodeDrain(helper, instance.Spec.NodeName)
	if err != nil {
		e, ok := err.(failedPodError)
		if !ok {
			// the type assertion has failed for some reason...
			// it shouldn't have, bail..
			return &status.Result{
				EvictedPods: evictedPods.readPods(),
			}, fmt.Errorf("error draining node: %v", err)
		}
		return &status.Result{
			EvictedPods: evictedPods.readPods(),
			FailedPods:  e.PodReasons(),
		}, fmt.Errorf("error draining node: %v", err.Error())
	}

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return h.addCompletedLabel(instance.Spec.NodeName)
	})
	if retryErr != nil {
		return &status.Result{
			EvictedPods: evictedPods.readPods(),
		}, fmt.Errorf("error labeling node as completed: %v", retryErr)
	}

	completedPhase := navarchosv1alpha1.ReplacementPhaseCompleted
	completedTime := metav1.Now()

	return &status.Result{
		EvictedPods:         evictedPods.readPods(),
		Phase:               &completedPhase,
		CompletionTimestamp: &completedTime,
	}, nil
}

// runNodeDrain uses the kubectl drain package to drain a node. If any pods
// fail, it unpacks the individual error from the aggregate and returns them
// individually
func runNodeDrain(drainer *drain.Helper, nodeName string) error {
	list, errs := drainer.GetPodsForDeletion(nodeName)
	if errs != nil {
		return utilerrors.NewAggregate(errs)
	}
	if warnings := list.Warnings(); warnings != "" {
		fmt.Fprintf(drainer.ErrOut, "WARNING: %s\n", warnings)
	}

	if err := drainer.DeleteOrEvictPods(list.Pods()); err != nil {

		return failedPodError{err: err}
	}
	return nil
}

func (h *NodeReplacementHandler) addCompletedLabel(nodeName string) error {
	node := &corev1.Node{}
	err := h.client.Get(context.Background(), client.ObjectKey{
		Name: nodeName,
	}, node)
	if err != nil {
		return err
	}

	nodeLabels := node.GetLabels()
	nodeLabels["navarchos.pusher.com/drain-completed"] = time.Now().Format("2006-01-02T15h04m05s")
	node.SetLabels(nodeLabels)

	return h.client.Update(context.Background(), node)
}
