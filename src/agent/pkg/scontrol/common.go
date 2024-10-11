// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scontrol

import (
	"errors"
	"fmt"
	"os/exec"
	"sigs.k8s.io/json"
)

// Number represents a numeric value type in the `scontrol` JSON output
type Number struct {
	Set      bool `json:"set"`
	Infinite bool `json:"infinite"`
	Number   int  `json:"number"`
}

// run executes an `scontrol` command with the given arguments, returning JSON bytes
func run(args ...string) ([]byte, error) {
	cmd := exec.Command("scontrol", append([]string{"--json"}, args...)...)
	return cmd.Output()
}

// decode decodes a configuration struct from the given JSON bytes
func decode[T any](input []byte) (*T, error) {
	var config T
	strictErrs, unmarshalErr := json.UnmarshalStrict(input, &config)

	if strictErrs != nil || unmarshalErr != nil {
		return nil, errors.Join(append([]error{
			fmt.Errorf("unable to decode JSON input into %T", config),
			unmarshalErr}, strictErrs...)...)
	}

	return &config, nil
}
