// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scontrol

// Number represents a numeric value type in the `scontrol` JSON output
type Number struct {
	Set      bool    `json:"set"`
	Infinite bool    `json:"infinite"`
	Number   float32 `json:"number"`
}
