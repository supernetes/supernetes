// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/supernetes/supernetes/common/pkg/log"
)

var logLevel string

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "config",
		Short: "Supernetes controller and agent configuration generator",
		Long: dedent.Dedent(`
			Generate the necessary configuration files for deploying a Supernetes controller
			and agent. Alongside basic controller and agent configuration, this tool is able
			to generate the linked mTLS key pairs that are used to mutually authenticate the
			controller and agent upon connection establishment.
		`),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			log.Init(logLevel)
		},
	}

	addGlobalFlags(root.PersistentFlags())
	root.AddCommand(NewCmdGenerate())
	return root
}

func addGlobalFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&logLevel, "log-level", "l", "info", "zerolog log level")
}
