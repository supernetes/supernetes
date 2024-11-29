// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package job

import (
	"encoding/json"
)

//goland:noinspection GoNameStartsWithPackageName
type JobState string

// Compile-time type checking
var checkState = JobState("")
var _ json.Unmarshaler = &checkState

func (j *JobState) UnmarshalJSON(bytes []byte) error {
	// On LUMI, Job.jobs.job_state is a single string, while on Mahti, it is a string array.
	// Mahti's behavior matches the slurmrestd docs, but we still need to support both variants.
	var state string
	if err := json.Unmarshal(bytes, &state); err == nil {
		*j = JobState(state)
		return nil
	}

	var states []string
	if err := json.Unmarshal(bytes, &states); err != nil {
		return err
	}

	if len(states) > 0 {
		*j = JobState(states[0])
	} else {
		*j = "" // This is not a parsing error
	}

	return nil
}
