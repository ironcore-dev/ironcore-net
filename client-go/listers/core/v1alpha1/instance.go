// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	corev1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// InstanceLister helps list Instances.
// All objects returned here must be treated as read-only.
type InstanceLister interface {
	// List lists all Instances in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*corev1alpha1.Instance, err error)
	// Instances returns an object that can list and get Instances.
	Instances(namespace string) InstanceNamespaceLister
	InstanceListerExpansion
}

// instanceLister implements the InstanceLister interface.
type instanceLister struct {
	listers.ResourceIndexer[*corev1alpha1.Instance]
}

// NewInstanceLister returns a new InstanceLister.
func NewInstanceLister(indexer cache.Indexer) InstanceLister {
	return &instanceLister{listers.New[*corev1alpha1.Instance](indexer, corev1alpha1.Resource("instance"))}
}

// Instances returns an object that can list and get Instances.
func (s *instanceLister) Instances(namespace string) InstanceNamespaceLister {
	return instanceNamespaceLister{listers.NewNamespaced[*corev1alpha1.Instance](s.ResourceIndexer, namespace)}
}

// InstanceNamespaceLister helps list and get Instances.
// All objects returned here must be treated as read-only.
type InstanceNamespaceLister interface {
	// List lists all Instances in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*corev1alpha1.Instance, err error)
	// Get retrieves the Instance from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*corev1alpha1.Instance, error)
	InstanceNamespaceListerExpansion
}

// instanceNamespaceLister implements the InstanceNamespaceLister
// interface.
type instanceNamespaceLister struct {
	listers.ResourceIndexer[*corev1alpha1.Instance]
}
