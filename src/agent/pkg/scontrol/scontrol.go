// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scontrol

import (
	"os/exec"

	"github.com/supernetes/supernetes/common/pkg/log"
)

// Run executes an `scontrol` command with the given arguments, returning JSON bytes
func Run(args ...string) ([]byte, error) {
	args = append([]string{"--json"}, args...)
	log.Trace().Strs("args", args).Msg("invoking scontrol")
	cmd := exec.Command("scontrol", args...)
	return cmd.Output()
}
