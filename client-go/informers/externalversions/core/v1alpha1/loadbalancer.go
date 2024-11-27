// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	corev1alpha1 "github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	internalinterfaces "github.com/ironcore-dev/ironcore-net/client-go/informers/externalversions/internalinterfaces"
	versioned "github.com/ironcore-dev/ironcore-net/client-go/ironcorenet/versioned"
	v1alpha1 "github.com/ironcore-dev/ironcore-net/client-go/listers/core/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// LoadBalancerInformer provides access to a shared informer and lister for
// LoadBalancers.
type LoadBalancerInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.LoadBalancerLister
}

type loadBalancerInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewLoadBalancerInformer constructs a new informer for LoadBalancer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewLoadBalancerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredLoadBalancerInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredLoadBalancerInformer constructs a new informer for LoadBalancer type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredLoadBalancerInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1alpha1().LoadBalancers(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CoreV1alpha1().LoadBalancers(namespace).Watch(context.TODO(), options)
			},
		},
		&corev1alpha1.LoadBalancer{},
		resyncPeriod,
		indexers,
	)
}

func (f *loadBalancerInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredLoadBalancerInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *loadBalancerInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&corev1alpha1.LoadBalancer{}, f.defaultInformer)
}

func (f *loadBalancerInformer) Lister() v1alpha1.LoadBalancerLister {
	return v1alpha1.NewLoadBalancerLister(f.Informer().GetIndexer())
}