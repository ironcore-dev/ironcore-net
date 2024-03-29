// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// IPAddressLister helps list IPAddresses.
// All objects returned here must be treated as read-only.
type IPAddressLister interface {
	// List lists all IPAddresses in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.IPAddress, err error)
	// Get retrieves the IPAddress from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.IPAddress, error)
	IPAddressListerExpansion
}

// iPAddressLister implements the IPAddressLister interface.
type iPAddressLister struct {
	indexer cache.Indexer
}

// NewIPAddressLister returns a new IPAddressLister.
func NewIPAddressLister(indexer cache.Indexer) IPAddressLister {
	return &iPAddressLister{indexer: indexer}
}

// List lists all IPAddresses in the indexer.
func (s *iPAddressLister) List(selector labels.Selector) (ret []*v1alpha1.IPAddress, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.IPAddress))
	})
	return ret, err
}

// Get retrieves the IPAddress from the index for a given name.
func (s *iPAddressLister) Get(name string) (*v1alpha1.IPAddress, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("ipaddress"), name)
	}
	return obj.(*v1alpha1.IPAddress), nil
}
