/*
Copyright 2019 Pusher Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodereplacement

import (
	"context"
	"fmt"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/handler"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	watchhandler "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new NodeReplacement Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNodeReplacement{Client: mgr.GetClient(), handler: handler.NewNodeReplacementHandler(mgr.GetClient()), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nodereplacement-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to NodeReplacement
	err = c.Watch(&source.Kind{Type: &navarchosv1alpha1.NodeReplacement{}}, &watchhandler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNodeReplacement{}

// ReconcileNodeReplacement reconciles a NodeReplacement object
type ReconcileNodeReplacement struct {
	client.Client
	handler *handler.NodeReplacementHandler
	scheme  *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NodeReplacement object and makes changes based on the state read
// and what is in the NodeReplacement.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=navarchos.pusher.com,resources=nodereplacements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=navarchos.pusher.com,resources=nodereplacements/status,verbs=get;update;patch
func (r *ReconcileNodeReplacement) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the NodeReplacement instance
	instance := &navarchosv1alpha1.NodeReplacement{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// if the NodeReplacement is Completed or Failed, nothing to do
	if instance.Status.Phase == navarchosv1alpha1.ReplacementPhaseCompleted || instance.Status.Phase == navarchosv1alpha1.ReplacementPhaseFailed {
		return reconcile.Result{}, nil
	}

	// Handle the instance when it is in a New phase, or skip this step
	if instance.Status.Phase == "" || instance.Status.Phase == navarchosv1alpha1.ReplacementPhaseNew {
		result := r.handler.HandleNew(instance)
		err = status.UpdateStatus(r.Client, instance, result)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating status: %v", err)
		}
		if result.Requeue {
			return reconcile.Result{Requeue: true}, nil
		}
	}

	// if the New phase was successful, we can now perform the InProgress phase
	// This phase can be repeated to allow retries until it either fails or
	// completes successfully
	// When the InProgress phase succeeds it should update the status to mark it
	// as Completed, breaking this loop.
	attempts := 0
	for instance.Status.Phase == navarchosv1alpha1.ReplacementPhaseInProgress && attempts < 5 {
		result := r.handler.HandleInProgress(instance)
		err = status.UpdateStatus(r.Client, instance, result)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating status: %v", err)
		}
		// We have had one more attempt at the InProgress Phase
		attempts++
	}

	// If we broke out of the above loop, then we have hit max retries and should
	// mark this NodeReplacement as a failure
	if attempts == 5 {
		failed := navarchosv1alpha1.ReplacementPhaseFailed
		err = status.UpdateStatus(r.Client, instance, &status.Result{Phase: &failed})
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("error updating status: %v", err)
		}
	}

	return reconcile.Result{}, nil
}
