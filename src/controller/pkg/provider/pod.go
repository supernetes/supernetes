// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"fmt"
	"sync"
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

// TODO: Another design consideration is what to do with Pod deletions. They should probably be essentially no-ops,
//  i.e., the deletion will cause the Pod to be removed from podProvider's tracking, but the deletion request sent to
//  the agent is just an `scancel`. Upon job reconciliation, if Slurm still tracks the job, the Pod is going to be re-
//  created (with a "Completed") status. Once a job is actually removed from Slurm tracking, the reconciler will also
//  remove the associated Pod.

type PodProvider interface {
	node.PodLifecycleHandler
	node.PodNotifier // Required for async provider compliance
	// UpdateStatus is an additional handler for asynchronously updating the status of a Pod
	UpdateStatus(ctx context.Context, pod *corev1.Pod) error
}

// podProvider implements the Virtual Kubelet pod lifecycle handler for Supernetes workloads
type podProvider struct {
	log           *zerolog.Logger
	pods          map[podKey]*corev1.Pod
	pendingStatus map[podKey]*corev1.PodStatus
	notifier      func(*corev1.Pod)
	mutex         sync.Mutex
}

func NewPodProvider(log *zerolog.Logger) PodProvider {
	return &podProvider{
		log:           log,
		pods:          make(map[podKey]*corev1.Pod),
		pendingStatus: make(map[podKey]*corev1.PodStatus),
	}
}

// NotifyPods should be called (by VK logic) before any other operations
func (p *podProvider) NotifyPods(_ context.Context, notifier func(*corev1.Pod)) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.notifier = notifier
}

func (p *podProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	key := keyFor(pod)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Trace().Msg("CreatePod called")

	if status, ok := p.pendingStatus[key]; ok {
		pod.Status = *status
		delete(p.pendingStatus, key)
		log.Trace().Msg("loaded pending pod status")
	}

	changePhase(pod, pod.Status.Phase)
	pod.Status.Message = "Supernetes workload was created"

	p.pods[key] = pod
	p.notifier(pod)

	log.Trace().Msg("pod created")
	return nil
}

func (p *podProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	key := keyFor(pod)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Trace().Msg("UpdatePod called")

	changePhase(pod, pod.Status.Phase) // Use existing phase of pod
	pod.Status.Message = "Supernetes workload was updated"

	p.pods[key] = pod
	p.notifier(pod)

	log.Trace().Msg("pod updated")
	return nil
}

func (p *podProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	key := keyFor(pod)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Trace().Msg("DeletePod called")

	if _, ok := p.pods[key]; !ok {
		log.Trace().Msg("unknown pod")
		return errdefs.NotFoundf("unknown pod %q", key)
	}

	changePhase(pod, corev1.PodSucceeded)
	pod.Status.Message = "Supernetes workload was deleted"

	delete(p.pods, key)
	p.notifier(pod)

	log.Trace().Msg("pod deleted")
	return nil
}

func (p *podProvider) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	key := keyFrom(name, namespace)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Trace().Msg("GetPod called")

	if pod, ok := p.pods[key]; ok {
		log.Trace().Msg("pod retrieved")
		return pod.DeepCopy(), nil
	}

	log.Trace().Msg("unknown pod")
	return nil, errdefs.NotFoundf("unknown pod %q", key)
}

func (p *podProvider) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	key := keyFrom(name, namespace)
	log := p.log.With().Fields(key.fields()).Logger()
	log.Trace().Msg("GetPodStatus called")

	if pod, ok := p.pods[key]; ok {
		log.Trace().Msg("pod status retrieved")
		return pod.Status.DeepCopy(), nil
	}

	log.Trace().Msg("unknown pod")
	return nil, errdefs.NotFoundf("unknown pod %q", key)
}

func (p *podProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	log := p.log
	log.Trace().Msg("GetPods called")

	pods := make([]*corev1.Pod, 0, len(p.pods))
	for _, pod := range p.pods {
		pods = append(pods, pod.DeepCopy())
	}

	log.Trace().Msgf("%d pod(s) retrieved", len(pods))
	return pods, nil
}

// UpdateStatus updates the status of the given Pod in the provider
func (p *podProvider) UpdateStatus(ctx context.Context, updatedPod *corev1.Pod) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	key := keyFor(updatedPod)
	log := p.log.With().Fields(key.fields()).Logger()

	pod, ok := p.pods[key]
	if !ok {
		log.Trace().Msg("pod not found, caching status")
		p.pendingStatus[key] = &updatedPod.Status
		return nil
	}

	if pod.Status.Phase != updatedPod.Status.Phase {
		pod.Status = updatedPod.Status
		changePhase(pod, pod.Status.Phase)
		pod.Status.Message = "Supernetes workload status was updated"

		p.pods[key] = pod
		p.notifier(pod)

		log.Trace().Msg("pod status updated")
		return nil
	}

	return nil
}

func changePhase(pod *corev1.Pod, phase corev1.PodPhase) {
	pod.Status.Phase = phase

	condition := corev1.ConditionFalse
	if phase == corev1.PodRunning {
		condition = corev1.ConditionTrue
	}

	pod.Status.Conditions = []corev1.PodCondition{
		{
			Type:   corev1.ContainersReady,
			Status: condition,
		},
		{
			Type:   corev1.PodInitialized,
			Status: corev1.ConditionTrue, // No init containers
		},
		{
			Type:   corev1.PodReady,
			Status: condition,
		},
		{
			Type:   corev1.PodScheduled,
			Status: corev1.ConditionTrue, // Scheduling has succeeded if we've reached this point
		},
	}

	// Helper for transferring over existing container status
	type containerStatus struct {
		startTime    metav1.Time
		restartCount int32
	}

	containerStatuses := make(map[string]containerStatus)
	for i := range pod.Status.ContainerStatuses {
		status := &pod.Status.ContainerStatuses[i]

		var startTime metav1.Time
		if status.State.Running != nil {
			startTime = status.State.Running.StartedAt
		}

		containerStatuses[status.Name] = containerStatus{
			startTime:    startTime,
			restartCount: status.RestartCount,
		}
	}

	pod.Status.ContainerStatuses = make([]corev1.ContainerStatus, 0, len(pod.Spec.Containers))
	now := metav1.NewTime(time.Now())
	for _, container := range pod.Spec.Containers {
		status := corev1.ContainerStatus{
			Name:  container.Name,
			Image: container.Image,
			Ready: phase == corev1.PodRunning,
		}

		// Transfer over existing status
		var startTime metav1.Time
		if s, ok := containerStatuses[container.Name]; ok {
			startTime = s.startTime
			status.RestartCount = s.restartCount
		}

		switch phase {
		case corev1.PodPending:
			status.State.Waiting = &corev1.ContainerStateWaiting{
				Reason:  "Pending",
				Message: "Supernetes workload pending",
			}
		case corev1.PodRunning:
			status.State.Running = &corev1.ContainerStateRunning{
				StartedAt: now,
			}
		case corev1.PodSucceeded, corev1.PodFailed:
			var exitCode int32
			var reason = "Completed"
			if phase != corev1.PodSucceeded {
				exitCode = 1
				reason = "Error"
			}

			status.State.Terminated = &corev1.ContainerStateTerminated{
				ExitCode:   exitCode,
				Message:    "Supernetes workload terminated",
				FinishedAt: now,
				Reason:     reason,
				StartedAt:  startTime,
			}
		}

		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, status)
	}
}
