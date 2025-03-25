// SPDX-FileCopyrightText: 2022 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package controllers

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/klog/v2"
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
