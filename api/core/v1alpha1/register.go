// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Package v1alpha1 contains API Schema definitions for the apinet v1alpha1 API group
// +groupName=core.apinet.ironcore.dev
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupName is the name of the apinet group.
const GroupName = "core.apinet.ironcore.dev"

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}

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
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
