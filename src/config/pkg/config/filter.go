// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

// Partition filtering helper for Filter
func (f *Filter) Partition(input string) bool {
	if f == nil {
		return true // No filtering
	}

	return f.PartitionRegex.MatchString(input)
}

// Node filtering helper for Filter
func (f *Filter) Node(input string) bool {
	if f == nil {
		return true // No filtering
	}

	return f.NodeRegex.MatchString(input)
}
