// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
		// TODO: Deal with corev1.DisruptionTarget
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
