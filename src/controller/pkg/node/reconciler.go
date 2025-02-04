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
	"github.com/supernetes/supernetes/controller/pkg/environment"
	"github.com/supernetes/supernetes/controller/pkg/reconciler"
	"github.com/supernetes/supernetes/controller/pkg/tracker"
	"github.com/supernetes/supernetes/controller/pkg/vk"
	vkauth "github.com/supernetes/supernetes/controller/pkg/vk/auth"
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
	Interval       time.Duration           // Reconciliation interval
	NodeClient     api.NodeApiClient       // Client for accessing the node API
	WorkloadClient api.WorkloadApiClient   // Client for accessing the workload API
	Tracker        tracker.Tracker         // Manager for tracked Pods
	KubeConfig     *rest.Config            // Configuration for accessing Kubernetes
	Environment    environment.Environment // Controller environment configuration
}

// nReconciler manages Virtual Kubelet instances
type nReconciler struct {
	ReconcilerConfig
	kubeClient kubernetes.Interface
	instances  map[string]*instance
	vkAuth     vkauth.Auth
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
	kubeClient, err := client.NewKubeClient(config.KubeConfig)
	if err != nil {
		return nil, err
	}

	logger := log.Scoped().Str("type", "node").Logger()
	vkAuth, err := vkauth.Start(ctx, kubeClient, &logger)
	if err != nil {
		return nil, err
	}

	nr := &nReconciler{
		ReconcilerConfig: config,
		kubeClient:       kubeClient,
		instances:        make(map[string]*instance),
		vkAuth:           vkAuth,
	}
	r, err := reconciler.New(ctx, &logger, config.Interval, nr)
	if err != nil {
		return nil, err
	}

	return &nReconcilerAdapter{Reconciler: r, reconciler: nr}, nil
}

func (r *nReconciler) Reconcile(ctx context.Context) error {
	stream, err := r.NodeClient.GetNodes(ctx, &emptypb.Empty{})
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
			// New node, create a new instance for it
			r.instances[node.Meta.Name] = newInstance(vk.NewInstance(vk.InstanceConfig{
				KubeClient:     r.kubeClient,
				Node:           node,
				WorkloadClient: r.WorkloadClient,
				Tracker:        r.Tracker,
				Environment:    r.Environment,
				VkAuth:         r.vkAuth,
			}))
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
