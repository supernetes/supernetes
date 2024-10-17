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

// TODO: Virtual Kubelet, through GetPod/CreatePod/etc. basically makes sure that the pods tracked by podProvider are
//  consistent with the API server. On startup, any pods that this contains but are not present (anymore) in the cluster
//  will be deleted, and any pods that should be tracked will be created on startup. This means that we should consider
//  the API server as the single source of truth instead of Slurm. That said, since jobs will also be created on the HPC
//  side by other users interacting with Slurm directly, we need some kind of reconciliation loop that periodically
//  requests the job list from the agent, and creates Pods based on that. Those pods will then obviously be picked up by
//  podProvider again, but if they already contain a job ID or something, this can just update their status and not
//  actually deploy anything.
//
// TODO: Another design consideration is what to do with Pod deletions. They should probably be essentially no-ops,
//  i.e., the deletion will cause the Pod to be removed from podProvider's tracking, but the deletion request sent to
//  the agent is just an `scancel`. Upon job reconciliation, if Slurm still tracks the job, the Pod is going to be re-
//  created (with a "Completed") status. Once a job is actually removed from Slurm tracking, the reconciler will also
//  remove the associated Pod.

// podProvider implements the Virtual Kubelet pod lifecycle handler for Supernetes workloads
type podProvider struct {
	log      *zerolog.Logger
	pods     map[podKey]*corev1.Pod
	notifier func(*corev1.Pod)
}

var _ node.PodNotifier = &podProvider{} // Required for async provider compliance

func NewPodProvider(log *zerolog.Logger) node.PodLifecycleHandler {
	return &podProvider{
		log:  log,
		pods: make(map[podKey]*corev1.Pod),
	}
}

// NotifyPods should be called (by VK logic) before any other operations
func (p *podProvider) NotifyPods(_ context.Context, notifier func(*corev1.Pod)) {
	p.notifier = notifier
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
	p.notifier(pod)

	log.Debug().Msg("pod created")
	return nil
}

func (p *podProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	// TODO: Implement
	key := keyFor(pod)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Debug().Msg("TODO UpdatePod called")

	p.pods[key] = pod
	p.notifier(pod)

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
	p.notifier(pod)

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

	log.Debug().Msg("unknown pod")
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

	log.Debug().Msg("unknown pod")
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
