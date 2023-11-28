// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/ironcore-dev/ironcore-net/internal/app/apiserver"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/component-base/cli"
)

func main() {
	ctx := server.SetupSignalContext()
	options := apiserver.NewIronCoreNetServerOptions()
	cmd := apiserver.NewCommandStartIronCoreNetServer(ctx, options)
	code := cli.Run(cmd)
	os.Exit(code)
}
