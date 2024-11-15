// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package tracker

import (
	"context"
	"sync"

	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type tracked struct {
	meta          v1.ObjectMeta
	statusUpdater StatusUpdater
}

type Tracker interface {
	Track(pod *corev1.Pod, statusUpdater StatusUpdater)
	Untrack(pod *corev1.Pod)
	StatusUpdater
}

type tracker struct {
	tracked map[string]*tracked
	mutex   sync.RWMutex
}

func New() Tracker {
	return &tracker{
		tracked: make(map[string]*tracked),
	}
}

func (t *tracker) Track(pod *corev1.Pod, statusUpdater StatusUpdater) {
	info := &tracked{
		meta:          pod.ObjectMeta, // Shallow copy
		statusUpdater: statusUpdater,
	}

	log.Trace().Str("name", pod.Name).Str("namespace", pod.Namespace).Msg("tracking pod")

	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.tracked[pod.Labels[supernetes.LabelWorkloadIdentifier]] = info
}

func (t *tracker) Untrack(pod *corev1.Pod) {
	log.Trace().Str("name", pod.Name).Str("namespace", pod.Namespace).Msg("untracking pod")

	t.mutex.Lock()
	defer t.mutex.Unlock()
	delete(t.tracked, pod.Labels[supernetes.LabelWorkloadIdentifier])
}

func (t *tracker) UpdateStatus(ctx context.Context, pod *corev1.Pod, cache bool) error {
	t.mutex.RLock()
	info, ok := t.tracked[pod.Labels[supernetes.LabelWorkloadIdentifier]]
	t.mutex.RUnlock()

	if ok {
		// statusPod is a carrier that binds together the ObjectMeta of the
		// original tracked Pod and the status of the passed-in untracked pod.
		statusPod := &corev1.Pod{
			ObjectMeta: info.meta,
			Status:     pod.Status,
		}

		return info.statusUpdater.UpdateStatus(ctx, statusPod, cache)
	}

	return nil
}
