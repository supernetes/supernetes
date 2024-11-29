// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
	corev1 "k8s.io/api/core/v1"
)

// getNodes builds a list of the node names for the job
func getNodes(pod *corev1.Pod, nodeName string) []string {
	nodes := []string{nodeName} // The primary node name is always included

	// Source additional nodes from the additional nodes label
	if additional, ok := pod.Labels[supernetes.LabelAdditionalNodes]; ok {
		nodes = append(nodes, strings.Split(additional, ",")...)
	}

	return nodes
}

// getLabels extracts labels in a Supernetes scope
func getLabels(pod *corev1.Pod, scope string) map[string]string {
	labels := make(map[string]string)

	for k, v := range pod.Labels {
		kt := strings.TrimPrefix(k, scope+"/")
		if kt != k {
			labels[kt] = v
		}
	}

	return labels
}

// workloadMeta builds an api.WorkloadMeta for the given corev1.Pod
func workloadMeta(pod *corev1.Pod) *api.WorkloadMeta {
	var identifier string
	if id, ok := pod.Labels[supernetes.LabelWorkloadIdentifier]; ok {
		identifier = id
	}

	return &api.WorkloadMeta{
		Name:       pod.Name,
		Identifier: identifier,
		Extra:      getLabels(pod, supernetes.ScopeExtra),
	}
}

// convertToWorkload converts the given corev1.Pod spec to an api.Workload for deployment
func convertToWorkload(pod *corev1.Pod, nodeName string) (*api.Workload, error) {
	if len(pod.Spec.Containers) != 1 {
		return nil, errors.New("pod must have exactly one container")
	}
	container := &pod.Spec.Containers[0]

	return &api.Workload{
		Meta: workloadMeta(pod),
		Spec: &api.WorkloadSpec{
			Image:      container.Image,
			Command:    container.Command,
			Args:       container.Args,
			NodeNames:  getNodes(pod, nodeName),
			JobOptions: getLabels(pod, supernetes.ScopeOption),
		},
	}, nil
}

// applyWorkloadMeta applies the configuration from the given api.WorkloadMeta to the given corev1.Pod
func applyWorkloadMeta(meta *api.WorkloadMeta, pod *corev1.Pod) {
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}

	for k, v := range meta.Extra {
		pod.Labels[fmt.Sprintf("%s/%s", supernetes.ScopeExtra, k)] = v
	}

	pod.Labels[supernetes.LabelWorkloadIdentifier] = meta.Identifier
}
