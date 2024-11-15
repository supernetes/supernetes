// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package job

import (
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/supernetes/supernetes/agent/pkg/agent"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
)

func (j *Job) ConvertToApi(nodeFilter func(string) bool) *api.Workload {
	// Resolve the nodes where the job is running
	// TODO: This might not be 100% reliable for all jobs
	nodes := make([]*api.NodeMeta, 0, len(j.JobResources.AllocatedNodes))
	for i := range j.JobResources.AllocatedNodes {
		name := j.JobResources.AllocatedNodes[i].Nodename
		if !nodeFilter(name) {
			continue // Node was excluded by filter
		}

		nodes = append(nodes, &api.NodeMeta{Name: name})
	}

	workload := &api.Workload{
		Meta: &api.WorkloadMeta{
			Name:       j.Name,
			Identifier: strconv.Itoa(j.JobID),
			Extra:      map[string]string{"job-state": string(j.JobState)},
			//Labels: labels,
		},
		//Spec: &api.WorkloadSpec{
		//	Image: image,
		//	Args:  args,
		//},
		Status: &api.WorkloadStatus{
			Phase:     parseJobState(j.JobState),
			StdOut:    readIo(j.JobID, "stout"),
			StdErr:    readIo(j.JobID, "stderr"),
			StartTime: int64(j.StartTime.Number),
			Nodes:     nodes,
		},
	}

	return workload
}

func readIo(jobId int, kind string) string {
	data, err := os.ReadFile(path.Join(agent.IoDir(), fmt.Sprintf("%d.%s", jobId, kind)))
	if os.IsNotExist(err) {
		return ""
	}

	if err != nil {
		log.Err(err).Int("id", jobId).Str("kind", kind).Msg("failed to read I/O file for job")
		return ""
	}

	return string(data)
}

func parseJobState(jobState JobState) api.WorkloadPhase {
	// `scontrol` job state codes: https://slurm.schedmd.com/squeue.html#SECTION_JOB-STATE-CODES
	switch jobState {
	case "BF", "BOOT_FAIL":
		return api.WorkloadPhase_Failed
	case "CA", "CANCELLED":
		return api.WorkloadPhase_Failed
	case "CD", "COMPLETED":
		return api.WorkloadPhase_Succeeded
	case "CF", "CONFIGURING":
		return api.WorkloadPhase_Pending
	case "CG", "COMPLETING":
		return api.WorkloadPhase_Running
	case "DL", "DEADLINE":
		return api.WorkloadPhase_Failed
	case "F", "FAILED":
		return api.WorkloadPhase_Failed
	case "NF", "NODE_FAIL":
		return api.WorkloadPhase_Failed
	case "OOM", "OUT_OF_MEMORY":
		return api.WorkloadPhase_Failed
	case "PD", "PENDING":
		return api.WorkloadPhase_Pending
	case "PR", "PREEMPTED":
		return api.WorkloadPhase_Failed
	case "R", "RUNNING":
		return api.WorkloadPhase_Running
	case "RD", "RESV_DEL_HOLD":
		return api.WorkloadPhase_Pending
	case "RF", "REQUEUE_FED":
		return api.WorkloadPhase_Pending
	case "RH", "REQUEUE_HOLD":
		return api.WorkloadPhase_Pending
	case "RQ", "REQUEUED":
		return api.WorkloadPhase_Pending
	case "RS", "RESIZING":
		return api.WorkloadPhase_Pending
	case "RV", "REVOKED":
		return api.WorkloadPhase_Unknown // It's not clear whether the job is running here anymore
	case "SI", "SIGNALING":
		return api.WorkloadPhase_Pending
	case "SE", "SPECIAL_EXIT":
		return api.WorkloadPhase_Succeeded
	case "SO", "STAGE_OUT":
		return api.WorkloadPhase_Pending
	case "ST", "STOPPED":
		return api.WorkloadPhase_Pending
	case "S", "SUSPENDED":
		return api.WorkloadPhase_Pending
	case "TO", "TIMEOUT":
		return api.WorkloadPhase_Failed
	default:
		return api.WorkloadPhase_Unknown
	}
}
