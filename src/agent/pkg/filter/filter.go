// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package filter

import "github.com/supernetes/supernetes/config/pkg/config"

// Filter configuration for retrieving nodes and jobs
type Filter interface {
	Partition(string) bool // Match Slurm partition
	Node(string) bool      // Match node name
}

// config.Filter should be compliant with Filter
var _ Filter = &config.Filter{}
