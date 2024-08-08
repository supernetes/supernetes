// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vk

import (
	"context"

	"github.com/pkg/errors"
	"github.com/supernetes/supernetes/common/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

// DisableKubeProxy prevents kube-proxy pods from being deployed on Virtual Kubelet nodes
func DisableKubeProxy(k8sClient kubernetes.Interface) error {
	log.Debug().Msg("patching kube-proxy DaemonSet to exclude type=virtual-kubelet")
	_, err := k8sClient.AppsV1().DaemonSets("kube-system").Patch(
		context.Background(),
		"kube-proxy",
		types.StrategicMergePatchType,
		[]byte("{\"spec\":{\"template\":{\"spec\":{\"affinity\":{\"nodeAffinity\":{\"requiredDuringSchedulingIgnoredDuringExecution\":{\"nodeSelectorTerms\":[{\"matchExpressions\":[{\"key\":\"type\",\"operator\":\"NotIn\",\"values\":[\"virtual-kubelet\"]}]}]}}}}}}}"),
		metav1.PatchOptions{},
	)

	return errors.Wrap(err, "patching kube-proxy DaemonSet failed")
}
