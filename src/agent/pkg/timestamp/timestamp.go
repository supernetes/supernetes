// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package timestamp

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/supernetes/supernetes/common/pkg/log"
)

// Run starts the stdin line timestamper
func Run() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Printf("%s %s\n", time.Now().Format(time.RFC3339), scanner.Text())
	}

	log.FatalErr(scanner.Err()).Msg("failed to scan stdin")
}
