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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	v1 "github.com/openshift/api/route/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/istio/pkg/kube"
	"istio.io/istio/pkg/servicemesh/controller"
)

const (
	maistraPrefix               = "maistra.io/"
	generatedByLabel            = maistraPrefix + "generated-by"
	generatedByValue            = "ior"
	originalHostAnnotation      = maistraPrefix + "original-host"
	gatewayNameLabel            = maistraPrefix + "gateway-name"
	gatewayNamespaceLabel       = maistraPrefix + "gateway-namespace"
	gatewayResourceVersionLabel = maistraPrefix + "gateway-resourceVersion"
)

type syncRoutes struct {
	gateway *networking.Gateway
	routes  []*v1.Route
}

// route manages the integration between Istio Gateways and OpenShift Routes
type route struct {
	pilotNamespace string
	routerClient   routev1.RouteV1Interface
	kubeClient     kubernetes.Interface
	store          model.ConfigStoreCache
	gatewaysMap    map[string]*syncRoutes
	gatewaysLock   sync.Mutex

	// memberroll functionality
	mrc           controller.MemberRollController
	namespaceLock sync.Mutex
	namespaces    []string
}

// NewRouterClient returns an OpenShift client for Routers
func NewRouterClient() (routev1.RouteV1Interface, error) {
	config, err := kube.BuildClientConfig("", "")
	if err != nil {
		return nil, err
	}

	client, err := routev1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// newRoute returns a new instance of Route object
func newRoute(
	kubeClient KubeClient,
	routerClient routev1.RouteV1Interface,
	store model.ConfigStoreCache,
	pilotNamespace string,
	mrc controller.MemberRollController) (*route, error) {

	if !kubeClient.IsRouteSupported() {
		return nil, fmt.Errorf("routes are not supported in this cluster")
	}

	r := &route{}

	r.kubeClient = kubeClient.GetActualClient()
	r.routerClient = routerClient
	r.pilotNamespace = pilotNamespace
	r.store = store
	r.mrc = mrc
	r.namespaces = []string{pilotNamespace}

	if err := r.initGateways(); err != nil {
		return nil, err
	}

	if r.mrc != nil {
		r.mrc.Register(r, "ior")
	}

	return r, nil
}

func (r *route) initGateways() error {
	r.gatewaysMap = make(map[string]*syncRoutes)

	configs, err := r.store.List(collections.IstioNetworkingV1Alpha3Gateways.Resource().GroupVersionKind(), model.NamespaceAll)
	if err != nil {
		return fmt.Errorf("could not get list of Gateways: %s", err)
	}
	for _, cfg := range configs {
		gw := cfg.Spec.(*networking.Gateway)
		IORLog.Debugf("Parsing Gateway %s/%s", cfg.Namespace, cfg.Name)
		r.gatewaysMap[gatewaysMapKey(cfg.Namespace, cfg.Name)] = &syncRoutes{
			gateway: gw,
		}

		// for _, server := range gw.Servers {
		// 	for _, host := range server.Hosts {
		// 	}
		// }
	}

	return r.syncNamespaces()
}

func (r *route) syncNamespaces() error {
	// r.namespaceLock.Lock()
	// defer r.namespaceLock.Unlock()

	var routes []v1.Route
	for _, ns := range r.namespaces {
		routeList, err := r.routerClient.Routes(ns).List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", generatedByLabel, generatedByValue),
		})
		if err != nil {
			return fmt.Errorf("could not get list of Routes in namespace %s: %s", ns, err)
		}
		routes = append(routes, routeList.Items...)
	}

	for _, route := range routes {
		IORLog.Debugf("Parsing Route %s/%s", route.Namespace, route.Name)
		syncRoute, ok := r.gatewaysMap[gatewaysMapKey(route.Labels[gatewayNamespaceLabel], route.Labels[gatewayNameLabel])]
		if ok {
			IORLog.Debugf("Found a Gateway (%s/%s) for this route", route.Labels[gatewayNamespaceLabel], route.Labels[gatewayNameLabel])
			syncRoute.routes = append(syncRoute.routes, &route)
			// TODO: Compare route's host with gateway hosts and edit/remove
		} else {
			IORLog.Infof("Found a route (%s/%s) created by IOR that does not have a matching Gateway. Removing it.", route.Namespace, route.Name)
			// TODO: Remove the route
		}
	}

	return nil
}

func gatewaysMapKey(namespace, name string) string {
	return namespace + "/" + name
}

func (r *route) SetNamespaces(namespaces ...string) {
	r.namespaceLock.Lock()
	defer r.namespaceLock.Unlock()

	IORLog.Debugf("SetNamespaces() called. Namespaces = %v", namespaces)
	r.namespaces = namespaces

	if err := r.syncNamespaces(); err != nil {
		IORLog.Error(err)
	}
}

func (r *route) syncGatewaysAndRoutes2(event model.Event, curr config.Config) error {
	r.namespaceLock.Lock()
	defer r.namespaceLock.Unlock()

	var result *multierror.Error

	switch event {
	case model.EventAdd:
		gw := curr.Spec.(*networking.Gateway)
		syncRoute := &syncRoutes{
			gateway: gw,
		}
		r.gatewaysMap[gatewaysMapKey(curr.Namespace, curr.Name)] = syncRoute

		for _, server := range gw.Servers {
			for _, host := range server.Hosts {
				route, err := r.createRoute(curr.Meta, gw, host, server.Tls)
				if err != nil {
					result = multierror.Append(result, err)
				} else {
					syncRoute.routes = append(syncRoute.routes, route)
				}
			}
		}

	case model.EventUpdate:
		//TODO

	case model.EventDelete:
		key := gatewaysMapKey(curr.Namespace, curr.Name)
		syncRoute, ok := r.gatewaysMap[key]
		if !ok {
			result = multierror.Append(result, fmt.Errorf("Could not find gateway %s/%s", curr.Namespace, curr.Name))
		}

		for _, route := range syncRoute.routes {
			result = multierror.Append(result, r.deleteRoute(route))
		}

		delete(r.gatewaysMap, key)
	}

	return result.ErrorOrNil()
}

func (r *route) syncGatewaysAndRoutes() error {
	r.namespaceLock.Lock()
	defer r.namespaceLock.Unlock()

	configs, err := r.store.List(collections.IstioNetworkingV1Alpha3Gateways.Resource().GroupVersionKind(), model.NamespaceAll)
	if err != nil {
		return fmt.Errorf("could not get list of Gateways: %s", err)
	}

	var routes []v1.Route

	for _, ns := range r.namespaces {
		routeList, err := r.routerClient.Routes(ns).List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", generatedByLabel, generatedByValue),
		})
		if err != nil {
			return fmt.Errorf("could not get list of Routes in namespace %s: %s", ns, err)
		}
		routes = append(routes, routeList.Items...)
	}

	var result *multierror.Error
	routesMap := make(map[string]*v1.Route, len(routes))
	for _, route := range routes {
		_, err := findConfig(configs, route.Labels[gatewayNameLabel], route.Labels[gatewayNamespaceLabel], route.Labels[gatewayResourceVersionLabel])
		if err != nil {
			result = multierror.Append(r.deleteRoute(&route))
		} else {
			routesMap[getHost(route)] = &route
		}
	}

	for _, cfg := range configs {
		gateway := cfg.Spec.(*networking.Gateway)
		IORLog.Debugf("Found Gateway: %s/%s", cfg.Namespace, cfg.Name)

		for _, server := range gateway.Servers {
			for _, host := range server.Hosts {
				_, ok := routesMap[host]
				if !ok {
					_, err := r.createRoute(cfg.Meta, gateway, host, server.Tls)
					result = multierror.Append(err)
				}

			}
		}
	}

	return result.ErrorOrNil()
}

func getHost(route v1.Route) string {
	if host := route.ObjectMeta.Annotations[originalHostAnnotation]; host != "" {
		return host
	}
	return route.Spec.Host
}

func (r *route) deleteRoute(route *v1.Route) error {
	var immediate int64
	host := getHost(*route)
	err := r.routerClient.Routes(route.Namespace).Delete(context.TODO(), route.ObjectMeta.Name, metav1.DeleteOptions{GracePeriodSeconds: &immediate})
	if err != nil {
		return fmt.Errorf("error deleting route %s/%s: %s", route.ObjectMeta.Namespace, route.ObjectMeta.Name, err)
	}

	IORLog.Infof("Deleted route %s/%s (gateway hostname: %s)", route.ObjectMeta.Namespace, route.ObjectMeta.Name, host)
	return nil
}

// must be called with lock held
func (r *route) createRoute(metadata config.Meta, gateway *networking.Gateway, originalHost string, tls *networking.ServerTLSSettings) (*v1.Route, error) {
	var wildcard = v1.WildcardPolicyNone
	actualHost := originalHost

	IORLog.Debugf("Creating route for hostname %s", originalHost)

	if originalHost == "*" {
		IORLog.Warnf("Gateway %s/%s: Hostname * is not supported at the moment. Letting OpenShift create it instead.", metadata.Namespace, metadata.Name)
		actualHost = ""
	} else if strings.HasPrefix(originalHost, "*.") {
		// FIXME: Update link below to version 4.5 when it's out
		// Wildcards are not enabled by default in OCP 3.x.
		// See https://docs.openshift.com/container-platform/3.11/install_config/router/default_haproxy_router.html#using-wildcard-routes
		// FIXME(2): Is there a way to check if OCP supports wildcard and print out a warning if not?
		wildcard = v1.WildcardPolicySubdomain
		actualHost = "wildcard." + strings.TrimPrefix(originalHost, "*.")
	}

	var tlsConfig *v1.TLSConfig
	targetPort := "http2"
	if tls != nil {
		tlsConfig = &v1.TLSConfig{Termination: v1.TLSTerminationPassthrough}
		targetPort = "https"
		if tls.HttpsRedirect {
			tlsConfig.InsecureEdgeTerminationPolicy = v1.InsecureEdgeTerminationPolicyRedirect
		}
	}

	serviceNamespace, serviceName, err := r.findService(gateway)
	if err != nil {
		return nil, err
	}

	annotations := map[string]string{
		originalHostAnnotation: originalHost,
	}
	for keyName, keyValue := range metadata.Annotations {
		if !strings.HasPrefix(keyName, "kubectl.kubernetes.io") {
			annotations[keyName] = keyValue
		}
	}

	nr, err := r.routerClient.Routes(serviceNamespace).Create(context.TODO(), &v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s-%s", metadata.Namespace, metadata.Name, hostHash(actualHost)),
			Namespace: serviceNamespace,
			Labels: map[string]string{
				generatedByLabel:            generatedByValue,
				gatewayNamespaceLabel:       metadata.Namespace,
				gatewayNameLabel:            metadata.Name,
				gatewayResourceVersionLabel: metadata.ResourceVersion,
			},
			Annotations: annotations,
		},
		Spec: v1.RouteSpec{
			Host: actualHost,
			Port: &v1.RoutePort{
				TargetPort: intstr.IntOrString{
					Type:   intstr.String,
					StrVal: targetPort,
				},
			},
			To: v1.RouteTargetReference{
				Name: serviceName,
			},
			TLS:            tlsConfig,
			WildcardPolicy: wildcard,
		},
	}, metav1.CreateOptions{})

	if err != nil {
		return nil, fmt.Errorf("error creating a route for the host %s (gateway: %s/%s): %s", originalHost, metadata.Namespace, metadata.Name, err)
	}

	// IORLog.Infof("Created route %s/%s for hostname %s (gateway: %s/%s)",
	// 	nr.ObjectMeta.Namespace, nr.ObjectMeta.Name,
	// 	nr.Spec.Host,
	// 	metadata.Namespace, metadata.Name)

	return nr, nil
}

// findService tries to find a service that matches with the given gateway selector
// Returns the namespace and service name that is a match, or an error
// must be called with lock held
func (r *route) findService(gateway *networking.Gateway) (string, string, error) {
	gwSelector := labels.SelectorFromSet(gateway.Selector)

	for _, ns := range r.namespaces {
		// Get the list of pods that match the gateway selector
		podList, err := r.kubeClient.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: gwSelector.String()})
		if err != nil {
			return "", "", fmt.Errorf("could not get the list of pods in namespace %s: %v", ns, err)
		}

		// Get the list of services in this namespace
		svcList, err := r.kubeClient.CoreV1().Services(ns).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return "", "", fmt.Errorf("could not get the list of services in namespace %s: %v", ns, err)
		}

		// Look for a service whose selector matches the pod labels
		for _, pod := range podList.Items {
			podLabels := labels.Set(pod.ObjectMeta.Labels)

			for _, svc := range svcList.Items {
				svcSelector := labels.SelectorFromSet(svc.Spec.Selector)
				if svcSelector.Matches(podLabels) {
					return ns, svc.Name, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("could not find a service that matches the gateway selector `%s'. Namespaces where we looked at: %v",
		gwSelector.String(), r.namespaces)
}

func findConfig(list []config.Config, name, namespace, resourceVersion string) (config.Config, error) {
	for _, item := range list {
		if item.Name == name && item.Namespace == namespace && item.ResourceVersion == resourceVersion {
			return item, nil
		}
	}
	return config.Config{}, fmt.Errorf("config not found")
}

// hostHash applies a sha256 on the host and truncate it to the first 8 bytes
// This gives enough uniqueness for a given hostname
func hostHash(name string) string {
	if name == "" {
		name = "star"
	}

	hash := sha256.Sum256([]byte(name))
	return hex.EncodeToString(hash[:8])
}
