// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned/typed/core/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeCoreV1alpha1 struct {
	*testing.Fake
}

func (c *FakeCoreV1alpha1) DaemonSets(namespace string) v1alpha1.DaemonSetInterface {
	return newFakeDaemonSets(c, namespace)
}

func (c *FakeCoreV1alpha1) IPs(namespace string) v1alpha1.IPInterface {
	return newFakeIPs(c, namespace)
}

func (c *FakeCoreV1alpha1) IPAddresses() v1alpha1.IPAddressInterface {
	return newFakeIPAddresses(c)
}

func (c *FakeCoreV1alpha1) Instances(namespace string) v1alpha1.InstanceInterface {
	return newFakeInstances(c, namespace)
}

func (c *FakeCoreV1alpha1) LoadBalancers(namespace string) v1alpha1.LoadBalancerInterface {
	return newFakeLoadBalancers(c, namespace)
}

func (c *FakeCoreV1alpha1) LoadBalancerRoutings(namespace string) v1alpha1.LoadBalancerRoutingInterface {
	return newFakeLoadBalancerRoutings(c, namespace)
}

func (c *FakeCoreV1alpha1) NATGateways(namespace string) v1alpha1.NATGatewayInterface {
	return newFakeNATGateways(c, namespace)
}

func (c *FakeCoreV1alpha1) NATGatewayAutoscalers(namespace string) v1alpha1.NATGatewayAutoscalerInterface {
	return newFakeNATGatewayAutoscalers(c, namespace)
}

func (c *FakeCoreV1alpha1) NATTables(namespace string) v1alpha1.NATTableInterface {
	return newFakeNATTables(c, namespace)
}

func (c *FakeCoreV1alpha1) Networks(namespace string) v1alpha1.NetworkInterface {
	return newFakeNetworks(c, namespace)
}

func (c *FakeCoreV1alpha1) NetworkIDs() v1alpha1.NetworkIDInterface {
	return newFakeNetworkIDs(c)
}

func (c *FakeCoreV1alpha1) NetworkInterfaces(namespace string) v1alpha1.NetworkInterfaceInterface {
	return newFakeNetworkInterfaces(c, namespace)
}

func (c *FakeCoreV1alpha1) NetworkPolicies(namespace string) v1alpha1.NetworkPolicyInterface {
	return newFakeNetworkPolicies(c, namespace)
}

func (c *FakeCoreV1alpha1) NetworkPolicyRules(namespace string) v1alpha1.NetworkPolicyRuleInterface {
	return newFakeNetworkPolicyRules(c, namespace)
}

func (c *FakeCoreV1alpha1) Nodes() v1alpha1.NodeInterface {
	return newFakeNodes(c)
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeCoreV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
