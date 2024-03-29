// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/gardener/gardener-discovery-server/cmd/discovery-server/app"

	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	cmd := app.NewCommand()
	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)

	if err := cmd.ExecuteContext(ctrl.SetupSignalHandler()); err != nil {
		ctrl.Log.Error(err, "Failed to run application", "name", cmd.Name())
		os.Exit(1)
	}
}
