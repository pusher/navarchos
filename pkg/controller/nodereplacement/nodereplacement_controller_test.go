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
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("NodeReplacement controller suite", func() {
	var c client.Client

	var nodeReplacement *navarchosv1alpha1.NodeReplacement
	var requests <-chan reconcile.Request
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	const timeout = time.Second * 5

	var waitForReconcile = func(obj metav1.Object) {
		request := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}
		// wait for reconcile for the object
		Eventually(requests, timeout).Should(Receive(Equal(request)))
	}

	BeforeEach(func() {
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c = mgr.GetClient()

		var recFn reconcile.Reconciler
		recFn, requests = SetupTestReconcile(newReconciler(mgr))
		Expect(add(mgr, recFn)).NotTo(HaveOccurred())

		stopMgr, mgrStopped = StartTestManager(mgr)

		nodeReplacement = &navarchosv1alpha1.NodeReplacement{}
	})

	AfterEach(func() {
		close(stopMgr)
		mgrStopped.Wait()
	})

	Context("when a NodeReplacement is reconciled", func() {
		BeforeEach(func() {
			waitForReconcile(nodeReplacement)
			c.Get(
				context.TODO(),
				types.NamespacedName{
					Name:      nodeReplacement.GetName(),
					Namespace: nodeReplacement.GetNamespace(),
				},
				nodeReplacement,
			)
		})
	})
})
