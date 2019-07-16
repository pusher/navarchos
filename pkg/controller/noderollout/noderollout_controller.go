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

package noderollout

import (
	"context"
	"fmt"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/noderollout/handler"
	"github.com/pusher/navarchos/pkg/controller/noderollout/status"
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

// Add creates a new NodeRollout Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	h := handler.NewNodeRolloutHandler(mgr.GetClient(), &handler.Options{})
	return &ReconcileNodeRollout{Client: mgr.GetClient(), handler: h, scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("noderollout-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to NodeRollout
	err = c.Watch(&source.Kind{Type: &navarchosv1alpha1.NodeRollout{}}, &watchhandler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for NodeReplacements created by NodeRollout
	err = c.Watch(&source.Kind{Type: &navarchosv1alpha1.NodeReplacement{}}, &watchhandler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &navarchosv1alpha1.NodeRollout{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNodeRollout{}

// ReconcileNodeRollout reconciles a NodeRollout object
type ReconcileNodeRollout struct {
	client.Client
	handler *handler.NodeRolloutHandler
	scheme  *runtime.Scheme
}

// Reconcile reads that state of the cluster for a NodeRollout object and makes changes based on the state read
// and what is in the NodeRollout.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=navarchos.pusher.com,resources=noderollouts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=navarchos.pusher.com,resources=noderollouts/status,verbs=get;update;patch
func (r *ReconcileNodeRollout) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the NodeRollout instance
	instance := &navarchosv1alpha1.NodeRollout{}
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

	result := r.handler.Handle(instance)
	err = status.UpdateStatus(r.Client, instance, result)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error updating status: %v", err)
	}

	return reconcile.Result{}, nil
}
