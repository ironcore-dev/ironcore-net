// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/listers"
	"k8s.io/client-go/tools/cache"
)

// DaemonSetLister helps list DaemonSets.
// All objects returned here must be treated as read-only.
type DaemonSetLister interface {
	// List lists all DaemonSets in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.DaemonSet, err error)
	// DaemonSets returns an object that can list and get DaemonSets.
	DaemonSets(namespace string) DaemonSetNamespaceLister
	DaemonSetListerExpansion
}

// daemonSetLister implements the DaemonSetLister interface.
type daemonSetLister struct {
	listers.ResourceIndexer[*v1alpha1.DaemonSet]
}

// NewDaemonSetLister returns a new DaemonSetLister.
func NewDaemonSetLister(indexer cache.Indexer) DaemonSetLister {
	return &daemonSetLister{listers.New[*v1alpha1.DaemonSet](indexer, v1alpha1.Resource("daemonset"))}
}

// DaemonSets returns an object that can list and get DaemonSets.
func (s *daemonSetLister) DaemonSets(namespace string) DaemonSetNamespaceLister {
	return daemonSetNamespaceLister{listers.NewNamespaced[*v1alpha1.DaemonSet](s.ResourceIndexer, namespace)}
}

// DaemonSetNamespaceLister helps list and get DaemonSets.
// All objects returned here must be treated as read-only.
type DaemonSetNamespaceLister interface {
	// List lists all DaemonSets in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.DaemonSet, err error)
	// Get retrieves the DaemonSet from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.DaemonSet, error)
	DaemonSetNamespaceListerExpansion
}

// daemonSetNamespaceLister implements the DaemonSetNamespaceLister
// interface.
type daemonSetNamespaceLister struct {
	listers.ResourceIndexer[*v1alpha1.DaemonSet]
}
