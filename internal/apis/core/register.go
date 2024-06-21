// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Package core contains API Schema definitions for the apinet core API group
// +groupName=core.apinet.ironcore.dev
package core

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the name of the apinet group.
const GroupName = "core.apinet.ironcore.dev"

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion,
		&DaemonSet{},
		&DaemonSetList{},
		&Instance{},
		&InstanceList{},
		&IP{},
		&IPList{},
		&IPAddress{},
		&IPAddressList{},
		&LoadBalancer{},
		&LoadBalancerList{},
		&LoadBalancerRouting{},
		&LoadBalancerRoutingList{},
		&NATGateway{},
		&NATGatewayList{},
		&NATGatewayAutoscaler{},
		&NATGatewayAutoscalerList{},
		&NATTable{},
		&NATTableList{},
		&Network{},
		&NetworkList{},
		&NetworkID{},
		&NetworkIDList{},
		&NetworkInterface{},
		&NetworkInterfaceList{},
		&NetworkPolicy{},
		&NetworkPolicyList{},
		&NetworkPolicyRule{},
		&NetworkPolicyRuleList{},
		&Node{},
		&NodeList{},
	)
	return nil
}
