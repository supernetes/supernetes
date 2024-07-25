// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vk

import (
	"fmt"

	api "github.com/supernetes/supernetes/api/v1alpha1"
	"k8s.io/client-go/kubernetes"
)

// Manager manages Virtual Kubelet instances
type Manager struct {
	k8sInterface kubernetes.Interface
	instances    map[string]*instance
}

func NewManager() (*Manager, error) {
	k8sInterface, err := newK8sInterface()
	if err != nil {
		return nil, fmt.Errorf("failed to create K8s interface: %v", err)
	}

	return &Manager{
		k8sInterface: k8sInterface,
		instances:    make(map[string]*instance),
	}, nil
}

func (m *Manager) Reconcile(nodeList []*api.Node) error {
	// Untrack everything
	for _, instance := range m.instances {
		instance.tracked = false
	}

	for _, node := range nodeList {
		if instance, ok := m.instances[node.Meta.Name]; ok {
			// Existing node, still tracked
			instance.tracked = true
		} else {
			// New node, spawn new instance for it
			m.instances[node.Meta.Name] = newInstance(m.k8sInterface, node)
		}
	}

	// Stop and remove all instances that are no longer tracked
	for name, instance := range m.instances {
		if !instance.tracked {
			instance.cancel()
			delete(m.instances, name)
		}
	}

	return nil
}
