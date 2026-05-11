// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

// Package origin provides a mechanism for tracking the provenance of Kubernetes objects
// that are created by controllers as projections of source objects. It encodes the source
// object's name, namespace, and UID into labels and annotations on the target object,
// allowing controllers to determine which source object a target was derived from.
package origin

import (
	"fmt"

	"github.com/ironcore-dev/controller-utils/metautils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Origin defines a provenance scheme for tracking which source object a target object
// was derived from. It stores the source's UID and (optionally) namespace as labels,
// and the source's name as an annotation on the target object. Set Namespaced to true
// if the source objects are namespace-scoped.
type Origin struct {
	Name       string
	Namespaced bool
}

func (s *Origin) key(name string) string {
	return fmt.Sprintf("%s-%s", s.Name, name)
}

// UIDLabelKey returns the label key used to store the source object's UID.
func (s *Origin) UIDLabelKey() string { return s.key("uid") }

// NamespaceLabelKey returns the label key used to store the source object's namespace.
func (s *Origin) NamespaceLabelKey() string { return s.key("namespace") }

// NameAnnotationKey returns the annotation key used to store the source object's name.
func (s *Origin) NameAnnotationKey() string { return s.key("name") }

func (s *Origin) labelKeys() []string {
	keys := []string{
		s.UIDLabelKey(),
	}
	if s.Namespaced {
		keys = append(keys, s.NamespaceLabelKey())
	}

	return keys
}

func (s *Origin) annotationKeys() []string {
	return []string{s.NameAnnotationKey()}
}

// RemoveOrigin removes all origin labels and annotations from the given object.
func (s *Origin) RemoveOrigin(obj client.Object) {
	labels := obj.GetLabels()
	for _, labelKey := range s.labelKeys() {
		delete(labels, labelKey)
	}
	obj.SetLabels(labels)

	annotations := obj.GetAnnotations()
	for _, annotationKey := range s.annotationKeys() {
		delete(annotations, annotationKey)
	}
	obj.SetAnnotations(annotations)
}

// SetOrigin stamps the target object with origin labels and annotations derived from the
// source object. Existing labels and annotations on the target are preserved.
func (s *Origin) SetOrigin(sourceObj, obj client.Object) {
	metautils.SetLabels(obj, s.Labels(sourceObj))
	metautils.SetAnnotations(obj, s.Annotations(sourceObj))
}

// Labels returns the origin labels that should be set on a target object for the given source.
func (s *Origin) Labels(sourceObj client.Object) map[string]string {
	lbls := map[string]string{
		s.UIDLabelKey(): string(sourceObj.GetUID()),
	}
	if s.Namespaced {
		lbls[s.NamespaceLabelKey()] = sourceObj.GetNamespace()
	}

	return lbls
}

// Annotations returns the origin annotations that should be set on a target object for the given source.
func (s *Origin) Annotations(sourceObj client.Object) map[string]string {
	return map[string]string{
		s.NameAnnotationKey(): sourceObj.GetName(),
	}
}

// StemsFrom reports whether obj was derived from sourceObj by comparing the stored UID.
func (s *Origin) StemsFrom(obj, sourceObj client.Object) bool {
	data := s.DataOf(obj)
	if data == nil {
		return false
	}

	return data.UID == sourceObj.GetUID()
}

// StemsFromKey reports whether obj was derived from a source object identified by
// the given namespace/name key.
func (s *Origin) StemsFromKey(obj client.Object, key client.ObjectKey) bool {
	data := s.DataOf(obj)
	if data == nil {
		return false
	}

	return data.Namespace == key.Namespace && data.Name == key.Name
}

// Data holds the source object's namespace, name, and UID as extracted from origin
// labels and annotations on a target object.
type Data struct {
	Namespace string
	Name      string
	UID       types.UID
}

// DataOf extracts the origin data from obj's labels and annotations. It returns nil if
// any required field is missing (UID label, name annotation, and namespace label for
// namespaced origins).
func (s *Origin) DataOf(obj client.Object) *Data {
	lbls := obj.GetLabels()
	annotations := obj.GetAnnotations()

	var namespace string
	if s.Namespaced {
		var ok bool
		namespace, ok = lbls[s.NamespaceLabelKey()]
		if !ok {
			return nil
		}
	}

	name, ok := annotations[s.NameAnnotationKey()]
	if !ok {
		return nil
	}

	uid, ok := lbls[s.UIDLabelKey()]
	if !ok {
		return nil
	}

	return &Data{
		Namespace: namespace,
		Name:      name,
		UID:       types.UID(uid),
	}
}
