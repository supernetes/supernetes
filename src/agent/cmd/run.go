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
	"github.com/supernetes/supernetes/agent/pkg/agent"
	"github.com/supernetes/supernetes/common/pkg/log"
)

func NewCmdRun() *cobra.Command {
	flags := agent.NewFlags()

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the Supernetes agent",
		Long: dedent.Dedent(`
			Run the Supernetes agent. Supplying an agent configuration is mandatory.
		`),
		Run: func(cmd *cobra.Command, args []string) {
			options, err := flags.NewOptions(args)
			log.FatalErr(err).Msg("failed to parse options")
			agent.Run(options)
		},
	}

	addRunFlags(cmd.Flags(), flags)
	markRunFlags(cmd)
	return cmd
}

func addRunFlags(fs *pflag.FlagSet, flags *agent.Flags) {
	fs.StringVarP(&flags.ConfigPath, "config", "c", "", "path to agent configuration file")
}

func markRunFlags(cmd *cobra.Command) {
	fatal(cmd.MarkFlagFilename("config", "yaml"))
	fatal(cmd.MarkFlagRequired("config"))
}
