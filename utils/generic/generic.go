// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package generic

func EqualPointers[E comparable](e1, e2 *E) bool {
	e1Nil := e1 == nil
	e2Nil := e2 == nil
	if e1Nil != e2Nil {
		// Nil-ness of pointers is different - not equal.
		return false
	}
	if e1Nil {
		// Both are nil - equal.
		return true
	}
	// Do actual comparison.
	return *e1 == *e2
}
