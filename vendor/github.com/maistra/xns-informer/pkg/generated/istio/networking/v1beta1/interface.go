// Code generated by xns-informer-gen. DO NOT EDIT.

package v1beta1

import (
	xnsinformers "github.com/maistra/xns-informer/pkg/informers"
	informers "istio.io/client-go/pkg/informers/externalversions/networking/v1beta1"
)

type Interface interface {
	DestinationRules() informers.DestinationRuleInformer
	Gateways() informers.GatewayInformer
	ServiceEntries() informers.ServiceEntryInformer
	Sidecars() informers.SidecarInformer
	VirtualServices() informers.VirtualServiceInformer
	WorkloadEntries() informers.WorkloadEntryInformer
}

type version struct {
	factory xnsinformers.SharedInformerFactory
}

func New(factory xnsinformers.SharedInformerFactory) Interface {
	return &version{factory: factory}
}
func (v *version) DestinationRules() informers.DestinationRuleInformer {
	return NewDestinationRuleInformer(v.factory)
}
func (v *version) Gateways() informers.GatewayInformer {
	return NewGatewayInformer(v.factory)
}
func (v *version) ServiceEntries() informers.ServiceEntryInformer {
	return NewServiceEntryInformer(v.factory)
}
func (v *version) Sidecars() informers.SidecarInformer {
	return NewSidecarInformer(v.factory)
}
func (v *version) VirtualServices() informers.VirtualServiceInformer {
	return NewVirtualServiceInformer(v.factory)
}
func (v *version) WorkloadEntries() informers.WorkloadEntryInformer {
	return NewWorkloadEntryInformer(v.factory)
}
