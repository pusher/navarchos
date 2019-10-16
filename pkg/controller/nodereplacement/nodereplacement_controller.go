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
	"log"

	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/handler"
	"github.com/pusher/navarchos/pkg/controller/nodereplacement/status"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
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
	h := handler.NewNodeReplacementHandler(mgr.GetClient(), &handler.Options{
		Config: mgr.GetConfig(),
	})
	return &ReconcileNodeReplacement{Client: mgr.GetClient(),
		handler:  h,
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor("nodereplacement-controller")}
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

	err = mgr.GetCache().IndexField(&corev1.Pod{}, "spec.nodeName", func(obj runtime.Object) []string {
		pod, _ := obj.(*corev1.Pod)
		return []string{pod.Spec.NodeName}
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileNodeReplacement{}

// ReconcileNodeReplacement reconciles a NodeReplacement object
type ReconcileNodeReplacement struct {
	client.Client
	handler  *handler.NodeReplacementHandler
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a NodeReplacement object and makes changes based on the state read
// and what is in the NodeReplacement.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=navarchos.pusher.com,resources=nodereplacements,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=navarchos.pusher.com,resources=nodereplacements/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=pods/eviction,verbs=create
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch
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

	result, err := r.handler.Handle(instance)
	if err != nil {
		// Ensure we attempt to update the status even when the handler fails
		statusErr := status.UpdateStatus(r.Client, instance, result)
		if statusErr != nil {
			log.Printf("error updating status: %v", statusErr)
		}

		return reconcile.Result{}, fmt.Errorf("error handling replacement %s: %v", instance.GetName(), err)
	}
	err = status.UpdateStatus(r.Client, instance, result)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("error updating status: %v", err)
	}
	if result.Requeue {
		log.Printf("requeueing replacement %s: %s", instance.GetName(), result.RequeueReason)
		r.recorder.Eventf(instance, corev1.EventTypeNormal, "ReplacementRequeue", result.RequeueReason)
		return reconcile.Result{
			Requeue: true,
		}, nil
	}

	return reconcile.Result{}, nil
}
