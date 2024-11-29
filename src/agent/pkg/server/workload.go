// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package server

import (
	"context"
	"errors"

	"github.com/supernetes/supernetes/agent/pkg/filter"
	"github.com/supernetes/supernetes/agent/pkg/job"
	"github.com/supernetes/supernetes/agent/pkg/sbatch"
	"github.com/supernetes/supernetes/agent/pkg/scancel"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// workloadServer is used to implement supernetes.WorkloadApiServer
type workloadServer struct {
	api.UnimplementedWorkloadApiServer
	runtime sbatch.Runtime
	filter  filter.Filter
}

func NewWorkloadServer(runtime sbatch.Runtime, filter filter.Filter) api.WorkloadApiServer {
	return &workloadServer{
		runtime: runtime,
		filter:  filter,
	}
}

func (s *workloadServer) Create(_ context.Context, workload *api.Workload) (*api.WorkloadMeta, error) {
	log.Debug().Stringer("workload", workload).Msg("Create invoked")
	jobId, err := s.runtime.Run(workload)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("id", jobId).Msg("job dispatched")

	// Update job id tracking label
	workload.Meta.Identifier = jobId
	return workload.Meta, nil
}

func (s *workloadServer) Update(ctx context.Context, workload *api.Workload) (*emptypb.Empty, error) {
	log.Debug().Stringer("workload", workload).Msg("Update invoked")

	return nil, status.Errorf(codes.Unimplemented, "method Update not implemented")
}

// Delete for Slurm just means cancelling the job, it's the best we can do
func (s *workloadServer) Delete(ctx context.Context, workload *api.WorkloadMeta) (*emptypb.Empty, error) {
	log.Debug().Stringer("workload", workload).Msg("Delete invoked")
	return nil, scancel.Run(workload)
}

func (s *workloadServer) Get(ctx context.Context, workloadMeta *api.WorkloadMeta) (*api.Workload, error) {
	log.Debug().Stringer("workloadMeta", workloadMeta).Msg("Get invoked")

	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}

func (s *workloadServer) GetStatus(ctx context.Context, workloadMeta *api.WorkloadMeta) (*api.WorkloadStatus, error) {
	log.Debug().Stringer("workloadMeta", workloadMeta).Msg("GetStatus invoked")

	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}

func (s *workloadServer) List(_ *emptypb.Empty, stream grpc.ServerStreamingServer[api.Workload]) error {
	log.Debug().Msg("List invoked")
	jobData, err := job.ReadJobData(nil)
	if err != nil {
		msg := "failed to read job data from Slurm"
		log.Err(err).Msg(msg)
		return errors.New(msg)
	}

	for i := range jobData.Jobs {
		j := &jobData.Jobs[i]
		if !s.filter.Partition(j.Partition) {
			continue // Job not in filtered partitions
		}

		workload := j.ConvertToApi(s.filter.Node)
		if err := stream.Send(workload); err != nil {
			return err
		}
	}

	log.Debug().
		Int("all", len(jobData.Jobs)).
		Msg("sent job list")

	return nil
}
