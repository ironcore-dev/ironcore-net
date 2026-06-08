// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Package migrations provides concrete migration implementations for use with the
// migration.Migrator.
package migrations

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ironcore-dev/controller-utils/metautils"
	"github.com/ironcore-dev/ironcore-net/utils/origin"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OriginTypeMigration pairs an Origin with a Kubernetes object type that should be migrated.
type OriginTypeMigration struct {
	Origin *origin.Origin
	Type   client.Object
}

// OriginMetadataMigration migrates objects from an old origin metadata layout (where the
// source name was stored as a label) to the current layout (where it is stored as an
// annotation). Objects that have already been migrated are skipped. The migration lists
// all objects of the given Type that still carry the old-style name label and patches
// each one to move the name value into an annotation.
type OriginMetadataMigration struct {
	client.Client
	Origin      *origin.Origin
	Type        client.Object
	ListOptions []client.ListOption
}

func (m *OriginMetadataMigration) outdatedSelector() labels.Selector {
	sel := labels.NewSelector()
	req, _ := labels.NewRequirement(m.Origin.NameAnnotationKey(), selection.Exists, nil)
	return sel.Add(*req)
}

// Migrate lists all objects that still have the old-style origin name label and patches
// each one to move the name into an annotation. It retries on conflict and ignores
// objects that have been deleted between listing and patching.
func (m *OriginMetadataMigration) Migrate(ctx context.Context) error {
	_, list, err := metautils.NewListForObject(m.Scheme(), m.Type)
	if err != nil {
		return fmt.Errorf("new list: %w", err)
	}

	if err := m.List(ctx, list,
		append(slices.Clone(m.ListOptions), client.MatchingLabelsSelector{Selector: m.outdatedSelector()})...,
	); err != nil {
		return fmt.Errorf("list: %w", err)
	}

	var errs []error
	if err := metautils.EachListItem(list, func(obj client.Object) error {
		if err := m.migrateObject(ctx, obj); err != nil {
			errs = append(errs, fmt.Errorf("migrate object %s: %w", client.ObjectKeyFromObject(obj), err))
		}
		return nil
	}); err != nil {
		errs = append(errs, fmt.Errorf("each list item: %w", err))
	}
	return errors.Join(errs...)
}

func (m *OriginMetadataMigration) getOldOriginData(obj client.Object) *origin.Data {
	name, ok := obj.GetLabels()[m.Origin.NameAnnotationKey()]
	if !ok {
		return nil
	}

	uid, ok := obj.GetLabels()[m.Origin.UIDLabelKey()]
	if !ok {
		return nil
	}

	namespace, ok := obj.GetLabels()[m.Origin.NamespaceLabelKey()]
	if !ok && m.Origin.Namespaced {
		return nil
	}

	return &origin.Data{
		Namespace: namespace,
		Name:      name,
		UID:       types.UID(uid),
	}
}

func (m *OriginMetadataMigration) patchObject(ctx context.Context, obj client.Object, data *origin.Data) error {
	patch := client.MergeFrom(obj.DeepCopyObject().(client.Object))
	metautils.DeleteLabel(obj, m.Origin.NameAnnotationKey())
	metautils.SetAnnotation(obj, m.Origin.NameAnnotationKey(), data.Name)
	return m.Patch(ctx, obj, patch)
}

func (m *OriginMetadataMigration) migrateObject(ctx context.Context, obj client.Object) error {
	log := ctrl.LoggerFrom(ctx)

	var refetch bool
	return client.IgnoreNotFound(retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if refetch {
			key := client.ObjectKeyFromObject(obj)
			if err := m.Get(ctx, key, obj); err != nil {
				return fmt.Errorf("error getting object %s after conflict: %w", key, err)
			}
		}
		refetch = true

		oldData := m.getOldOriginData(obj)
		if oldData == nil {
			log.V(1).Info("No need to migrate object")
			return nil
		}

		return m.patchObject(ctx, obj, oldData)
	}))
}
