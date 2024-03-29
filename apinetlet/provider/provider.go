// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/types"
)

const (
	apiNetPrefix = "ironcore-net://"
)

// ParseNetworkInterfaceID parses network interface provider IDs.
// The format of a network interface provider id is as follows:
// ironcore-net://<namespace>/<name>/<node>/<uid>
func ParseNetworkInterfaceID(id string) (namespace, name, node string, uid types.UID, err error) {
	parts := strings.SplitN(strings.TrimPrefix(id, apiNetPrefix), "/", 5)
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("invalid provider id %q", id)
	}

	namespace = parts[0]
	name = parts[1]
	node = parts[2]
	uid = types.UID(parts[3])
	if allErrs := validation.NameIsDNSSubdomain(node, false); len(allErrs) != 0 {
		return "", "", "", "", fmt.Errorf("node is not a dns label: %v", allErrs)
	}

	return namespace, name, node, uid, nil
}

// GetNetworkInterfaceID creates a network interface provider ID.
// The format of a network interface provider id is as follows:
// ironcore-net://<namespace>/<name>/<node>/<uid>
func GetNetworkInterfaceID(namespace, name, node string, uid types.UID) string {
	var sb strings.Builder
	sb.Grow(len(apiNetPrefix) + len(namespace) + 1 + len(name) + 1 + len(node) + 1 + len(uid))
	sb.WriteString(apiNetPrefix)
	sb.WriteString(namespace)
	sb.WriteRune('/')
	sb.WriteString(name)
	sb.WriteRune('/')
	sb.WriteString(node)
	sb.WriteRune('/')
	sb.WriteString(string(uid))
	return sb.String()
}

// ParseNetworkID parses network provider IDs into the apinet network name.
// The format of a network provider ID is as follows:
// ironcore-net://<id>/<name>/<uid>
func ParseNetworkID(s string) (namespace, name, id string, uid types.UID, err error) {
	parts := strings.SplitN(strings.TrimPrefix(s, apiNetPrefix), "/", 5)
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("invalid provider id %q", id)
	}

	namespace = parts[0]
	name = parts[1]
	id = parts[2]
	uid = types.UID(parts[3])
	return namespace, name, id, uid, nil
}

// GetNetworkID creates a network ID from the given id, name and UID.
// The format of a network provider ID is as follows:
// ironcore-net://<namespace>/<name>/<id>/<uid>
func GetNetworkID(namespace, name, id string, uid types.UID) string {
	var sb strings.Builder
	sb.Grow(len(apiNetPrefix) + len(namespace) + 1 + len(id) + 1 + len(name) + 1 + len(uid))
	sb.WriteString(apiNetPrefix)
	sb.WriteString(namespace)
	sb.WriteRune('/')
	sb.WriteString(name)
	sb.WriteRune('/')
	sb.WriteString(id)
	sb.WriteRune('/')
	sb.WriteString(string(uid))
	return sb.String()
}
