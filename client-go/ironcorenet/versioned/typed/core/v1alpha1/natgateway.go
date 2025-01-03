// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	json "encoding/json"
	"fmt"
	"time"

	v1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	corev1alpha1 "github.com/ironcore-dev/ironcore-net/client-go/applyconfigurations/core/v1alpha1"
	scheme "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// NATGatewaysGetter has a method to return a NATGatewayInterface.
// A group's client should implement this interface.
type NATGatewaysGetter interface {
	NATGateways(namespace string) NATGatewayInterface
}

// NATGatewayInterface has methods to work with NATGateway resources.
type NATGatewayInterface interface {
	Create(ctx context.Context, nATGateway *v1alpha1.NATGateway, opts v1.CreateOptions) (*v1alpha1.NATGateway, error)
	Update(ctx context.Context, nATGateway *v1alpha1.NATGateway, opts v1.UpdateOptions) (*v1alpha1.NATGateway, error)
	UpdateStatus(ctx context.Context, nATGateway *v1alpha1.NATGateway, opts v1.UpdateOptions) (*v1alpha1.NATGateway, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.NATGateway, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.NATGatewayList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NATGateway, err error)
	Apply(ctx context.Context, nATGateway *corev1alpha1.NATGatewayApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NATGateway, err error)
	ApplyStatus(ctx context.Context, nATGateway *corev1alpha1.NATGatewayApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NATGateway, err error)
	NATGatewayExpansion
}

// nATGateways implements NATGatewayInterface
type nATGateways struct {
	client rest.Interface
	ns     string
}

// newNATGateways returns a NATGateways
func newNATGateways(c *CoreV1alpha1Client, namespace string) *nATGateways {
	return &nATGateways{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the nATGateway, and returns the corresponding nATGateway object, and an error if there is any.
func (c *nATGateways) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.NATGateway, err error) {
	result = &v1alpha1.NATGateway{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("natgateways").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of NATGateways that match those selectors.
func (c *nATGateways) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.NATGatewayList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.NATGatewayList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("natgateways").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested nATGateways.
func (c *nATGateways) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("natgateways").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a nATGateway and creates it.  Returns the server's representation of the nATGateway, and an error, if there is any.
func (c *nATGateways) Create(ctx context.Context, nATGateway *v1alpha1.NATGateway, opts v1.CreateOptions) (result *v1alpha1.NATGateway, err error) {
	result = &v1alpha1.NATGateway{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("natgateways").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(nATGateway).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a nATGateway and updates it. Returns the server's representation of the nATGateway, and an error, if there is any.
func (c *nATGateways) Update(ctx context.Context, nATGateway *v1alpha1.NATGateway, opts v1.UpdateOptions) (result *v1alpha1.NATGateway, err error) {
	result = &v1alpha1.NATGateway{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("natgateways").
		Name(nATGateway.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(nATGateway).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *nATGateways) UpdateStatus(ctx context.Context, nATGateway *v1alpha1.NATGateway, opts v1.UpdateOptions) (result *v1alpha1.NATGateway, err error) {
	result = &v1alpha1.NATGateway{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("natgateways").
		Name(nATGateway.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(nATGateway).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the nATGateway and deletes it. Returns an error if one occurs.
func (c *nATGateways) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("natgateways").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *nATGateways) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("natgateways").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched nATGateway.
func (c *nATGateways) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.NATGateway, err error) {
	result = &v1alpha1.NATGateway{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("natgateways").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// Apply takes the given apply declarative configuration, applies it and returns the applied nATGateway.
func (c *nATGateways) Apply(ctx context.Context, nATGateway *corev1alpha1.NATGatewayApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NATGateway, err error) {
	if nATGateway == nil {
		return nil, fmt.Errorf("nATGateway provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(nATGateway)
	if err != nil {
		return nil, err
	}
	name := nATGateway.Name
	if name == nil {
		return nil, fmt.Errorf("nATGateway.Name must be provided to Apply")
	}
	result = &v1alpha1.NATGateway{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("natgateways").
		Name(*name).
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *nATGateways) ApplyStatus(ctx context.Context, nATGateway *corev1alpha1.NATGatewayApplyConfiguration, opts v1.ApplyOptions) (result *v1alpha1.NATGateway, err error) {
	if nATGateway == nil {
		return nil, fmt.Errorf("nATGateway provided to Apply must not be nil")
	}
	patchOpts := opts.ToPatchOptions()
	data, err := json.Marshal(nATGateway)
	if err != nil {
		return nil, err
	}

	name := nATGateway.Name
	if name == nil {
		return nil, fmt.Errorf("nATGateway.Name must be provided to Apply")
	}

	result = &v1alpha1.NATGateway{}
	err = c.client.Patch(types.ApplyPatchType).
		Namespace(c.ns).
		Resource("natgateways").
		Name(*name).
		SubResource("status").
		VersionedParams(&patchOpts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
