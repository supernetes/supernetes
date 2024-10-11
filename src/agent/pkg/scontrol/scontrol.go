// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scontrol

func ReadNodeInfo() (*NodeInfo, error) {
	nodeInfoBytes, err := run("show", "node")
	if err != nil {
		return nil, err
	}

	// TODO: Also handle the error field in the JSON
	return decode[NodeInfo](nodeInfoBytes)
}
