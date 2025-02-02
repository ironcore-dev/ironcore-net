// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"

	v1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/core/v1alpha1"
	scheme "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// NetworkIDsGetter has a method to return a NetworkIDInterface.
// A group's client should implement this interface.
type NetworkIDsGetter interface {
	NetworkIDs() NetworkIDInterface
}

// NetworkIDInterface has methods to work with NetworkID resources.
type NetworkIDInterface interface {
	Create(ctx context.Context, networkID *v1alpha1.NetworkID, opts v1.CreateOptions) (*v1alpha1.NetworkID, error)
	Update(ctx context.Context, networkID *v1alpha1.NetworkID, opts v1.UpdateOptions) (*v1alpha1.NetworkID, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.NetworkID, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.NetworkIDList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NetworkID, err error)
	Apply(ctx context.Context, networkID *corev1alpha1.NetworkIDApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NetworkID, err error)
	NetworkIDExpansion
}

// networkIDs implements NetworkIDInterface
type networkIDs struct {
	*gentype.ClientWithListAndApply[*v1alpha1.NetworkID, *v1alpha1.NetworkIDList, *corev1alpha1.NetworkIDApplyConfiguration]
}

// newNetworkIDs returns a NetworkIDs
func newNetworkIDs(c *CoreV1alpha1Client) *networkIDs {
	return &networkIDs{
		gentype.NewClientWithListAndApply[*v1alpha1.NetworkID, *v1alpha1.NetworkIDList, *corev1alpha1.NetworkIDApplyConfiguration](
			"networkids",
			c.RESTClient(),
			scheme.ParameterCodec,
			"",
			func() *v1alpha1.NetworkID { return &v1alpha1.NetworkID{} },
			func() *v1alpha1.NetworkIDList { return &v1alpha1.NetworkIDList{} }),
	}
}
