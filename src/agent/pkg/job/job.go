// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package job

import "github.com/supernetes/supernetes/agent/pkg/scontrol"

// ReadJobData queries Slurm about the specified job ID, or if nil, all jobs
func ReadJobData(jobId *string) (*Data, error) {
	args := []string{"show", "job"}
	if jobId != nil {
		args = append(args, *jobId)
	}

	data, err := scontrol.Run(args...)
	if err != nil {
		return nil, err
	}

	return scontrol.Decode[Data](data)
}
