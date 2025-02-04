// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

import (
	"github.com/lithammer/dedent"
	"github.com/spf13/cobra"
	"github.com/supernetes/supernetes/agent/pkg/timestamp"
)

func NewCmdTimestamp() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "timestamp",
		Short: "Prepend RFC3339 timestamps to each stdin line and print to stdout",
		Long: dedent.Dedent(`
			Prepend RFC3339 timestamps to each stdin line and print to stdout. This is
			an internal command used by the Supernetes agent for processing log data.
		`),
		Run: func(cmd *cobra.Command, args []string) {
			timestamp.Run()
		},
	}

	return cmd
}
