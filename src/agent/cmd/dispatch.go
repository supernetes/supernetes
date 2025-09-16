// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"os"

	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/supernetes/supernetes/agent/pkg/dispatch"
	"github.com/supernetes/supernetes/common/pkg/log"
)

func NewCmdDispatch() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dispatch <containers>",
		Short: "Internal command for the agent to dispatch containers through the HPC scheduler",
		Long: dedent.Dedent(`
			Internal command that takes in a Base64-encoded array of Supernetes container specifications
			and executes them in parallel, prepending RFC3339 timestamps and the container name to each
			output line. This will be executed by the HPC scheduler, such as Slurm.
		`),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				log.Fatal().Msgf("usage: %s %s", os.Args[0], cmd.Use)
			}

			dispatch.Run(args[0])
		},
	}

	return cmd
}
