// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"context"
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/klog/v2"
	"k8s.io/utils/lru"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetTargetNodeName(inst *v1alpha1.Instance) (string, error) {
	if nodeRef := inst.Spec.NodeRef; nodeRef != nil {
		return nodeRef.Name, nil
	}

	if inst.Spec.Affinity == nil ||
		inst.Spec.Affinity.NodeAffinity == nil ||
		inst.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		return "", fmt.Errorf("no spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution for load balancer instance %s", klog.KObj(inst))
	}

	terms := inst.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(terms) < 1 {
		return "", fmt.Errorf("no nodeSelectorTerms in requiredDuringSchedulingIgnoredDuringExecution of load balancer instance %s", klog.KObj(inst))
	}

	for _, term := range terms {
		for _, exp := range term.MatchFields {
			if exp.Key == metav1.ObjectNameField &&
				exp.Operator == v1alpha1.NodeSelectorOpIn {
				if len(exp.Values) != 1 {
					return "", fmt.Errorf("the matchFields value of '%s' is not unique for load balancer instance %s",
						metav1.ObjectNameField, klog.KObj(inst))
				}

				return exp.Values[0], nil
			}
		}
	}
	return "", fmt.Errorf("no node name found for load balancer instance %s", klog.KObj(inst))
}

// ReplaceDaemonSetInstanceNodeNameNodeAffinity replaces the RequiredDuringSchedulingIgnoredDuringExecution
// NodeAffinity of the given affinity with a new NodeAffinity that selects the given nodeName.
// Note that this function assumes that no NodeAffinity conflicts with the selected nodeName.
func ReplaceDaemonSetInstanceNodeNameNodeAffinity(affinity *v1alpha1.Affinity, nodeName string) *v1alpha1.Affinity {
	nodeSelReq := v1alpha1.NodeSelectorRequirement{
		Key:      metav1.ObjectNameField,
		Operator: v1alpha1.NodeSelectorOpIn,
		Values:   []string{nodeName},
	}

	nodeSelector := &v1alpha1.NodeSelector{
		NodeSelectorTerms: []v1alpha1.NodeSelectorTerm{
			{
				MatchFields: []v1alpha1.NodeSelectorRequirement{nodeSelReq},
			},
		},
	}

	if affinity == nil {
		return &v1alpha1.Affinity{
			NodeAffinity: &v1alpha1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: nodeSelector,
			},
		}
	}

	if affinity.NodeAffinity == nil {
		affinity.NodeAffinity = &v1alpha1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: nodeSelector,
		}
		return affinity
	}

	nodeAffinity := affinity.NodeAffinity

	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nodeSelector
		return affinity
	}

	// Replace node selector with the new one.
	nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []v1alpha1.NodeSelectorTerm{
		{
			MatchFields: []v1alpha1.NodeSelectorRequirement{nodeSelReq},
		},
	}

	return affinity
}

// ComputeHash returns a hash value calculated from pod template and
// a collisionCount to avoid hash collision. The hash will be safe encoded to
// avoid bad words.
func ComputeHash(template *v1alpha1.InstanceTemplate, collisionCount *int32) string {
	podTemplateSpecHasher := fnv.New32a()
	_, _ = fmt.Fprintf(podTemplateSpecHasher, "%#+v", *template)

	// Add collisionCount in the hash if it exists.
	if collisionCount != nil {
		collisionCountBytes := make([]byte, 8)
		binary.LittleEndian.PutUint32(collisionCountBytes, uint32(*collisionCount))
		_, _ = podTemplateSpecHasher.Write(collisionCountBytes)
	}

	return rand.SafeEncodeString(fmt.Sprint(podTemplateSpecHasher.Sum32()))
}

func NewPartialObjectMetadata(restMapper meta.RESTMapper, gvr schema.GroupVersionResource) (*metav1.PartialObjectMetadata, error) {
	resList, err := restMapper.KindsFor(gvr)
	if err != nil {
		return nil, fmt.Errorf("error getting kinds for %s: %w", gvr.GroupResource(), err)
	}
	if len(resList) == 0 {
		return nil, fmt.Errorf("no kind for %s", gvr.GroupResource())
	}

	gvk := resList[0]
	return &metav1.PartialObjectMetadata{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
	}, nil
}

func GetWithAbsenceCache(
	ctx context.Context,
	apiReader client.Reader,
	absenceCache *lru.Cache,
	key client.ObjectKey,
	obj client.Object,
	uid types.UID,
) error {
	if _, ok := absenceCache.Get(uid); ok {
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}

	if err := apiReader.Get(ctx, key, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		absenceCache.Add(uid, nil)
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}
	if uid != obj.GetUID() {
		absenceCache.Add(uid, nil)
		return apierrors.NewNotFound(schema.GroupResource{}, key.String())
	}
	return nil
}
