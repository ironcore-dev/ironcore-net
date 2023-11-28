// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	utilapi "github.com/ironcore-dev/ironcore-net/utils/api"
	utilhandler "github.com/ironcore-dev/ironcore-net/utils/handler"
)

var (
	SourceAwareSystem = utilhandler.NewSourceAwareSystem(utilapi.PrefixSourceAwareSystem("apinetlet.ironcore.dev/"))

	EnqueueRequestForSource = SourceAwareSystem.EnqueueRequestForSource
)
