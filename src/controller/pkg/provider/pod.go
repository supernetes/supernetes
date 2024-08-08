// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// podKey is used to index Pod resources tracked by this provider
type podKey struct {
	name      string
	namespace string
}

func keyFor(pod *corev1.Pod) podKey {
	return podKey{
		name:      pod.Name,
		namespace: pod.Namespace,
	}
}

func keyFrom(name, namespace string) podKey {
	return podKey{
		name:      name,
		namespace: namespace,
	}
}

func (k *podKey) fields() map[string]interface{} {
	return map[string]interface{}{
		"name":      k.name,
		"namespace": k.namespace,
	}
}

func (k *podKey) String() string {
	return fmt.Sprintf("%s/%s", k.namespace, k.name)
}

var _ fmt.Stringer = &podKey{} // Static type assert

// podProvider implements the Virtual Kubelet pod lifecycle handler for Supernetes workloads
type podProvider struct {
	log  *zerolog.Logger
	pods map[podKey]*corev1.Pod
}

func NewPodProvider(log *zerolog.Logger) node.PodLifecycleHandler {
	return &podProvider{
		log:  log,
		pods: make(map[podKey]*corev1.Pod),
	}
}

func (p *podProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	// TODO: Implement
	key := keyFor(pod)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Debug().Msg("TODO CreatePod called")

	now := metav1.NewTime(time.Now())
	pod.Status = corev1.PodStatus{
		Phase: corev1.PodRunning,
		//HostIP:    "1.2.3.4",
		//PodIP:     "5.6.7.8",
		StartTime: &now,
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodInitialized,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
		},
	}

	for _, container := range pod.Spec.Containers {
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
			Name:         container.Name,
			Image:        container.Image,
			Ready:        true,
			RestartCount: 0,
			State: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: now,
				},
			},
		})
	}

	p.pods[key] = pod

	log.Debug().Msg("pod created")
	return nil
}

func (p *podProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	// TODO: Implement
	key := keyFor(pod)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Debug().Msg("TODO UpdatePod called")

	p.pods[key] = pod

	log.Debug().Msg("pod updated")
	return nil
}

func (p *podProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	// TODO: Implement
	key := keyFor(pod)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Debug().Msg("TODO DeletePod called")

	if _, ok := p.pods[key]; !ok {
		return errdefs.NotFound("pod not found")
	}

	now := metav1.Now()
	pod.Status.Phase = corev1.PodSucceeded
	pod.Status.Reason = "SupernetesPodDeleted"

	for i := range pod.Status.ContainerStatuses {
		pod.Status.ContainerStatuses[i].Ready = false
		pod.Status.ContainerStatuses[i].State = corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				Message:    "Supernetes terminated container upon deletion",
				FinishedAt: now,
				Reason:     "SupernetesPodContainerDeleted",
				StartedAt:  pod.Status.ContainerStatuses[i].State.Running.StartedAt,
			},
		}
	}

	delete(p.pods, key)

	log.Debug().Msg("pod deleted")
	return nil
}

func (p *podProvider) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	// TODO: Implement
	key := keyFrom(name, namespace)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Debug().Msg("TODO GetPod called")

	if pod, ok := p.pods[key]; ok {
		log.Debug().Msg("pod retrieved")
		return pod.DeepCopy(), nil
	}

	log.Error().Msg("unknown pod")
	return nil, errdefs.NotFoundf("unknown pod %q", key)
}

func (p *podProvider) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	// TODO: Implement
	key := keyFrom(name, namespace)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Debug().Msg("TODO GetPodStatus called")

	if pod, ok := p.pods[key]; ok {
		log.Debug().Msg("pod status retrieved")
		return pod.Status.DeepCopy(), nil
	}

	log.Error().Msg("unknown pod")
	return nil, errdefs.NotFoundf("unknown pod %q", key)
}

func (p *podProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	// TODO: Implement
	log := p.log
	log.Debug().Msg("TODO GetPods called")

	var pods []*corev1.Pod
	for _, pod := range p.pods {
		pods = append(pods, pod.DeepCopy())
	}

	log.Debug().Msgf("%d pod(s) retrieved", len(pods))
	return pods, nil
}
