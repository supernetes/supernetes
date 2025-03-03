// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package cmd

// Helper for handling fatal errors before logger initialization
func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
