// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scontrol

import (
	"encoding/json"
	"math"
	"strconv"
)

// Compile-time type checking
var _ json.Unmarshaler = &Number{}

func (n *Number) UnmarshalJSON(bytes []byte) error {
	// The `scontrol` output is incredibly inconsistent, even across just LUMI and Mahti the same
	// fields are sometimes ints, sometimes floats, and sometimes these scontrol.Number structs.
	// By implementing fallback parsing from float32 here we should have covered all bases.
	var number float32
	if err := json.Unmarshal(bytes, &number); err == nil {
		*n = Number{
			Set:      number > 0,                     // Assume set if positive
			Infinite: math.IsInf(float64(number), 1), // Test for positive infinity
			Number:   number,
		}

		return nil
	}

	type JsonNumber Number // Helper type to avoid infinite recursion
	var jsonNumber JsonNumber

	if err := json.Unmarshal(bytes, &jsonNumber); err != nil {
		return err
	}

	*n = Number(jsonNumber)
	return nil
}

// ToFloat converts the special number type to a regular float
func (n *Number) ToFloat() float32 {
	if !n.Set {
		return float32(math.NaN())
	}

	if n.Infinite {
		negative := 1
		if n.Number < 0 {
			negative = -1
		}

		return float32(math.Inf(negative))
	}

	return n.Number
}

// Compile-time type checking
var _ json.Unmarshaler = &Version{}

func (v *Version) UnmarshalJSON(bytes []byte) error {
	// On LUMI, Data.meta.slurm.version contains integer fields instead of the string fields
	// provided by Mahti and the slurmrestd API. To support both, do the conversion here.
	type IntVersion struct {
		Major int `json:"major"`
		Minor int `json:"minor"`
		Micro int `json:"micro"`
	}

	var intVersion IntVersion
	if err := json.Unmarshal(bytes, &intVersion); err == nil {
		*v = Version{
			Major: strconv.Itoa(intVersion.Major),
			Minor: strconv.Itoa(intVersion.Minor),
			Micro: strconv.Itoa(intVersion.Micro),
		}

		return nil
	}

	type JsonVersion Version // Helper type to avoid infinite recursion
	var jsonVersion JsonVersion

	if err := json.Unmarshal(bytes, &jsonVersion); err != nil {
		return err
	}

	*v = Version(jsonVersion)
	return nil
}
