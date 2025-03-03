// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package server

import (
	"bufio"
	"bytes"
	"container/ring"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/supernetes/supernetes/agent/pkg/cache"
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
	"google.golang.org/protobuf/types/known/timestamppb"
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

func (s *workloadServer) Logs(stream grpc.BidiStreamingServer[api.WorkloadLogRequest, api.WorkloadLogChunk]) error {
	log.Debug().Msg("Logs invoked")
	request, err := stream.Recv()
	if err != nil {
		return err
	}

	log.Trace().Str("id", request.Meta.Identifier).Msg("received WorkloadMeta")

	if len(request.Meta.Identifier) == 0 {
		err := errors.New("missing job identifier")
		log.Err(err).Str("name", request.Meta.Name).Msg("log streaming failed")
		return err
	}

	// path.Join always finishes with a path.Clean
	filePath := path.Join(cache.IoDir(), fmt.Sprintf("%s.out", request.Meta.Identifier))

	// Prevent escape from I/O directory using a malicious workload identifier
	if !strings.HasPrefix(filePath, cache.IoDir()) {
		err := errors.New("invalid job identifier")
		log.Err(err).Str("id", request.Meta.Identifier).Msg("prevented filesystem escape")
		return err
	}

	lines := make(chan []byte)
	stop := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		err := tailF(ctx, filePath, lines, int(request.Tail), request.Follow)
		if err != nil {
			log.Err(err).Str("path", filePath).Msg("tailing log failed")
		}

		select {
		case stop <- err:
		default:
		}
	}()

	go func() {
		for {
			// Block here until the controller closes the sender
			_, err := stream.Recv()
			if err == nil {
				continue // Ignore further requests
			}

			select {
			case stop <- err:
			default:
			}

			break
		}
	}()

	for {
		// Logger for line parser
		logger := log.Scoped().Str("id", request.Meta.Identifier).Logger()

		select {
		case err := <-stop:
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				err = nil // EOF and context cancellation are expected
			}

			log.Trace().Err(err).Msg("stopping log streaming")
			return err
		case line := <-lines:
			// Parse the timestamp from the line
			timestamp, line := parseLine(line, &logger)
			//log.Trace().Time("timestamp", timestamp).Bytes("line", line).Msg("retrieved line")

			// Construct and send a log chunk
			if err := stream.Send(&api.WorkloadLogChunk{
				Timestamp: timestamppb.New(timestamp),
				Line:      line,
			}); err != nil {
				return err
			}
		}
	}
}

// Lustre doesn't support inotify: http://lists.lustre.org/pipermail/lustre-discuss-lustre.org/2019-May/016469.html
// `tail` automatically reverts to polling once per second when it encounters a non-local filesystem, let's do the same
func tailF(ctx context.Context, filePath string, lines chan<- []byte, n int, follow bool) error {
	immediate := make(chan struct{}, 1)
	immediate <- struct{}{}

	var err error
	var file *os.File
	var fileInfo os.FileInfo
	var scanner *bufio.Scanner

	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	// Whether to tail n latest lines
	tailn := n > 0

	for {
		select {
		case <-immediate: // Fire once immediately
		case <-ticker.C:
		case <-ctx.Done():
			return nil
		}

		if file != nil {
			// From tail(1) for `-F`: "The file is closed and reopened when tail
			// detects that the filename being read from has a new inode number."
			fileInfo2, err := os.Stat(filePath)
			if err != nil {
				return errors.Wrap(err, "failed to stat file")
			}

			// Re-open the file if it has changed
			if !os.SameFile(fileInfo, fileInfo2) {
				if err := file.Close(); err != nil {
					return errors.Wrap(err, "failed to close file")
				}

				file = nil
			}
		}

		// If we don't have a file, try to open it here. Truncation shouldn't matter,
		// since the writing application (Slurm) will always continue where it left off
		if file == nil {
			file, err = os.Open(filePath)
			if err != nil {
				if errors.Is(err, os.ErrNotExist) {
					continue // This needs to wait again
				}
				return errors.Wrap(err, "failed to open file")
			}

			if fileInfo, err = file.Stat(); err != nil {
				return errors.Wrap(err, "failed to stat file")
			}

			scanner = bufio.NewScanner(file)
		}

		if file != nil {
			var buffer *ring.Ring
			if tailn {
				// If tailing for a specific line count, allocate a ring buffer here for tracking the n latest lines
				buffer = ring.New(n)
			}

			for scanner.Scan() {
				if tailn {
					// Track n latest lines in the ring buffer
					buffer.Value = scanner.Bytes()
					buffer = buffer.Next()
				} else {
					lines <- scanner.Bytes()
				}
			}

			if tailn {
				// Send the n latest lines
				buffer.Do(func(line any) {
					if line != nil {
						lines <- line.([]byte)
					}
				})
			}

			if err := scanner.Err(); err != nil {
				return errors.Wrap(err, "failed to scan file")
			}

			// If following is not requested, we're done
			if !follow {
				return nil
			}

			// The scanner stops permanently at the first EOF, so we need to
			// re-create it here to read the newly appended data next round
			scanner = bufio.NewScanner(file)

			// Tailing n lines is no longer relevant when following
			tailn = false
		}
	}
}

func parseLine(line []byte, log *zerolog.Logger) (time.Time, []byte) {
	splits := bytes.SplitN(line, []byte(" "), 2)
	if len(splits) < 2 {
		output := bytes.Join(splits, nil) // Handles both length 0 and 1
		log.Warn().Bytes("line", output).Msg("log line without timestamp")
		return time.Time{}, output
	}

	timestamp, err := time.Parse(time.RFC3339, string(splits[0]))
	if err != nil {
		log.Err(err).Bytes("line", splits[1]).Msg("unable to parse timestamp for log line")
	}

	// Zero timestamp on failure
	return timestamp, splits[1]
}
