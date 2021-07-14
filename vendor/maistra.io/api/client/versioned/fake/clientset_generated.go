// Copyright Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
	clientset "maistra.io/api/client/versioned"
	corev1 "maistra.io/api/client/versioned/typed/core/v1"
	fakecorev1 "maistra.io/api/client/versioned/typed/core/v1/fake"
	corev1alpha1 "maistra.io/api/client/versioned/typed/core/v1alpha1"
	fakecorev1alpha1 "maistra.io/api/client/versioned/typed/core/v1alpha1/fake"
	corev2 "maistra.io/api/client/versioned/typed/core/v2"
	fakecorev2 "maistra.io/api/client/versioned/typed/core/v2/fake"
	federationv1 "maistra.io/api/client/versioned/typed/federation/v1"
	fakefederationv1 "maistra.io/api/client/versioned/typed/federation/v1/fake"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	cs := &Clientset{tracker: o}
	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	cs.AddReactor("*", "*", testing.ObjectReaction(o))
	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	return cs
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
	tracker   testing.ObjectTracker
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

func (c *Clientset) Tracker() testing.ObjectTracker {
	return c.tracker
}

var _ clientset.Interface = &Clientset{}

// CoreV1 retrieves the CoreV1Client
func (c *Clientset) CoreV1() corev1.CoreV1Interface {
	return &fakecorev1.FakeCoreV1{Fake: &c.Fake}
}

// CoreV1alpha1 retrieves the CoreV1alpha1Client
func (c *Clientset) CoreV1alpha1() corev1alpha1.CoreV1alpha1Interface {
	return &fakecorev1alpha1.FakeCoreV1alpha1{Fake: &c.Fake}
}

// CoreV2 retrieves the CoreV2Client
func (c *Clientset) CoreV2() corev2.CoreV2Interface {
	return &fakecorev2.FakeCoreV2{Fake: &c.Fake}
}

// FederationV1 retrieves the FederationV1Client
func (c *Clientset) FederationV1() federationv1.FederationV1Interface {
	return &fakefederationv1.FakeFederationV1{Fake: &c.Fake}
}
