// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package networkid

import (
	"errors"
	"strconv"
)

var (
	ErrInvalidVNI = errors.New("invalid VNI")
)

func ParseVNI(id string) (int32, error) {
	vni64, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		return 0, ErrInvalidVNI
	}

	vni := int32(vni64)
	return vni, nil
}

func EncodeVNI(vni int32) string {
	return strconv.FormatInt(int64(vni), 10)
}
