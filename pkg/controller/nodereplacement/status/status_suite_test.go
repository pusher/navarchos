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

package status

import (
	"log"
	"path/filepath"
	"testing"

	"github.com/go-logr/glogr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pusher/navarchos/pkg/apis"
	"github.com/pusher/navarchos/test/reporters"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var cfg *rest.Config

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "NodeReplacement Status Suite", reporters.Reporters())
}

var t *envtest.Environment

var _ = BeforeSuite(func() {
	t = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "..", "..", "config", "crds")},
	}
	apis.AddToScheme(scheme.Scheme)

	logf.SetLogger(glogr.New())

	var err error
	if cfg, err = t.Start(); err != nil {
		log.Fatal(err)
	}
})

var _ = AfterSuite(func() {
	t.Stop()
})
