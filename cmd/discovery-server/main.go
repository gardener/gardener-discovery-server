// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log"

	"github.com/gardener/gardener-discovery-server/cmd/discovery-server/app"

	"github.com/spf13/pflag"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	ctrl "sigs.k8s.io/controller-runtime"
)

func main() {
	cmd := app.NewCommand()
	pflag.CommandLine.SetNormalizeFunc(cliflag.WordSepNormalizeFunc)
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := cmd.ExecuteContext(ctrl.SetupSignalHandler()); err != nil {
		ctrl.Log.Info(err.Error())
		log.Fatal(err)
	}
}
