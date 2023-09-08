// Copyright 2022 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package apiserver

import (
	"net/netip"

	"github.com/onmetal/onmetal-api-net/apimachinery/equality"
	"github.com/onmetal/onmetal-api-net/client-go/informers"
	v1alpha1client "github.com/onmetal/onmetal-api-net/client-go/onmetalapinet/typed/core/v1alpha1"
	"github.com/onmetal/onmetal-api-net/internal/apis/core"
	"github.com/onmetal/onmetal-api-net/internal/apis/core/install"
	"github.com/onmetal/onmetal-api-net/internal/registry/daemonset"
	"github.com/onmetal/onmetal-api-net/internal/registry/instance"
	"github.com/onmetal/onmetal-api-net/internal/registry/ip"
	"github.com/onmetal/onmetal-api-net/internal/registry/ip/ipaddressallocator"
	"github.com/onmetal/onmetal-api-net/internal/registry/ipaddress"
	"github.com/onmetal/onmetal-api-net/internal/registry/ipallocator"
	"github.com/onmetal/onmetal-api-net/internal/registry/loadbalancer"
	"github.com/onmetal/onmetal-api-net/internal/registry/loadbalancerrouting"
	"github.com/onmetal/onmetal-api-net/internal/registry/natgateway"
	"github.com/onmetal/onmetal-api-net/internal/registry/natgatewayautoscaler"
	"github.com/onmetal/onmetal-api-net/internal/registry/nattable"
	"github.com/onmetal/onmetal-api-net/internal/registry/network"
	"github.com/onmetal/onmetal-api-net/internal/registry/network/networkidallocator"
	"github.com/onmetal/onmetal-api-net/internal/registry/networkid"
	"github.com/onmetal/onmetal-api-net/internal/registry/networkinterface"
	"github.com/onmetal/onmetal-api-net/internal/registry/node"
	corev1 "k8s.io/api/core/v1"
	apimachineryequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

func init() {
	utilruntime.Must(equality.AddFuncs(apimachineryequality.Semantic))

	install.Install(Scheme)

	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

// ExtraConfig holds custom apiserver config
type ExtraConfig struct {
	MinVNI int32
	MaxVNI int32

	PublicPrefix []netip.Prefix

	VersionedInformers informers.SharedInformerFactory
}

// Config defines the config for the apiserver
type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// OnmetalAPIServer contains state for a Kubernetes cluster master/api server.
type OnmetalAPIServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of OnmetalAPIServer from the given config.
func (c completedConfig) New() (*OnmetalAPIServer, error) {
	genericServer, err := c.GenericConfig.New("onmetal-api-net-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	v1alpha1Client, err := v1alpha1client.NewForConfig(c.GenericConfig.LoopbackClientConfig)
	if err != nil {
		return nil, err
	}

	ipAddrAllocByFamily := make(map[corev1.IPFamily]ipaddressallocator.Interface)
	ipAllocByFamily := make(map[corev1.IPFamily]ipallocator.Interface)

	for _, publicPrefix := range c.ExtraConfig.PublicPrefix {
		ipAddrAlloc, err := ipaddressallocator.New(
			publicPrefix,
			v1alpha1Client,
			c.ExtraConfig.VersionedInformers.Core().V1alpha1().IPAddresses(),
		)
		if err != nil {
			return nil, err
		}

		ipAlloc, err := ipallocator.New(
			publicPrefix,
			v1alpha1Client,
			c.ExtraConfig.VersionedInformers.Core().V1alpha1().IPs(),
		)
		if err != nil {
			return nil, err
		}

		ipAddrAllocByFamily[ipAddrAlloc.IPFamily()] = ipAddrAlloc
		ipAllocByFamily[ipAlloc.IPFamily()] = ipAlloc
	}

	networkIDAllocator, err := networkidallocator.NewNetworkIDAllocator(
		v1alpha1Client,
		c.ExtraConfig.VersionedInformers.Core().V1alpha1().NetworkIDs(),
		c.ExtraConfig.MinVNI,
		c.ExtraConfig.MaxVNI,
	)
	if err != nil {
		return nil, err
	}

	s := &OnmetalAPIServer{
		GenericAPIServer: genericServer,
	}

	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(core.GroupName, Scheme, metav1.ParameterCodec, Codecs)
	v1alpha1storage := make(map[string]rest.Storage)

	daemonSetStorage, err := daemonset.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["daemonsets"] = daemonSetStorage.DaemonSet
	v1alpha1storage["daemonsets/status"] = daemonSetStorage.Status

	instanceStorage, err := instance.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["instances"] = instanceStorage.Instance
	v1alpha1storage["instances/status"] = instanceStorage.Status

	ipStorage, err := ip.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter, ipAddrAllocByFamily)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["ips"] = ipStorage.IP

	ipAddressStorage, err := ipaddress.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["ipaddresses"] = ipAddressStorage.IPAddress

	loadBalancerStorage, err := loadbalancer.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter, ipAllocByFamily)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["loadbalancers"] = loadBalancerStorage.LoadBalancer
	v1alpha1storage["loadbalancers/status"] = loadBalancerStorage.Status

	loadBalancerRoutingStorage, err := loadbalancerrouting.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["loadbalancerroutings"] = loadBalancerRoutingStorage.LoadBalancerRouting

	natGatewayStorage, err := natgateway.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter, ipAllocByFamily)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["natgateways"] = natGatewayStorage.NATGateway
	v1alpha1storage["natgateways/status"] = natGatewayStorage.Status

	natGatewayAutoscalerStorage, err := natgatewayautoscaler.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["natgatewayautoscalers"] = natGatewayAutoscalerStorage.NATGatewayAutoscaler
	v1alpha1storage["natgatewayautoscalers/status"] = natGatewayAutoscalerStorage.Status

	natTableStorage, err := nattable.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["nattables"] = natTableStorage.NATTable

	networkStorage, err := network.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter, networkIDAllocator)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["networks"] = networkStorage.Network
	v1alpha1storage["networks/status"] = networkStorage.Status

	networkIDStorage, err := networkid.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["networkids"] = networkIDStorage.NetworkID

	networkInterfaceStorage, err := networkinterface.NewStorage(
		Scheme,
		c.GenericConfig.RESTOptionsGetter,
		ipAllocByFamily,
	)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["networkinterfaces"] = networkInterfaceStorage.NetworkInterface
	v1alpha1storage["networkinterfaces/status"] = networkInterfaceStorage.Status

	nodeStorage, err := node.NewStorage(Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	v1alpha1storage["nodes"] = nodeStorage.Node
	v1alpha1storage["nodes/status"] = nodeStorage.Status

	apiGroupInfo.VersionedResourcesStorageMap["v1alpha1"] = v1alpha1storage

	if err := s.GenericAPIServer.InstallAPIGroups(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}
