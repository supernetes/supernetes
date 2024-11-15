// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package tracker

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// StatusUpdater provides an additional handler for asynchronously updating the status of a
// Pod. Implementors of UpdateStatus should not modify pod nor rely on pod.Spec being present.
type StatusUpdater interface {
	// The UpdateStatus cache field indicates whether the implementation is allowed to cache the update
	UpdateStatus(ctx context.Context, pod *corev1.Pod, cache bool) error
}
