// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
	"github.com/supernetes/supernetes/controller/pkg/tracker"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	corev1 "k8s.io/api/core/v1"
)

// TODO: Another design consideration is what to do with Pod deletions. They should probably be essentially no-ops,
//  i.e., the deletion will cause the Pod to be removed from podProvider's tracking, but the deletion request sent to
//  the agent is just an `scancel`. Upon job reconciliation, if Slurm still tracks the job, the Pod is going to be re-
//  created (with a "Completed") status. Once a job is actually removed from Slurm tracking, the reconciler will also
//  remove the associated Pod.

type PodProvider interface {
	nodeutil.Provider
	node.PodNotifier // Required for async provider compliance
	tracker.StatusUpdater
}

// podProvider implements the Virtual Kubelet pod lifecycle handler for Supernetes workloads
type podProvider struct {
	log            *zerolog.Logger
	pods           map[podKey]*corev1.Pod
	pendingStatus  map[podKey]*corev1.PodStatus
	nodeName       string
	workloadClient api.WorkloadApiClient
	tracker        tracker.Tracker
	notifier       func(*corev1.Pod)
	mutex          sync.Mutex
}

func NewPodProvider(log *zerolog.Logger, nodeName string, workloadClient api.WorkloadApiClient, tracker tracker.Tracker) PodProvider {
	return &podProvider{
		log:            log,
		pods:           make(map[podKey]*corev1.Pod),
		pendingStatus:  make(map[podKey]*corev1.PodStatus),
		nodeName:       nodeName,
		workloadClient: workloadClient,
		tracker:        tracker,
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

	// Detect a tracked job through the absence of the untracked workload kind label
	if l, ok := pod.Labels[supernetes.LabelWorkloadKind]; !ok || l != string(supernetes.KindUntracked) {
		log.Trace().Msg("tracked workload detected")

		// Detect if the pod hasn't been scheduled yet through the absence of the workload identifier label
		if _, ok := pod.Labels[supernetes.LabelWorkloadIdentifier]; !ok {
			log.Trace().Msg("deploying workload")
			workload, err := convertToWorkload(pod, p.nodeName)
			if err != nil {
				return err
			}

			meta, err := p.workloadClient.Create(ctx, workload)
			if err != nil {
				// Mark the pod as failed and continue, otherwise VK will attempt to run this over and over again
				log.Err(err).Msg("deploying workload failed")
				pod.Status.Phase = corev1.PodFailed
			} else {
				// Scheduling succeeded, update the Pod and assign it the workload identifier
				log.Trace().Msg("applying returned metadata")
				applyWorkloadMeta(meta, pod)
			}
		}

		if _, ok := pod.Labels[supernetes.LabelWorkloadIdentifier]; ok {
			// If we have an identifier, register the Pod into the tracker so that its status can
			// get continually updated alongside the workload Pods that the created job spawned.
			p.tracker.Track(pod, p)
		}
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

	// Detect a tracked job through the absence of the untracked workload kind label
	if l, ok := pod.Labels[supernetes.LabelWorkloadKind]; !ok || l != string(supernetes.KindUntracked) {
		// Issue an opportunistic deletion request to the agent
		if _, err := p.workloadClient.Delete(ctx, workloadMeta(pod)); err != nil {
			log.Err(err).Msg("deleting workload failed")
		}

		// Remove the deleted pod from the tracker
		p.tracker.Untrack(pod)
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
func (p *podProvider) UpdateStatus(ctx context.Context, updatedPod *corev1.Pod, cache bool) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	key := keyFor(updatedPod)
	log := p.log.With().Fields(key.fields()).Logger()

	pod, ok := p.pods[key]
	if !ok {
		if !cache {
			log.Trace().Msg("pod not found")
			return nil
		}

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
