// Copyright Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ior

import (
	"fmt"
	"sync"

	v1 "github.com/openshift/api/route/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// FakeRouter implements routev1.RouteInterface
type FakeRouter struct {
	routes     map[string]*v1.Route
	routesLock sync.Mutex
}

// FakeRouterClient implements routev1.RouteV1Interface
type FakeRouterClient struct {
	routesByNamespace     map[string]routev1.RouteInterface
	routesByNamespaceLock sync.Mutex
}

type fakeKubeClient struct {
	client kubernetes.Interface
}

// NewFakeKubeClient creates a new FakeKubeClient
func NewFakeKubeClient(client kubernetes.Interface) KubeClient {
	return &fakeKubeClient{client: client}
}

func (c *fakeKubeClient) IsRouteSupported() bool {
	return true
}

func (c *fakeKubeClient) GetActualClient() kubernetes.Interface {
	return c.client
}

// NewFakeRouterClient creates a new FakeRouterClient
func NewFakeRouterClient() routev1.RouteV1Interface {
	return &FakeRouterClient{
		routesByNamespace: make(map[string]routev1.RouteInterface),
	}
}

// NewFakeRouter creates a new FakeRouter
func NewFakeRouter() routev1.RouteInterface {
	return &FakeRouter{
		routes: make(map[string]*v1.Route),
	}
}

// RESTClient implements routev1.RouteV1Interface
func (rc *FakeRouterClient) RESTClient() rest.Interface {
	panic("not implemented")
}

// Routes implements routev1.RouteV1Interface
func (rc *FakeRouterClient) Routes(namespace string) routev1.RouteInterface {
	rc.routesByNamespaceLock.Lock()
	defer rc.routesByNamespaceLock.Unlock()

	if _, ok := rc.routesByNamespace[namespace]; !ok {
		rc.routesByNamespace[namespace] = NewFakeRouter()
	}
	return rc.routesByNamespace[namespace]
}

var generatedHostNumber int

// Create implements routev1.RouteInterface
func (fk *FakeRouter) Create(ctx context.Context, route *v1.Route, opts metav1.CreateOptions) (*v1.Route, error) {
	fk.routesLock.Lock()
	defer fk.routesLock.Unlock()

	if route.Spec.Host == "" {
		generatedHostNumber++
		route.Spec.Host = fmt.Sprintf("generated-host%d.com", generatedHostNumber)
	}

	fk.routes[route.Name] = route
	return route, nil
}

// Update implements routev1.RouteInterface
func (fk *FakeRouter) Update(ctx context.Context, route *v1.Route, opts metav1.UpdateOptions) (*v1.Route, error) {
	panic("not implemented")
}

// UpdateStatus implements routev1.RouteInterface
func (fk *FakeRouter) UpdateStatus(ctx context.Context, route *v1.Route, opts metav1.UpdateOptions) (*v1.Route, error) {
	panic("not implemented")
}

// Delete implements routev1.RouteInterface
func (fk *FakeRouter) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	fk.routesLock.Lock()
	defer fk.routesLock.Unlock()

	if _, ok := fk.routes[name]; !ok {
		return fmt.Errorf("route %s not found", name)
	}

	delete(fk.routes, name)
	return nil
}

// DeleteCollection implements routev1.RouteInterface
func (fk *FakeRouter) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	panic("not implemented")
}

// Get implements routev1.RouteInterface
func (fk *FakeRouter) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Route, error) {
	panic("not implemented")
}

// List implements routev1.RouteInterface
func (fk *FakeRouter) List(ctx context.Context, opts metav1.ListOptions) (*v1.RouteList, error) {
	fk.routesLock.Lock()
	defer fk.routesLock.Unlock()

	var items []v1.Route
	for _, route := range fk.routes {
		items = append(items, *route)
	}
	result := &v1.RouteList{Items: items}

	return result, nil
}

// Watch Create implements routev1.RouteInterface
func (fk *FakeRouter) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("not implemented")
}

// Patch implements routev1.RouteInterface
func (fk *FakeRouter) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions,
	subresources ...string) (result *v1.Route, err error) {
	panic("not implemented")
}
