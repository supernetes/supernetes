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
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/supernetes/supernetes/common/pkg/log"
	vkapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/api/statsv1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// GetContainerLogs retrieves the logs of a container by name from the provider.
func (p *podProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts vkapi.ContainerLogOpts) (io.ReadCloser, error) {
	log.Error().Msg("GetContainerLogs unimplemented")
	return nil, fmt.Errorf("unimplemented")
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *podProvider) RunInContainer(ctx context.Context, namespace, podName, containerName string, cmd []string, attach vkapi.AttachIO) error {
	log.Error().Msg("RunInContainer unimplemented")
	return fmt.Errorf("unimplemented")
}

// AttachToContainer attaches to the executing process of a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *podProvider) AttachToContainer(ctx context.Context, namespace, podName, containerName string, attach vkapi.AttachIO) error {
	log.Error().Msg("AttachToContainer unimplemented")
	return fmt.Errorf("unimplemented")
}

// GetStatsSummary gets the stats for the node, including running pods
func (p *podProvider) GetStatsSummary(context.Context) (*statsv1alpha1.Summary, error) {
	log.Warn().Msg("GetStatsSummary stub")
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
	log.Warn().Msg("GetMetricsResource stub")
	return []*dto.MetricFamily{
		{
			Name: ptr.To("cpu_usage_total"),
			Help: ptr.To("Total CPU usage"),
			Type: dto.MetricType_COUNTER.Enum(),
			Metric: []*dto.Metric{
				{
					Counter: &dto.Counter{Value: ptr.To(float64(1234))},
				},
			},
		},
	}, nil
}

// PortForward forwards a local port to a port on the pod
func (p *podProvider) PortForward(ctx context.Context, namespace, pod string, port int32, stream io.ReadWriteCloser) error {
	log.Error().Msg("PortForward unimplemented")
	return fmt.Errorf("unimplemented")
}
