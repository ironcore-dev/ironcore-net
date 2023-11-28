// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package testing

import (
	"fmt"
	"reflect"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
)

type asRef struct {
	matcher types.GomegaMatcher
}

func (a *asRef) toRef(actual any) any {
	rv := reflect.ValueOf(actual)
	pv := reflect.New(rv.Type())
	pv.Elem().Set(rv)
	return pv.Interface()
}

func (a *asRef) Match(actual any) (success bool, err error) {
	return a.matcher.Match(a.toRef(actual))
}

func (a *asRef) FailureMessage(actual any) (message string) {
	return a.matcher.FailureMessage(a.toRef(actual))
}

func (a *asRef) NegatedFailureMessage(actual any) (message string) {
	return a.matcher.NegatedFailureMessage(a.toRef(actual))
}

func AsRef(matcher types.GomegaMatcher) types.GomegaMatcher {
	return &asRef{matcher: matcher}
}

type capture struct {
	vType   reflect.Type
	intoV   reflect.Value
	matcher types.GomegaMatcher
}

func (c *capture) Match(actual any) (success bool, err error) {
	actualV := reflect.ValueOf(actual)
	if !actualV.CanConvert(c.vType) {
		return false, fmt.Errorf("cannot convert %T to %s", actual, c.vType)
	}

	success, err = c.matcher.Match(actual)
	if success {
		c.intoV.Elem().Set(actualV.Convert(c.vType))
	}
	return success, err
}

func (c *capture) FailureMessage(actual any) (message string) {
	return c.matcher.FailureMessage(actual)
}

func (c *capture) NegatedFailureMessage(actual any) (message string) {
	return c.matcher.NegatedFailureMessage(actual)
}

func Capture(into any, matcher types.GomegaMatcher) types.GomegaMatcher {
	intoV := reflect.ValueOf(into)
	if intoV.Kind() != reflect.Pointer {
		ginkgo.Fail(fmt.Sprintf("value %T is not a pointer-type", intoV))
	}

	return &capture{
		vType:   intoV.Type().Elem(),
		intoV:   intoV,
		matcher: matcher,
	}
}

type haveKeysWithValues[K comparable, V any] struct {
	keysWithValues map[K]V
}

func (k *haveKeysWithValues[K, V]) Match(actualV any) (success bool, err error) {
	actual, ok := actualV.(map[K]V)
	if !ok {
		var (
			k K
			v V
		)
		return false, fmt.Errorf("HaveKeysWithValues matcher requires a map[%T]%T.  Got:\n%s", k, v, format.Object(actual, 1))
	}

	if len(actual) < len(k.keysWithValues) {
		return false, nil
	}

	for k, v := range k.keysWithValues {
		aV, ok := actual[k]
		if !ok || !reflect.DeepEqual(aV, v) {
			return false, nil
		}
	}
	return true, nil
}

func (k *haveKeysWithValues[K, V]) FailureMessage(actual any) (message string) {
	return format.Message(actual, "to contain keys with values", k.keysWithValues)
}

func (k *haveKeysWithValues[K, V]) NegatedFailureMessage(actual any) (message string) {
	return format.Message(actual, "not to contain keys with values", k.keysWithValues)
}

func HaveKeysWithValues[M ~map[K]V, K comparable, V any](keysWithValues M) types.GomegaMatcher {
	return &haveKeysWithValues[K, V]{
		keysWithValues: keysWithValues,
	}
}
