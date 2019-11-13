package handler

import (
	"context"
	"fmt"
	"log"
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
// the succesfully evicted pods through the OnPodDeletedOrEvicted callback
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

// threadsafeErrWriter provides a threadsafe map[string]string. This is used to record
// errors from the drain call that otherwise would not be reported. A map is
// used as a set, removing duplicates.
type threadsafeErrWriter struct {
	errMap map[string]string
	sync.RWMutex
}

// Write implements the io.Writer interface, writing to a map[string]string (poName:error) as the
// concrete type. It also prints the data given to it to stdErr
func (w *threadsafeErrWriter) Write(data []byte) (int, error) {
	w.Lock()
	defer w.Unlock()
	if w.errMap == nil {
		w.errMap = make(map[string]string)
	}
	name, err := parsePodName(string(data))
	if err != nil {
		return fmt.Fprintf(os.Stderr, err.Error())
	}
	w.errMap[name] = string(data)
	return fmt.Fprintf(os.Stderr, string(data))
}

// readErrorMap returns the underlying []error as an aggregate error
func (w *threadsafeErrWriter) ReadErrorMap(evictedPods []string) map[string]string {
	w.Lock()
	defer w.Unlock()
	for _, podName := range evictedPods {
		if _, exists := w.errMap[podName]; exists {
			delete(w.errMap, podName)
		}
	}
	return w.errMap
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

// ReadErrorMap reads the aggregate error out as a map podName[error]
func (f *failedPodError) ReadErrorMap() map[string]string {
	// If the error is not an aggregate, bail..
	e, ok := f.err.(utilerrors.Aggregate)
	if !ok {
		log.Printf("error %v is not an aggregate", f.err)
		return map[string]string{}
	}

	errMap := make(map[string]string, len(e.Errors()))

	for _, err := range e.Errors() {
		name, parseErr := parsePodName(err.Error())
		if parseErr != nil {
			log.Printf("parsing error %v failed: %v\n", err, parseErr)
			continue
		}
		errMap[name] = err.Error()
	}

	return errMap
}

// handleInProgress handles a NodeReplacement in the in progress phase. It
// drains the node specified in the replacement and then marks it completed.
func (h *NodeReplacementHandler) handleInProgress(instance *navarchosv1alpha1.NodeReplacement) (*status.Result, error) {
	// evictedPods captures all pod names that are succesfully evicted. The map
	// is structured [PodName]EmptyString
	evictedPods := threadsafeEvictedPods{
		pods: []string{},
	}

	// errOut captures any errors thrown when a PDB blocks eviction, that
	// otherwise would not be reported. The map is structured podName[Error]
	errOut := &threadsafeErrWriter{
		errMap: make(map[string]string),
	}

	helper := &drain.Helper{
		Client:              h.k8sClient,
		IgnoreAllDaemonSets: h.ignoreAllDaemonSets,
		Timeout:             h.drainTimeout,
		GracePeriodSeconds:  int(h.evictionGracePeriod / time.Second),
		DeleteLocalData:     h.deleteLocalData,
		Force:               h.forcePodDeletion,
		Out:                 os.Stdout,
		ErrOut:              errOut,

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

		// If there is an error for any pod in both the aggregate error and
		// collected from ErrOut, ErrOut takes precedence
		outMap := errOut.ReadErrorMap(evictedPods.readPods())
		aggregateMap := e.ReadErrorMap()

		for k, v := range outMap {
			aggregateMap[k] = v
		}

		// outMap now contains the union of the two maps, with k,v from
		// outMap overwriting those of aggregate
		podReasons := buildPodReasonsFromMap(aggregateMap)

		return &status.Result{
			EvictedPods: evictedPods.readPods(),
			FailedPods:  podReasons,
		}, fmt.Errorf("error draining node: %v", err.Error())
	}

	outMap := errOut.ReadErrorMap(evictedPods.readPods())
	podReasons := buildPodReasonsFromMap(outMap)

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return h.addCompletedLabel(instance.Spec.NodeName)
	})
	if retryErr != nil {
		log.Printf("error labeling node as completed: %v", retryErr)
	}

	completedPhase := navarchosv1alpha1.ReplacementPhaseCompleted
	completedTime := metav1.Now()

	return &status.Result{
		EvictedPods:         evictedPods.readPods(),
		FailedPods:          podReasons,
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
		log.Printf("Warning: %s\n", warnings)
	}

	if err := drainer.DeleteOrEvictPods(list.Pods()); err != nil {
		return failedPodError{err: err}
	}
	return nil
}

// addCompletedLabel adds a label to the node with the passed name. The label
// has format: navarchos.pusher.com/drain-completed:YYYY-MM-DDThhmmss
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

// parsePodName parses a pod name from an error string generated by the drain
// package. If the provided string contains more than one set of double quoes it
// will fail. The string returned is the text within the double quotes.
func parsePodName(err string) (string, error) {
	split := strings.Split(err, "\"")
	// If there is more than one occurance of "" this won't work, bail..
	if len(split) != 3 {
		return "", fmt.Errorf("failed to parse error %s: does not contain exactly one double quoted substring", err)
	}
	return split[1], nil
}

// buildPodReasonsFromMap returns the supplied map as type PodReasons. It trims
// leading and trailing whitespace on the error
func buildPodReasonsFromMap(inMap map[string]string) []navarchosv1alpha1.PodReason {
	reasons := []navarchosv1alpha1.PodReason{}
	for name, err := range inMap {
		reasons = append(reasons, navarchosv1alpha1.PodReason{
			Name:   name,
			Reason: strings.TrimSpace(err),
		})
	}

	return reasons
}
