// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scontrol

import (
	"errors"
	"fmt"

	"github.com/supernetes/supernetes/common/pkg/log"
	"sigs.k8s.io/json"
)

// output is a helper type for acquiring the warnings and errors from an `scontrol --json` command output
type output interface {
	GetWarnings() []any
	GetErrors() []any
}

// Decode decodes a configuration struct from the given `scontrol` output JSON bytes
func Decode[T any, PT interface {
	output
	*T
}](input []byte) (PT, error) {
	var config T
	log.Trace().Str("target", fmt.Sprintf("%T", config)).Msg("unmarshalling bytes")
	strictErrs, unmarshalErr := json.UnmarshalStrict(input, &config)

	if strictErrs != nil || unmarshalErr != nil {
		return nil, errors.Join(append([]error{
			fmt.Errorf("unable to decode JSON input into %T", config),
			unmarshalErr,
		}, strictErrs...)...)
	}

	// Type gymnastics for interface compliance
	var configPtr PT = &config

	if errs := configPtr.GetErrors(); len(errs) > 0 {
		responseErrs := make([]error, len(errs))
		for _, e := range errs {
			responseErrs = append(responseErrs, fmt.Errorf("%v", e))
		}

		return nil, errors.Join(append([]error{
			errors.New("errors in scontrol response"),
		}, responseErrs...)...)
	}

	if warns := configPtr.GetWarnings(); len(warns) > 0 {
		responseWarns := make([]string, len(warns))
		for _, w := range warns {
			responseWarns = append(responseWarns, fmt.Sprintf("%v", w))
		}

		log.Warn().Strs("warnings", responseWarns).Msg("warnings in scontrol response")
	}

	return configPtr, nil
}
