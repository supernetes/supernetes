// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scancel

import (
	"os/exec"

	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
)

func Run(meta *api.WorkloadMeta) error {
	log.Trace().Str("id", meta.Identifier).Msg("invoking scancel")
	cmd := exec.Command("scancel", meta.Identifier)
	err := cmd.Run()
	if err != nil {
		log.Err(err).Msg("failed to cancel job")
		return err
	}

	return nil
}
