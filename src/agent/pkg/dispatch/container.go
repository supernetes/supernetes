// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package dispatch

import (
	"os/exec"

	"github.com/pkg/errors"
	"github.com/supernetes/supernetes/common/pkg/log"
)

func containerRuntime() string {
	runtimes := []string{
		"singularity",
		"apptainer",
	}

	for _, runtime := range runtimes {
		log.Debug().Str("runtime", runtime).Msg("locating container runtime")
		path, err := exec.LookPath(runtime)
		if err == nil {
			log.Debug().Str("path", path).Msg("located container runtime")
			return path
		}

		if errors.Is(err, exec.ErrNotFound) {
			continue
		}

		log.Fatal().Err(err).Str("runtime", runtime).Msg("failed to locate container runtime")
	}

	log.Fatal().Strs("runtimes", runtimes).Msg("no container runtime found")
	return "" // never reached
}
