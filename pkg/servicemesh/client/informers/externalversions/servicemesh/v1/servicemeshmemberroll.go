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

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	"context"
	time "time"

	servicemeshv1 "istio.io/istio/pkg/servicemesh/apis/servicemesh/v1"
	versioned "istio.io/istio/pkg/servicemesh/client/clientset/versioned"
	internalinterfaces "istio.io/istio/pkg/servicemesh/client/informers/externalversions/internalinterfaces"
	v1 "istio.io/istio/pkg/servicemesh/client/listers/servicemesh/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ServiceMeshMemberRollInformer provides access to a shared informer and lister for
// ServiceMeshMemberRolls.
type ServiceMeshMemberRollInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ServiceMeshMemberRollLister
}

type serviceMeshMemberRollInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewServiceMeshMemberRollInformer constructs a new informer for ServiceMeshMemberRoll type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewServiceMeshMemberRollInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredServiceMeshMemberRollInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredServiceMeshMemberRollInformer constructs a new informer for ServiceMeshMemberRoll type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredServiceMeshMemberRollInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MaistraV1().ServiceMeshMemberRolls(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MaistraV1().ServiceMeshMemberRolls(namespace).Watch(context.TODO(), options)
			},
		},
		&servicemeshv1.ServiceMeshMemberRoll{},
		resyncPeriod,
		indexers,
	)
}

func (f *serviceMeshMemberRollInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredServiceMeshMemberRollInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *serviceMeshMemberRollInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&servicemeshv1.ServiceMeshMemberRoll{}, f.defaultInformer)
}

func (f *serviceMeshMemberRollInformer) Lister() v1.ServiceMeshMemberRollLister {
	return v1.NewServiceMeshMemberRollLister(f.Informer().GetIndexer())
}
