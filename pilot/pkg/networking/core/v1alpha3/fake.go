// Copyright Istio Authors
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

package v1alpha3

import (
	"bytes"
	"errors"
	"sync"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"

	meshconfig "istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pilot/pkg/config/kube/crd"
	"istio.io/istio/pilot/pkg/config/memory"
	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/networking/plugin"
	"istio.io/istio/pilot/pkg/networking/plugin/registry"
	"istio.io/istio/pilot/pkg/serviceregistry"
	"istio.io/istio/pilot/pkg/serviceregistry/aggregate"
	memregistry "istio.io/istio/pilot/pkg/serviceregistry/memory"
	"istio.io/istio/pilot/pkg/serviceregistry/serviceentry"
	"istio.io/istio/pilot/test/xdstest"
	"istio.io/istio/pkg/config/mesh"
	"istio.io/istio/pkg/config/schema/collections"
	"istio.io/istio/pkg/test"
	"istio.io/istio/pkg/test/util/retry"
)

type TestOptions struct {
	// If provided, these configs will be used directly
	Configs        []model.Config
	ConfigPointers []*model.Config

	// If provided, the yaml string will be parsed and used as configs
	ConfigString string
	// If provided, the ConfigString will be treated as a go template, with this as input params
	ConfigTemplateInput interface{}

	// Services to pre-populate as part of the service discovery
	Services  []*model.Service
	Instances []*model.ServiceInstance

	// If provided, this mesh config will be used
	MeshConfig      *meshconfig.MeshConfig
	NetworksWatcher mesh.NetworksWatcher

	// Additional service registries to use. A ServiceEntry and memory registry will always be created.
	ServiceRegistries []serviceregistry.Instance

	// ConfigGen plugins to use. If not set, all default plugins will be used
	Plugins []plugin.Plugin

	// Mutex used for push context access. Should generally only be used by NewFakeDiscoveryServer
	PushContextLock *sync.RWMutex
}

type ConfigGenTest struct {
	t                    test.Failer
	pushContextLock      *sync.RWMutex
	store                model.ConfigStoreCache
	env                  *model.Environment
	ConfigGen            *ConfigGeneratorImpl
	MemRegistry          *memregistry.ServiceDiscovery
	ServiceEntryRegistry *serviceentry.ServiceEntryStore
}

func NewConfigGenTest(t test.Failer, opts TestOptions) *ConfigGenTest {
	t.Helper()
	stop := make(chan struct{})
	t.Cleanup(func() {
		close(stop)
	})

	configs := getConfigs(t, opts)
	configStore := memory.MakeWithLedger(collections.Pilot, &model.DisabledLedger{}, true)

	configController := memory.NewSyncController(configStore)
	go configController.Run(stop)

	m := opts.MeshConfig
	if m == nil {
		def := mesh.DefaultMeshConfig()
		m = &def
	}

	serviceDiscovery := aggregate.NewController(aggregate.Options{})
	se := serviceentry.NewServiceDiscovery(configController, model.MakeIstioStore(configStore), &FakeXdsUpdater{})
	// TODO allow passing in registry, for k8s, mem reigstry
	serviceDiscovery.AddRegistry(se)
	msd := memregistry.NewServiceDiscovery(opts.Services)
	for _, instance := range opts.Instances {
		msd.AddInstance(instance.Service.Hostname, instance)
	}
	msd.ClusterID = string(serviceregistry.Mock)
	serviceDiscovery.AddRegistry(serviceregistry.Simple{
		ClusterID:        string(serviceregistry.Mock),
		ProviderID:       serviceregistry.Mock,
		ServiceDiscovery: msd,
		Controller:       msd.Controller,
	})
	for _, reg := range opts.ServiceRegistries {
		serviceDiscovery.AddRegistry(reg)
	}

	env := &model.Environment{}
	env.PushContext = model.NewPushContext()
	env.ServiceDiscovery = serviceDiscovery
	env.IstioConfigStore = model.MakeIstioStore(configStore)
	env.Watcher = mesh.NewFixedWatcher(m)
	if opts.NetworksWatcher == nil {
		opts.NetworksWatcher = mesh.NewFixedNetworksWatcher(nil)
	}
	env.NetworksWatcher = opts.NetworksWatcher

	// Setup configuration. This should be done after registries are added so they can process events.
	for _, cfg := range configs {
		if _, err := configStore.Create(cfg); err != nil {
			t.Fatalf("failed to create config %v: %v", cfg.Name, err)
		}
	}

	// TODO allow passing event handlers for controller

	retry.UntilSuccessOrFail(t, func() error {
		if !serviceDiscovery.HasSynced() {
			return errors.New("not synced")
		}
		return nil
	})

	se.ResyncEDS()
	if err := env.PushContext.InitContext(env, nil, nil); err != nil {
		t.Fatalf("Failed to initialize push context: %v", err)
	}

	if opts.Plugins == nil {
		opts.Plugins = registry.NewPlugins([]string{plugin.Authn, plugin.Authz})
	}

	fake := &ConfigGenTest{
		t:                    t,
		store:                configController,
		env:                  env,
		ConfigGen:            NewConfigGenerator(opts.Plugins),
		MemRegistry:          msd,
		ServiceEntryRegistry: se,
		pushContextLock:      opts.PushContextLock,
	}
	return fake
}

// SetupProxy initializes a proxy for the current environment. This should generally be used when creating
// any proxy. For example, `p := SetupProxy(&model.Proxy{...})`.
func (f *ConfigGenTest) SetupProxy(p *model.Proxy) *model.Proxy {
	// Setup defaults
	if p == nil {
		p = &model.Proxy{}
	}
	if p.Metadata == nil {
		p.Metadata = &model.NodeMetadata{}
	}
	if p.Metadata.IstioVersion == "" {
		p.Metadata.IstioVersion = "1.8.0"
		p.IstioVersion = model.ParseIstioVersion(p.Metadata.IstioVersion)
	}
	if p.Type == "" {
		p.Type = model.SidecarProxy
	}
	if p.ConfigNamespace == "" {
		p.ConfigNamespace = "default"
	}
	if p.ID == "" {
		p.ID = "app.test"
	}
	if len(p.IPAddresses) == 0 {
		p.IPAddresses = []string{"1.1.1.1"}
	}

	// Initialize data structures
	pc := f.PushContext()
	p.SetSidecarScope(pc)
	p.SetGatewaysForProxy(pc)
	if err := p.SetServiceInstances(f.env.ServiceDiscovery); err != nil {
		f.t.Fatal(err)
	}
	p.DiscoverIPVersions()
	return p
}

// TODO do we need lock around push context?
func (f *ConfigGenTest) Listeners(p *model.Proxy) []*listener.Listener {
	return f.ConfigGen.BuildListeners(p, f.PushContext())
}

func (f *ConfigGenTest) Clusters(p *model.Proxy) []*cluster.Cluster {
	return f.ConfigGen.BuildClusters(p, f.PushContext())
}

func (f *ConfigGenTest) Routes(p *model.Proxy) []*route.RouteConfiguration {
	return f.ConfigGen.BuildHTTPRoutes(p, f.PushContext(), xdstest.ExtractRoutesFromListeners(f.Listeners(p)))
}

func (f *ConfigGenTest) PushContext() *model.PushContext {
	if f.pushContextLock != nil {
		f.pushContextLock.RLock()
		defer f.pushContextLock.RUnlock()
	}
	return f.env.PushContext
}

func (f *ConfigGenTest) Env() *model.Environment {
	return f.env
}

func (f *ConfigGenTest) Store() model.ConfigStoreCache {
	return f.store
}

var _ model.XDSUpdater = &FakeXdsUpdater{}

func getConfigs(t test.Failer, opts TestOptions) []model.Config {
	for _, p := range opts.ConfigPointers {
		if p != nil {
			opts.Configs = append(opts.Configs, *p)
		}
	}
	if len(opts.Configs) > 0 {
		return opts.Configs
	}
	configStr := opts.ConfigString
	if opts.ConfigTemplateInput != nil {
		tmpl := template.Must(template.New("").Funcs(sprig.TxtFuncMap()).Parse(opts.ConfigString))
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, opts.ConfigTemplateInput); err != nil {
			t.Fatalf("failed to execute template: %v", err)
		}
		configStr = buf.String()
	}
	configs, _, err := crd.ParseInputs(configStr)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	// setup default namespace if not defined
	for i, c := range configs {
		if c.Namespace == "" {
			c.Namespace = "default"
		}
		configs[i] = c
	}
	return configs
}

type FakeXdsUpdater struct{}

func (f *FakeXdsUpdater) ConfigUpdate(*model.PushRequest) {}

func (f *FakeXdsUpdater) EDSUpdate(_, _, _ string, _ []*model.IstioEndpoint) {}

func (f *FakeXdsUpdater) EDSCacheUpdate(_, _, _ string, _ []*model.IstioEndpoint) {}

func (f *FakeXdsUpdater) SvcUpdate(_, _, _ string, _ model.Event) {}

func (f *FakeXdsUpdater) ProxyUpdate(_, _ string) {}
