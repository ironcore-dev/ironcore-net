// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

const (
	ControllerRevisionHashLabel = "apinet.ironcore.dev/controller-revision-hash"

	IPFamilyLabel = "apinet.ironcore.dev/ip-family"
	IPIPLabel     = "apinet.ironcore.dev/ip"

	TopologyLabelPrefix    = "topology.core.apinet.ironcore.dev/"
	TopologyPartitionLabel = TopologyLabelPrefix + "partition"
	TopologyZoneLabel      = TopologyLabelPrefix + "zone"
)
