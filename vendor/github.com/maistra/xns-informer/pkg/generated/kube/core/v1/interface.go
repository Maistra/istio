/*
Copyright Red Hat, Inc.

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

// Code generated by xns-informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/maistra/xns-informer/pkg/generated/kube/internalinterfaces"
	informers "github.com/maistra/xns-informer/pkg/informers"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// ComponentStatuses returns a ComponentStatusInformer.
	ComponentStatuses() ComponentStatusInformer
	// ConfigMaps returns a ConfigMapInformer.
	ConfigMaps() ConfigMapInformer
	// Endpoints returns a EndpointsInformer.
	Endpoints() EndpointsInformer
	// Events returns a EventInformer.
	Events() EventInformer
	// LimitRanges returns a LimitRangeInformer.
	LimitRanges() LimitRangeInformer
	// Namespaces returns a NamespaceInformer.
	Namespaces() NamespaceInformer
	// Nodes returns a NodeInformer.
	Nodes() NodeInformer
	// PersistentVolumes returns a PersistentVolumeInformer.
	PersistentVolumes() PersistentVolumeInformer
	// PersistentVolumeClaims returns a PersistentVolumeClaimInformer.
	PersistentVolumeClaims() PersistentVolumeClaimInformer
	// Pods returns a PodInformer.
	Pods() PodInformer
	// PodTemplates returns a PodTemplateInformer.
	PodTemplates() PodTemplateInformer
	// ReplicationControllers returns a ReplicationControllerInformer.
	ReplicationControllers() ReplicationControllerInformer
	// ResourceQuotas returns a ResourceQuotaInformer.
	ResourceQuotas() ResourceQuotaInformer
	// Secrets returns a SecretInformer.
	Secrets() SecretInformer
	// Services returns a ServiceInformer.
	Services() ServiceInformer
	// ServiceAccounts returns a ServiceAccountInformer.
	ServiceAccounts() ServiceAccountInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespaces       informers.NamespaceSet
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespaces informers.NamespaceSet, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespaces: namespaces, tweakListOptions: tweakListOptions}
}

// ComponentStatuses returns a ComponentStatusInformer.
func (v *version) ComponentStatuses() ComponentStatusInformer {
	return &componentStatusInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// ConfigMaps returns a ConfigMapInformer.
func (v *version) ConfigMaps() ConfigMapInformer {
	return &configMapInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// Endpoints returns a EndpointsInformer.
func (v *version) Endpoints() EndpointsInformer {
	return &endpointsInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// Events returns a EventInformer.
func (v *version) Events() EventInformer {
	return &eventInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// LimitRanges returns a LimitRangeInformer.
func (v *version) LimitRanges() LimitRangeInformer {
	return &limitRangeInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// Namespaces returns a NamespaceInformer.
func (v *version) Namespaces() NamespaceInformer {
	return &namespaceInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// Nodes returns a NodeInformer.
func (v *version) Nodes() NodeInformer {
	return &nodeInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// PersistentVolumes returns a PersistentVolumeInformer.
func (v *version) PersistentVolumes() PersistentVolumeInformer {
	return &persistentVolumeInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// PersistentVolumeClaims returns a PersistentVolumeClaimInformer.
func (v *version) PersistentVolumeClaims() PersistentVolumeClaimInformer {
	return &persistentVolumeClaimInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// Pods returns a PodInformer.
func (v *version) Pods() PodInformer {
	return &podInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// PodTemplates returns a PodTemplateInformer.
func (v *version) PodTemplates() PodTemplateInformer {
	return &podTemplateInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// ReplicationControllers returns a ReplicationControllerInformer.
func (v *version) ReplicationControllers() ReplicationControllerInformer {
	return &replicationControllerInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// ResourceQuotas returns a ResourceQuotaInformer.
func (v *version) ResourceQuotas() ResourceQuotaInformer {
	return &resourceQuotaInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// Secrets returns a SecretInformer.
func (v *version) Secrets() SecretInformer {
	return &secretInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// Services returns a ServiceInformer.
func (v *version) Services() ServiceInformer {
	return &serviceInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}

// ServiceAccounts returns a ServiceAccountInformer.
func (v *version) ServiceAccounts() ServiceAccountInformer {
	return &serviceAccountInformer{factory: v.factory, namespaces: v.namespaces, tweakListOptions: v.tweakListOptions}
}
