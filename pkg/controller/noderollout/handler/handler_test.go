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

package handler

import (
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"github.com/pusher/navarchos/pkg/controller/noderollout/status"
	"github.com/pusher/navarchos/test/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var _ = Describe("Handler suite", func() {
	var m utils.Matcher
	var h *NodeRolloutHandler
	var result *status.Result

	var nodeRollout *navarchosv1alpha1.NodeRollout
	var mgrStopped *sync.WaitGroup
	var stopMgr chan struct{}

	var masterNode1 *corev1.Node
	var masterNode2 *corev1.Node
	var workerNode1 *corev1.Node
	var workerNode2 *corev1.Node
	var otherNode *corev1.Node

	const timeout = time.Second * 5
	const consistentlyTimeout = time.Second
	BeforeEach(func() {
		mgr, err := manager.New(cfg, manager.Options{})
		Expect(err).NotTo(HaveOccurred())
		c := mgr.GetClient()
		m = utils.Matcher{Client: c}

		stopMgr, mgrStopped = StartTestManager(mgr)

		nodeRollout = utils.ExampleNodeRollout.DeepCopy()
		m.Create(nodeRollout).Should(Succeed())

		// Create some nodes to act as owners for the NodeReplacements created
		masterNode1 = utils.ExampleNodeMaster1.DeepCopy()
		masterNode2 = utils.ExampleNodeMaster2.DeepCopy()
		workerNode1 = utils.ExampleNodeWorker1.DeepCopy()
		workerNode2 = utils.ExampleNodeWorker2.DeepCopy()
		otherNode = utils.ExampleNodeOther.DeepCopy()

		m.Create(masterNode1).Should(Succeed())
		m.Create(masterNode2).Should(Succeed())
		m.Create(workerNode1).Should(Succeed())
		m.Create(workerNode2).Should(Succeed())
		m.Create(otherNode).Should(Succeed())
	})

	AfterEach(func() {
		close(stopMgr)
		mgrStopped.Wait()

		utils.DeleteAll(cfg, timeout,
			&navarchosv1alpha1.NodeRolloutList{},
			&navarchosv1alpha1.NodeReplacementList{},
			&corev1.NodeList{},
		)
	})
})
