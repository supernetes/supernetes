// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package node

import "github.com/supernetes/supernetes/agent/pkg/scontrol"

// ReadNodeData queries Slurm about the specified node ID, or if nil, all nodes
func ReadNodeData(nodeName *string) (*Data, error) {
	args := []string{"show", "node"}
	if nodeName != nil {
		args = append(args, *nodeName)
	}

	data, err := scontrol.Run(args...)
	if err != nil {
		return nil, err
	}

	return scontrol.Decode[Data](data)
}
