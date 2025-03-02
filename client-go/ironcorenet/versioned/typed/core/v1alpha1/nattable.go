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

// NATTablesGetter has a method to return a NATTableInterface.
// A group's client should implement this interface.
type NATTablesGetter interface {
	NATTables(namespace string) NATTableInterface
}

// NATTableInterface has methods to work with NATTable resources.
type NATTableInterface interface {
	Create(ctx context.Context, nATTable *v1alpha1.NATTable, opts v1.CreateOptions) (*v1alpha1.NATTable, error)
	Update(ctx context.Context, nATTable *v1alpha1.NATTable, opts v1.UpdateOptions) (*v1alpha1.NATTable, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.NATTable, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.NATTableList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NATTable, err error)
	Apply(ctx context.Context, nATTable *corev1alpha1.NATTableApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NATTable, err error)
	NATTableExpansion
}

// nATTables implements NATTableInterface
type nATTables struct {
	*gentype.ClientWithListAndApply[*v1alpha1.NATTable, *v1alpha1.NATTableList, *corev1alpha1.NATTableApplyConfiguration]
}

// newNATTables returns a NATTables
func newNATTables(c *CoreV1alpha1Client, namespace string) *nATTables {
	return &nATTables{
		gentype.NewClientWithListAndApply[*v1alpha1.NATTable, *v1alpha1.NATTableList, *corev1alpha1.NATTableApplyConfiguration](
			"nattables",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *v1alpha1.NATTable { return &v1alpha1.NATTable{} },
			func() *v1alpha1.NATTableList { return &v1alpha1.NATTableList{} }),
	}
}
