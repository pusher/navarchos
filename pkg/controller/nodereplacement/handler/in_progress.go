package handler

import (
	"fmt"
	"os"
	"sync"
	"time"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/kubectl/pkg/drain"
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

// handleInProgress handles a NodeReplacement in the in progress phase. It
// drains the node specified in the replacement and then marks it completed.
func (h *NodeReplacementHandler) handleInProgress(instance *navarchosv1alpha1.NodeReplacement) (*status.Result, error) {
	evictedPods := threadsafeEvictedPods{
		pods: []string{},
	}

	helper := &drain.Helper{
		Client:              h.k8sClient,
		IgnoreAllDaemonSets: true,
		Timeout:             15 * time.Minute,
		GracePeriodSeconds:  int(h.evictionGracePeriod / time.Second),
		DeleteLocalData:     true,
		Out:                 os.Stdout,
		ErrOut:              os.Stderr,

		OnPodDeletedOrEvicted: func(pod *corev1.Pod, _ bool) {
			evictedPods.writePod(pod.GetName())
		},
	}

	err := runNodeDrain(helper, instance.Spec.NodeName)
	if err != nil {
		return &status.Result{
			EvictedPods: evictedPods.readPods(),
		}, fmt.Errorf("error draining node: %v", err)
	}

	completedPhase := navarchosv1alpha1.ReplacementPhaseCompleted
	completedTime := metav1.Now()

	return &status.Result{
		EvictedPods:         evictedPods.readPods(),
		Phase:               &completedPhase,
		CompletionTimestamp: &completedTime,
	}, nil
}

// runNodeDrain uses the kubectl drain package to drain a node.
func runNodeDrain(drainer *drain.Helper, nodeName string) error {
	list, errs := drainer.GetPodsForDeletion(nodeName)
	if errs != nil {
		return utilerrors.NewAggregate(errs)
	}
	if warnings := list.Warnings(); warnings != "" {
		fmt.Fprintf(drainer.ErrOut, "WARNING: %s\n", warnings)
	}

	if err := drainer.DeleteOrEvictPods(list.Pods()); err != nil {
		// Maybe warn about non-deleted pods here
		return err
	}
	return nil
}
