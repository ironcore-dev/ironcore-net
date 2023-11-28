// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	MetalnetFieldOwner = client.FieldOwner("metalnetlet.apinet.ironcore.dev/controller-manager")

	PartitionFieldOwnerPrefix = "partition.metalnetlet.apinet.ironcore.dev/"

	PartitionFinalizerPrefix = "partition.metalnetlet.apinet.ironcore.dev/"
)

func PartitionFieldOwner(partitionName string) client.FieldOwner {
	return client.FieldOwner(PartitionFieldOwnerPrefix + partitionName)
}

func PartitionFinalizer(partitionName string) string {
	return PartitionFinalizerPrefix + partitionName
}

func PartitionNodeName(partitionName, metalnetNodeName string) string {
	return fmt.Sprintf("%s.%s", partitionName, metalnetNodeName)
}

func ParseNodeName(partitionName, nodeName string) (string, error) {
	prefix := partitionName + "."
	if !strings.HasPrefix(nodeName, prefix) {
		return "", fmt.Errorf("node name %q does not belong to partition %s", nodeName, partitionName)
	}
	return strings.TrimPrefix(nodeName, prefix), nil
}

func IsNodeOnPartitionPredicate(partitionName string) predicate.Predicate {
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		node := obj.(*v1alpha1.Node)
		_, err := ParseNodeName(partitionName, node.Name)
		return err == nil
	})
}

func GetMetalnetNode(ctx context.Context, partitionName string, metalnetClient client.Client, nodeName string) (*corev1.Node, error) {
	metalnetNodeName, err := ParseNodeName(partitionName, nodeName)
	if err != nil {
		// Ignore any parsing error, what we know is that the node does not exist on our side.
		return nil, nil
	}

	metalnetNode := &corev1.Node{}
	metalnetNodeKey := client.ObjectKey{Name: metalnetNodeName}
	if err := metalnetClient.Get(ctx, metalnetNodeKey, metalnetNode); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		return nil, nil
	}
	return metalnetNode, nil
}
