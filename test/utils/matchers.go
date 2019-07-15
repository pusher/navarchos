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

package utils

import (
	"context"
	"reflect"

	"github.com/onsi/gomega"
	gtypes "github.com/onsi/gomega/types"
	navarchosv1alpha1 "github.com/pusher/navarchos/pkg/apis/navarchos/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Matcher has Gomega Matchers that use the controller-runtime client
type Matcher struct {
	Client client.Client
}

// Object is the combination of two interfaces as a helper for passing
// Kubernetes objects between methods
type Object interface {
	runtime.Object
	metav1.Object
}

// UpdateFunc modifies the object fetched from the API server before sending
// the update
type UpdateFunc func(Object) Object

// Create creates the object on the API server
func (m *Matcher) Create(obj Object, extras ...interface{}) gomega.GomegaAssertion {
	err := m.Client.Create(context.TODO(), obj)
	return gomega.Expect(err, extras)
}

// Delete deletes the object from the API server
func (m *Matcher) Delete(obj Object, extras ...interface{}) gomega.GomegaAssertion {
	err := m.Client.Delete(context.TODO(), obj)
	return gomega.Expect(err, extras)
}

// Update udpates the object on the API server by fetching the object
// and applying a mutating UpdateFunc before sending the update
func (m *Matcher) Update(obj Object, fn UpdateFunc, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	update := func() error {
		err := m.Client.Get(context.TODO(), key, obj)
		if err != nil {
			return err
		}
		return m.Client.Update(context.TODO(), fn(obj))
	}
	return gomega.Eventually(update, intervals...)
}

// Get gets the object from the API server
func (m *Matcher) Get(obj Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	get := func() error {
		return m.Client.Get(context.TODO(), key, obj)
	}
	return gomega.Eventually(get, intervals...)
}

// Consistently continually gets the object from the API for comparison
func (m *Matcher) Consistently(obj Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	return m.consistentlyObject(obj, intervals...)
}

// consistentlyObject gets an individual object from the API server
func (m *Matcher) consistentlyObject(obj Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	get := func() Object {
		err := m.Client.Get(context.TODO(), key, obj)
		if err != nil {
			panic(err)
		}
		return obj
	}
	return gomega.Consistently(get, intervals...)
}

// Eventually continually gets the object from the API for comparison
func (m *Matcher) Eventually(obj runtime.Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	// If the object is a list, return a list
	if meta.IsListType(obj) {
		return m.eventuallyList(obj, intervals...)
	}
	if o, ok := obj.(Object); ok {
		return m.eventuallyObject(o, intervals...)
	}
	//Should not get here
	panic("Unknown object.")
}

// eventuallyObject gets an individual object from the API server
func (m *Matcher) eventuallyObject(obj Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	key := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
	get := func() Object {
		err := m.Client.Get(context.TODO(), key, obj)
		if err != nil {
			panic(err)
		}
		return obj
	}
	return gomega.Eventually(get, intervals...)
}

// eventuallyList gets a list type  from the API server
func (m *Matcher) eventuallyList(obj runtime.Object, intervals ...interface{}) gomega.GomegaAsyncAssertion {
	list := func() runtime.Object {
		err := m.Client.List(context.TODO(), &client.ListOptions{}, obj)
		if err != nil {
			panic(err)
		}
		return obj
	}
	return gomega.Eventually(list, intervals...)
}

// WithNodeRolloutStatusField gets the value of the named field from the
// NodeRollouts Status
func WithNodeRolloutStatusField(field string, matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj *navarchosv1alpha1.NodeRollout) interface{} {
		r := reflect.ValueOf(obj.Status)
		f := reflect.Indirect(r).FieldByName(field)
		return f.Interface()
	}, matcher)
}

// WithNodeRolloutConditionField gets the value of the named field from the
// NodeRolloutCondition
func WithNodeRolloutConditionField(field string, matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj navarchosv1alpha1.NodeRolloutCondition) interface{} {
		r := reflect.ValueOf(obj)
		f := reflect.Indirect(r).FieldByName(field)
		return f.Interface()
	}, matcher)
}

// WithNodeReplacementStatusField gets the value of the named field from the
// NodeReplacements Status
func WithNodeReplacementStatusField(field string, matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj *navarchosv1alpha1.NodeReplacement) interface{} {
		r := reflect.ValueOf(obj.Status)
		f := reflect.Indirect(r).FieldByName(field)
		return f.Interface()
	}, matcher)
}

// WithNodeReplacementConditionField gets the value of the named field from the
// NodeReplacementCondition
func WithNodeReplacementConditionField(field string, matcher gtypes.GomegaMatcher) gtypes.GomegaMatcher {
	return gomega.WithTransform(func(obj navarchosv1alpha1.NodeReplacementCondition) interface{} {
		r := reflect.ValueOf(obj)
		f := reflect.Indirect(r).FieldByName(field)
		return f.Interface()
	}, matcher)
}
