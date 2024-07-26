// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vk

import (
	"context"

	"github.com/virtual-kubelet/virtual-kubelet/node"
	corev1 "k8s.io/api/core/v1"
)

// podInterface implements the RPC communication for workloads between the controller and agent
type podInterface struct{}

var _ node.PodLifecycleHandler = &podInterface{}

func (p podInterface) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	//TODO implement me
	panic("implement me")
}

func (p podInterface) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	//TODO implement me
	panic("implement me")
}

func (p podInterface) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	//TODO implement me
	panic("implement me")
}

func (p podInterface) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	//TODO implement me
	panic("implement me")
}

func (p podInterface) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (p podInterface) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	//TODO implement me
	panic("implement me")
}
