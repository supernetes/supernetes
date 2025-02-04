// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cache

import (
	"os"
	"path"

	"github.com/supernetes/supernetes/common/pkg/log"
)

var cacheDir string

func Dir() string {
	if cacheDir == "" {
		var err error
		cacheDir, err = os.UserCacheDir()
		log.FatalErr(err).Msg("failed to resolve cache directory")
	}

	return path.Join(cacheDir, "supernetes")
}

func IoDir() string {
	return path.Join(Dir(), "io")
}
