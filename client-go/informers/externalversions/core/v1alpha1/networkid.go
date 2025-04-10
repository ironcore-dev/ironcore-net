// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	context "context"
	time "time"

	apicorev1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	internalinterfaces "github.com/ironcore-dev/ironcore-net/client-go/informers/externalversions/internalinterfaces"
	versioned "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned"
	corev1alpha1 "github.com/ironcore-dev/ironcore-net/client-go/listers/core/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// NetworkIDInformer provides access to a shared informer and lister for
// NetworkIDs.
type NetworkIDInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() corev1alpha1.NetworkIDLister
}

type networkIDInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// NewNetworkIDInformer constructs a new informer for NetworkID type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewNetworkIDInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredNetworkIDInformer(client, resyncPeriod, indexers, nil)
}

// NewFilteredNetworkIDInformer constructs a new informer for NetworkID type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredNetworkIDInformer(client versioned.Interface, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1alpha1().NetworkIDs().List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1alpha1().NetworkIDs().Watch(context.TODO(), options)
			},
		},
		&apicorev1alpha1.NetworkID{},
		resyncPeriod,
		indexers,
	)
}

func (f *networkIDInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredNetworkIDInformer(client, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *networkIDInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&apicorev1alpha1.NetworkID{}, f.defaultInformer)
}

func (f *networkIDInformer) Lister() corev1alpha1.NetworkIDLister {
	return corev1alpha1.NewNetworkIDLister(f.Informer().GetIndexer())
}
