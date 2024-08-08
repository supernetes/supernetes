// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package controller

import (
	"context"

	"github.com/pkg/errors"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/controller/pkg/vk"
	"k8s.io/client-go/kubernetes"
)

type instance struct {
	tracked  bool
	instance vk.Instance
	ctx      context.Context
	cancel   func()
}

func newInstance(ctx context.Context, i vk.Instance) *instance {
	ctx, cancel := context.WithCancel(ctx)
	return &instance{
		tracked:  true, // New instances are always tracked
		instance: i,
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (i *instance) start() error {
	return i.instance.Run(i.ctx)
}

func (i *instance) stop() {
	i.cancel()
}

// Manager manages Virtual Kubelet instances
type Manager struct {
	k8sInterface kubernetes.Interface
	instances    map[string]*instance
}

func NewManager(k8sInterface kubernetes.Interface) *Manager {
	return &Manager{
		k8sInterface: k8sInterface,
		instances:    make(map[string]*instance),
	}
}

func (m *Manager) Reconcile(nodeList []*api.Node) error {
	// Untrack everything
	for _, i := range m.instances {
		i.tracked = false
	}

	for _, node := range nodeList {
		if i, ok := m.instances[node.Meta.Name]; ok {
			// Existing node, still tracked
			i.tracked = true
		} else {
			// New node, spawn a new instance for it
			i := newInstance(context.Background(), vk.NewInstance(m.k8sInterface, node))
			if err := i.start(); err != nil {
				// TODO: Should this be fatal or not? -> Probably fatal, since then it will be tracked in restarts
				return errors.Wrap(err, "starting instance failed")
			}

			m.instances[node.Meta.Name] = i
		}
	}

	// Stop and remove all instances that are no longer tracked
	for name, i := range m.instances {
		if !i.tracked {
			i.stop()
			delete(m.instances, name)
		}
	}

	return nil
}
