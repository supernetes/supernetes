// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"sync"
	"time"

	dto "github.com/prometheus/client_model/go"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"k8s.io/utils/ptr"
)

// MetricsProvider implements the necessary node metrics for Metrics Server. Details here:
// https://github.com/kubernetes-sigs/metrics-server/blob/029dfa4e03b0f01b3ec4000ee25030e40511823f/pkg/scraper/client/resource/decode.go#L33-L34
type MetricsProvider interface {
	Update(status *api.NodeStatus) // Update the metrics based on the given status
	CpuMetric() *dto.MetricFamily  // Get the `node_cpu_usage_seconds_total` metric
	MemMetric() *dto.MetricFamily  // Get the `node_memory_working_set_bytes` metric
}

type metricsProvider struct {
	mutex       sync.Mutex
	coreSeconds float64
	workingSet  float64
	timestamp   time.Time
	loadAvg     float32
}

func NewMetricsProvider() MetricsProvider {
	return &metricsProvider{}
}

func (m *metricsProvider) Update(status *api.NodeStatus) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	diff := now.Sub(m.timestamp)

	// Integrate core seconds
	m.coreSeconds += float64(m.loadAvg) * diff.Seconds()
	m.workingSet = float64(status.WsBytes)

	m.timestamp = now
	m.loadAvg = status.CpuLoad
}

func (m *metricsProvider) CpuMetric() *dto.MetricFamily {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return &dto.MetricFamily{
		Name: ptr.To("node_cpu_usage_seconds_total"),
		Help: ptr.To("Cumulative cpu time consumed by the node in core-seconds"),
		Type: dto.MetricType_COUNTER.Enum(),
		Metric: []*dto.Metric{
			{
				Counter:     &dto.Counter{Value: ptr.To(m.coreSeconds)},
				TimestampMs: ptr.To(m.timestamp.UnixMilli()),
			},
		},
	}
}

func (m *metricsProvider) MemMetric() *dto.MetricFamily {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return &dto.MetricFamily{
		Name: ptr.To("node_memory_working_set_bytes"),
		Help: ptr.To("Current working set of the node in bytes"),
		Type: dto.MetricType_GAUGE.Enum(),
		Metric: []*dto.Metric{
			{
				Gauge:       &dto.Gauge{Value: ptr.To(m.workingSet)},
				TimestampMs: ptr.To(m.timestamp.UnixMilli()),
			},
		},
	}
}
