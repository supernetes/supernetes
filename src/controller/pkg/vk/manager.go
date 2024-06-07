// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vk

import (
	"fmt"

	"github.com/supernetes/supernetes/api"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// Manager manages Virtual Kubelet instances
type Manager struct {
	k8sInterface corev1.CoreV1Interface
	instances    map[string]*instance
}

func NewManager() (*Manager, error) {
	k8sInterface, err := newCoreV1Interface()
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
		if instance, ok := m.instances[node.Name]; ok {
			// Existing node, still tracked
			instance.tracked = true
		} else {
			// New node, spawn new instance for it
			m.instances[node.Name] = newInstance(m.k8sInterface.Nodes(), node)
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
