// Copyright 2023 IronCore authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package nodeaffinity

import (
	"errors"

	"github.com/ironcore-dev/ironcore-net/api/core/v1alpha1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type LazyErrorNodeSelector struct {
	terms []nodeSelectorTerm
}

type RequiredNodeAffinity struct {
	nodeSelector *LazyErrorNodeSelector
}

type nodeSelectorTerm struct {
	matchLabels labels.Selector
	matchFields fields.Selector
	parseErrs   []error
}

func (t *nodeSelectorTerm) match(nodeLabels labels.Set, nodeFields fields.Set) (bool, []error) {
	if t.parseErrs != nil {
		return false, t.parseErrs
	}
	if t.matchLabels != nil && !t.matchLabels.Matches(nodeLabels) {
		return false, nil
	}
	if t.matchFields != nil && len(nodeFields) > 0 && !t.matchFields.Matches(nodeFields) {
		return false, nil
	}
	return true, nil
}

var validSelectorOperators = []string{
	string(v1alpha1.NodeSelectorOpIn),
	string(v1alpha1.NodeSelectorOpNotIn),
	string(v1alpha1.NodeSelectorOpExists),
	string(v1alpha1.NodeSelectorOpDoesNotExist),
	string(v1alpha1.NodeSelectorOpGt),
	string(v1alpha1.NodeSelectorOpLt),
}

func nodeSelectorRequirementsAsSelector(nsm []v1alpha1.NodeSelectorRequirement, path *field.Path) (labels.Selector, []error) {
	if len(nsm) == 0 {
		return labels.Nothing(), nil
	}
	var errs []error
	selector := labels.NewSelector()
	for i, expr := range nsm {
		p := path.Index(i)
		var op selection.Operator
		switch expr.Operator {
		case v1alpha1.NodeSelectorOpIn:
			op = selection.In
		case v1alpha1.NodeSelectorOpNotIn:
			op = selection.NotIn
		case v1alpha1.NodeSelectorOpExists:
			op = selection.Exists
		case v1alpha1.NodeSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		case v1alpha1.NodeSelectorOpGt:
			op = selection.GreaterThan
		case v1alpha1.NodeSelectorOpLt:
			op = selection.LessThan
		default:
			errs = append(errs, field.NotSupported(p.Child("operator"), expr.Operator, validSelectorOperators))
			continue
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values, field.WithPath(p))
		if err != nil {
			errs = append(errs, err)
		} else {
			selector = selector.Add(*r)
		}
	}
	if len(errs) != 0 {
		return nil, errs
	}
	return selector, nil
}

var validFieldSelectorOperators = []string{
	string(v1alpha1.NodeSelectorOpIn),
	string(v1alpha1.NodeSelectorOpNotIn),
}

func nodeSelectorRequirementsAsFieldSelector(nsr []v1alpha1.NodeSelectorRequirement, path *field.Path) (fields.Selector, []error) {
	if len(nsr) == 0 {
		return fields.Nothing(), nil
	}
	var errs []error

	var selectors []fields.Selector
	for i, expr := range nsr {
		p := path.Index(i)
		switch expr.Operator {
		case v1alpha1.NodeSelectorOpIn:
			if len(expr.Values) != 1 {
				errs = append(errs, field.Invalid(p.Child("values"), expr.Values, "must have one element"))
			} else {
				selectors = append(selectors, fields.OneTermEqualSelector(expr.Key, expr.Values[0]))
			}

		case v1alpha1.NodeSelectorOpNotIn:
			if len(expr.Values) != 1 {
				errs = append(errs, field.Invalid(p.Child("values"), expr.Values, "must have one element"))
			} else {
				selectors = append(selectors, fields.OneTermNotEqualSelector(expr.Key, expr.Values[0]))
			}

		default:
			errs = append(errs, field.NotSupported(p.Child("operator"), expr.Operator, validFieldSelectorOperators))
		}
	}

	if len(errs) != 0 {
		return nil, errs
	}
	return fields.AndSelectors(selectors...), nil
}

func newNodeSelectorTerm(term *v1alpha1.NodeSelectorTerm, path *field.Path) nodeSelectorTerm {
	var (
		parsedTerm nodeSelectorTerm
		errs       []error
	)
	if len(term.MatchExpressions) != 0 {
		p := path.Child("matchExpressions")
		parsedTerm.matchLabels, errs = nodeSelectorRequirementsAsSelector(term.MatchExpressions, p)
		if errs != nil {
			parsedTerm.parseErrs = append(parsedTerm.parseErrs, errs...)
		}
	}
	if len(term.MatchFields) != 0 {
		p := path.Child("matchField")
		parsedTerm.matchFields, errs = nodeSelectorRequirementsAsFieldSelector(term.MatchFields, p)
		if errs != nil {
			parsedTerm.parseErrs = append(parsedTerm.parseErrs, errs...)
		}
	}
	return parsedTerm
}

func isEmptyNodeSelectorTerm(term *v1alpha1.NodeSelectorTerm) bool {
	return len(term.MatchFields) == 0 && len(term.MatchExpressions) == 0
}

func (s *LazyErrorNodeSelector) Match(node *v1alpha1.Node) (bool, error) {
	if node == nil {
		return false, nil
	}
	nodeLabels := labels.Set(node.Labels)
	nodeFields := extractNodeFields(node)

	var errs []error
	for _, term := range s.terms {
		match, tErrs := term.match(nodeLabels, nodeFields)
		if len(tErrs) > 0 {
			errs = append(errs, tErrs...)
			continue
		}
		if match {
			return true, nil
		}
	}
	return false, errors.Join(errs...)
}

func extractNodeFields(node *v1alpha1.Node) fields.Set {
	f := make(fields.Set)
	if len(node.Name) > 0 {
		f["metadata.name"] = node.Name
	}
	return f
}

func NewLazyErrorNodeSelector(ns *v1alpha1.NodeSelector) *LazyErrorNodeSelector {
	p := field.ToPath()
	parsedTerms := make([]nodeSelectorTerm, 0, len(ns.NodeSelectorTerms))
	path := p.Child("nodeSelectorTerms")
	for i, term := range ns.NodeSelectorTerms {
		if isEmptyNodeSelectorTerm(&term) {
			continue
		}

		p := path.Index(i)
		parsedTerms = append(parsedTerms, newNodeSelectorTerm(&term, p))
	}

	return &LazyErrorNodeSelector{
		terms: parsedTerms,
	}
}

func (s RequiredNodeAffinity) Match(node *v1alpha1.Node) (bool, error) {
	if s.nodeSelector != nil {
		return s.nodeSelector.Match(node)
	}
	return true, nil
}

func GetRequiredNodeAffinity(inst *v1alpha1.Instance) RequiredNodeAffinity {
	var affinity *LazyErrorNodeSelector
	if inst.Spec.Affinity != nil &&
		inst.Spec.Affinity.NodeAffinity != nil &&
		inst.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		affinity = NewLazyErrorNodeSelector(inst.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	}

	return RequiredNodeAffinity{nodeSelector: affinity}
}
