// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package client

import (
	utilapi "github.com/ironcore-dev/ironcore-net/utils/api"
	utilclient "github.com/ironcore-dev/ironcore-net/utils/client"
)

var (
	SourceAwareSystem = utilclient.NewSourceAwareSystem(utilapi.PrefixSourceAwareSystem("metalnetlet.ironcore.dev/"))

	SourceLabelKeysE = SourceAwareSystem.SourceLabelKeysE

	SourceLabelKeys = SourceAwareSystem.SourceLabelKeys

	SourceLabelsE = SourceAwareSystem.SourceLabelsE

	SourceLabels = SourceAwareSystem.SourceLabels

	MatchingSourceLabelsE = SourceAwareSystem.MatchingSourceLabelsE

	MatchingSourceLabels = SourceAwareSystem.MatchingSourceLabels

	HasSourceLabelsE = SourceAwareSystem.HasSourceLabelsE

	HasSourceLabels = SourceAwareSystem.HasSourceLabels

	SourceKeyLabelsE = SourceAwareSystem.SourceKeyLabelsE

	SourceKeyLabels = SourceAwareSystem.SourceKeyLabels

	MatchingSourceKeyLabelsE = SourceAwareSystem.MatchingSourceKeyLabelsE

	MatchingSourceKeyLabels = SourceAwareSystem.MatchingSourceKeyLabels

	SourceObjectKeyFromObjectE = SourceAwareSystem.SourceObjectKeyFromObjectE

	SourceObjectKeyFromObject = SourceAwareSystem.SourceObjectKeyFromObject

	SourceObjectDataFromObjectE = SourceAwareSystem.SourceObjectDataFromObjectE

	SourceObjectDataFromObject = SourceAwareSystem.SourceObjectDataFromObject
)

type SourceObjectData = utilclient.SourceObjectData
