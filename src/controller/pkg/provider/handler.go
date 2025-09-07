// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
	"github.com/supernetes/supernetes/api/v1alpha1"
	suerr "github.com/supernetes/supernetes/common/pkg/error"
	sulog "github.com/supernetes/supernetes/common/pkg/log"
	vkapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	statsv1alpha1 "k8s.io/kubelet/pkg/apis/stats/v1alpha1"
	"k8s.io/utils/ptr"
)

func (p *podProvider) PodHandlerConfig() vkapi.PodHandlerConfig {
	return vkapi.PodHandlerConfig{
		RunInContainer:     p.RunInContainer,
		AttachToContainer:  p.AttachToContainer,
		PortForward:        p.PortForward,
		GetContainerLogs:   p.GetContainerLogs,
		GetPods:            p.GetPods,
		GetStatsSummary:    p.GetStatsSummary,
		GetMetricsResource: p.GetMetricsResource,
	}
}

type logStream grpc.BidiStreamingClient[v1alpha1.WorkloadLogRequest, v1alpha1.WorkloadLogChunk]

// logReader buffers traffic between the workload stream and the calling reader
type logReader struct {
	opts   vkapi.ContainerLogOpts
	chunks chan *v1alpha1.WorkloadLogChunk
	cache  []byte
	count  int
	err    atomic.Pointer[error]
	stream logStream
	closed atomic.Bool
	log    *zerolog.Logger
}

func startReader(stream logStream, opts vkapi.ContainerLogOpts, log *zerolog.Logger) *logReader {
	l := &logReader{
		opts:   opts,
		chunks: make(chan *v1alpha1.WorkloadLogChunk, 1024),
		stream: stream,
		log:    log,
	}

	go func() {
		log.Trace().Msg("log stream receiver started")
		defer log.Trace().Msg("log stream receiver stopped")
		defer func() { _ = l.Close() }()
		defer close(l.chunks)

		for {
			chunk, err := l.stream.Recv()
			if err != nil {
				l.err.Store(&err)
				return
			}

			l.chunks <- chunk
		}
	}()
	return l
}

func (l *logReader) Read(p []byte) (int, error) {
	var n int

	defer func() {
		// Handle LimitBytes
		l.count += n
		delta := l.count - l.opts.LimitBytes
		if l.opts.LimitBytes > 0 && delta > 0 {
			// Output must be delta bytes shorter to not exceed the limit
			p = p[:len(p)-delta]
			n -= delta
			_ = l.Close()
		}
	}()

	// If there's cached data, use it first
	if len(l.cache) > 0 {
		n += copy(p, l.cache)
		l.cache = l.cache[n:]
	}

	// If the cache filled p, return early.
	// This also applies if len(p) == 0.
	if n == len(p) {
		return n, nil
	}

	var c *v1alpha1.WorkloadLogChunk

	for {
		var more bool

		// Receive a line
		c, more = <-l.chunks
		if !more {
			err := *l.err.Load()
			if err != nil {
				// Clear the cache, we don't want to send anything after an error
				l.cache = nil

				// EOF and context cancellation are expected
				if !errors.Is(err, io.EOF) && !suerr.IsContextCanceled(err) {
					l.log.Err(err).Msg("streaming logs failed")
				}
			}

			return n, err // Channel closed, stop
		}

		if l.opts.Tail > 0 {
			break // Tailing a line count overrides sinceSeconds and sinceTime
		}

		// Agent-side timestamp of log chunk
		timestamp := c.Timestamp.AsTime()

		if l.opts.SinceSeconds > 0 {
			if time.Since(timestamp) > time.Duration(l.opts.SinceSeconds)*time.Second {
				continue // Log line too old, skip it
			}
		}

		if l.opts.SinceTime.Compare(timestamp) > 0 {
			continue // Log line too old, skip it
		}

		break
	}

	var b []byte

	// Prepend timestamps if requested
	if l.opts.Timestamps {
		b = append(b, []byte(c.Timestamp.AsTime().Format(time.RFC3339)+" ")...)
	}

	b = append(append(b, c.Line...), '\n')
	n2 := copy(p, b)
	n += n2

	// Some of the line didn't fit, append it to the cache
	if n2 < len(b) {
		l.cache = append(l.cache, b[n2:]...)
	}

	//log.Trace().Bytes("bytes", p[:n]).Int("n", n).Msg("sending bytes")
	return n, nil
}

func (l *logReader) Close() error {
	if l.closed.CompareAndSwap(false, true) {
		l.log.Trace().Msg("log stream reader closed")
		return l.stream.CloseSend()
	}

	return nil
}

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *podProvider) GetContainerLogs(ctx context.Context, namespace, podName, _ string, opts vkapi.ContainerLogOpts) (io.ReadCloser, error) {
	log := sulog.Scoped().Str("namespace", namespace).Str("pod", podName).Logger()
	log.Trace().Msg("GetContainerLogs called")

	pod, err := p.GetPod(ctx, namespace, podName)
	if err != nil {
		return nil, err
	}

	// Note: opts.Previous is described in `kubectl logs --help` as follows: "If true, print the logs for the previous
	// instance of the container in a pod if it exists." This is probably referring to when the container or Pod is
	// restarted with the Pod resource itself still referring to the same instance. This is not relevant for Supernetes,
	// since once the workload associated with a Pod completes, successfully or not, it can't be restarted. Slurm jobs
	// are one-shot by definition, and can only be rescheduled by resubmitting them, leading to a new  workload
	// identifier that must be associated with a fresh Pod.

	stream, err := p.workloadClient.Logs(ctx)
	if err != nil {
		return nil, err
	}

	if err := stream.Send(&v1alpha1.WorkloadLogRequest{
		Meta:   workloadMeta(pod),
		Follow: opts.Follow,
		Tail:   int32(opts.Tail),
	}); err != nil {
		return nil, err
	}

	return startReader(stream, opts, &log), nil
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *podProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach vkapi.AttachIO) error {
	sulog.Error().Msg("RunInContainer unimplemented")
	return fmt.Errorf("unimplemented")
}

// AttachToContainer attaches to the executing process of a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *podProvider) AttachToContainer(ctx context.Context, namespace, podName, containerName string, attach vkapi.AttachIO) error {
	sulog.Error().Msg("AttachToContainer unimplemented")
	return fmt.Errorf("unimplemented")
}

// GetStatsSummary gets the stats for the node, including running pods
func (p *podProvider) GetStatsSummary(context.Context) (*statsv1alpha1.Summary, error) {
	sulog.Trace().Msg("GetStatsSummary stub")
	return &statsv1alpha1.Summary{
		Node: statsv1alpha1.NodeStats{
			NodeName:  p.nodeName,
			StartTime: metav1.NewTime(time.Now().Add(-time.Hour)),
			CPU: &statsv1alpha1.CPUStats{
				Time:                 metav1.Now(),
				UsageNanoCores:       ptr.To(uint64(1000)),
				UsageCoreNanoSeconds: ptr.To(uint64(1000000)),
			},
		},
	}, nil
}

// GetMetricsResource gets the metrics for the node, including running pods
func (p *podProvider) GetMetricsResource(context.Context) ([]*dto.MetricFamily, error) {
	// TODO: This is verbose, disable logging for now
	//sulog.Trace().Msg("GetMetricsResource called")

	// For Metrics Server, the metrics this needs to report are here:
	//  https://github.com/kubernetes-sigs/metrics-server/blob/029dfa4e03b0f01b3ec4000ee25030e40511823f/pkg/scraper/client/resource/decode.go#L33-L34
	//	- node_cpu_usage_seconds_total, which is an integral of cpu-seconds
	//	- node_memory_working_set_bytes, which is the total working set memory usage on the node
	// Helpful command for debugging:
	//  kubectl get --raw /api/v1/nodes/<node>/proxy/metrics/resource | grep node_cpu_usage_seconds_total
	return []*dto.MetricFamily{
		p.metricsProvider.CpuMetric(),
		p.metricsProvider.MemMetric(),
	}, nil
}

// PortForward forwards a local port to a port on the pod
func (p *podProvider) PortForward(ctx context.Context, namespace, pod string, port int32, stream io.ReadWriteCloser) error {
	sulog.Error().Msg("PortForward unimplemented")
	return fmt.Errorf("unimplemented")
}
