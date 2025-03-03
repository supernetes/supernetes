// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"os"

	"github.com/supernetes/supernetes/agent/cmd"
)

/*
TODO: Slurm node discovery and workload dispatching
 - scontrol show partition [standard] --json
 - scontrol show node --json
 - sinfo -N --json (but this produces much more output without that much more information)
 - Helpful tool: https://mholt.github.io/json-to-go/
*/

func main() {
	if cmd.NewRootCommand().Execute() != nil {
		os.Exit(1)
	}
}
