// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package node

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/controller/pkg/client"
	"github.com/supernetes/supernetes/controller/pkg/metrics"
	"github.com/supernetes/supernetes/controller/pkg/reconciler"
	"github.com/supernetes/supernetes/controller/pkg/tracker"
	"github.com/supernetes/supernetes/controller/pkg/vk"
	"google.golang.org/protobuf/types/known/emptypb"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type instance struct {
	tracked  bool
	instance vk.Instance
	ctx      context.Context
	cancel   func()
}

func newInstance(i vk.Instance) *instance {
	return &instance{
		tracked:  true, // New instances are always tracked
		instance: i,
	}
}

func (i *instance) start(ctx context.Context) {
	if i.ctx == nil || i.ctx.Err() != nil {
		i.ctx, i.cancel = context.WithCancel(ctx)
	} else {
		return // Already running
	}

	go func() {
		if err := i.instance.Run(i.ctx, i.cancel); err != nil && !errors.Is(err, context.Canceled) {
			log.Err(err).Msg("failed to start Virtual Kubelet instance")
		}
	}()
}

func (i *instance) stop() {
	if i.ctx != nil {
		i.cancel()
	}
}

type ReconcilerConfig struct {
	Interval       time.Duration         // Reconciliation interval
	NodeClient     api.NodeApiClient     // Client for accessing the node API
	WorkloadClient api.WorkloadApiClient // Client for accessing the workload API
	K8sConfig      *rest.Config          // Configuration for accessing Kubernetes
	Tracker        tracker.Tracker       // Manager for tracked Pods
	Metrics        metrics.Config        // Metrics configuration interface
}

// nReconciler manages Virtual Kubelet instances
type nReconciler struct {
	nodeClient     api.NodeApiClient
	workloadClient api.WorkloadApiClient
	k8sClient      kubernetes.Interface
	tracker        tracker.Tracker
	metrics        metrics.Config
	instances      map[string]*instance
}

// nReconcilerAdapter is a helper for adding additional methods to nReconciler
type nReconcilerAdapter struct {
	reconciler.Reconciler
	reconciler *nReconciler
}

type Reconciler interface {
	reconciler.Reconciler
	tracker.StatusUpdater
}

func NewReconciler(ctx context.Context, config ReconcilerConfig) (Reconciler, error) {
	k8sClient, err := client.NewK8sClient(config.K8sConfig)
	if err != nil {
		return nil, err
	}

	logger := log.Scoped().Str("type", "node").Logger()
	nr := &nReconciler{
		nodeClient:     config.NodeClient,
		workloadClient: config.WorkloadClient,
		k8sClient:      k8sClient,
		tracker:        config.Tracker,
		metrics:        config.Metrics,
		instances:      make(map[string]*instance),
	}
	r, err := reconciler.New(ctx, &logger, config.Interval, nr)
	if err != nil {
		return nil, err
	}

	return &nReconcilerAdapter{Reconciler: r, reconciler: nr}, nil
}

func (r *nReconciler) Reconcile(ctx context.Context) error {
	stream, err := r.nodeClient.GetNodes(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}

	// Untrack everything
	for _, i := range r.instances {
		i.tracked = false
	}

	for {
		// Query the agent for nodes
		node, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if i, ok := r.instances[node.Meta.Name]; ok {
			// Existing node, still tracked
			// TODO: Need to check here whether the node actually still exists as a resource. If not, the most atomic
			//  way to get it back is to re-create the instance.
			i.tracked = true
		} else {
			config := vk.InstanceConfig{
				Client:         r.k8sClient,
				Node:           node,
				WorkloadClient: r.workloadClient,
				Tracker:        r.tracker,
				Metrics:        r.metrics,
			}

			// New node, create a new instance for it
			r.instances[node.Meta.Name] = newInstance(vk.NewInstance(config))
		}
	}

	// Start/stop tracked/untracked instances
	for _, i := range r.instances {
		if i.tracked {
			i.start(ctx)
		} else {
			i.stop()
		}
	}

	// Remove all instances that are no longer tracked
	for name, i := range r.instances {
		if !i.tracked {
			delete(r.instances, name)
		}
	}

	return nil
}

func (r *nReconcilerAdapter) UpdateStatus(ctx context.Context, pod *corev1.Pod, cache bool) error {
	if pod.Spec.NodeName == "" {
		return nil // Pod is not scheduled onto any node
	}

	if instance, ok := r.reconciler.instances[pod.Spec.NodeName]; ok {
		return instance.instance.UpdateStatus(ctx, pod, cache)
	}

	return nil // Pod is associated with unknown node
}
