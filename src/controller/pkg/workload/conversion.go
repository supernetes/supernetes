// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package workload

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/pkg/errors"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
	"github.com/supernetes/supernetes/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func convertToPods(workload *api.Workload) ([]*corev1.Pod, error) {
	pods := make([]*corev1.Pod, 0, max(len(workload.Status.Nodes), 1))

	if len(workload.Status.Nodes) == 0 {
		// Map the workload into a single pod if it's not allocated to any nodes
		pod, err := convertToPod(workload, nil, 0)
		if err != nil {
			return nil, err
		}

		return append(pods, pod), nil
	}

	for i, node := range workload.Status.Nodes {
		pod, err := convertToPod(workload, &node.Name, i)
		if err != nil {
			return nil, err
		}

		pods = append(pods, pod)
	}

	return pods, nil
}

func convertToPod(workload *api.Workload, node *string, index int) (*corev1.Pod, error) {
	nodeName := ""
	schedulingGates := make([]corev1.PodSchedulingGate, 0)
	if node != nil {
		nodeName = *node
	} else {
		// Unallocated workloads do not get scheduled
		schedulingGates = append(schedulingGates, corev1.PodSchedulingGate{
			Name: supernetes.SGWorkloadUnallocated,
		})
	}

	labels := map[string]string{
		supernetes.LabelWorkloadIdentifier: workload.Meta.Identifier,
		supernetes.LabelWorkloadKind:       string(supernetes.KindUntracked),
	}

	// Add diagnostics metadata under supernetes.ScopeExtra
	for k, v := range workload.Meta.Extra {
		labels[fmt.Sprintf("%s/%s", supernetes.ScopeExtra, k)] = v
	}

	// Virtual Kubelet always waits until the grace period, reducing
	// it from the default (30 seconds) greatly speeds up pod deletion
	var terminationGracePeriod int64 = 1

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName(workload, index),
			Namespace: supernetes.NamespaceWorkload, // Namespace for untracked workloads TODO: make this configurable
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:                     supernetes.ContainerPlaceholder,
				Image:                    supernetes.ImagePlaceholder,
				Command:                  nil,
				Args:                     nil,
				WorkingDir:               "",
				Ports:                    nil,
				EnvFrom:                  nil,
				Env:                      nil,
				Resources:                corev1.ResourceRequirements{},
				ResizePolicy:             nil,
				RestartPolicy:            nil,
				VolumeMounts:             nil,
				VolumeDevices:            nil,
				LivenessProbe:            nil,
				ReadinessProbe:           nil,
				StartupProbe:             nil,
				Lifecycle:                nil,
				TerminationMessagePath:   "",
				TerminationMessagePolicy: "",
				ImagePullPolicy:          "",
				SecurityContext:          nil,
				Stdin:                    false,
				StdinOnce:                false,
				TTY:                      false,
			}}, // TODO: Can this be empty for untracked workloads?
			NodeName: nodeName,
			Tolerations: []corev1.Toleration{{
				Key:      supernetes.TaintNoSchedule,
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			}},
			SchedulingGates:               schedulingGates,
			TerminationGracePeriodSeconds: &terminationGracePeriod,
		},
		Status: corev1.PodStatus{
			Phase: workloadPhaseToPodPhase(workload.Status.Phase),
			//Conditions:        nil,
			//Message:           "",
			//Reason:            "",
			//HostIP:  "",
			//HostIPs: nil,
			//PodIP:   "",
			//PodIPs:  nil,
			StartTime: &metav1.Time{Time: time.Unix(workload.Status.StartTime, 0)},
		},
	}

	if err := util.AddGVK(pod); err != nil {
		return nil, errors.Wrap(err, "unable to set GVK for pod")
	}

	return pod, nil
}

func podName(workload *api.Workload, index int) string {
	prefix := fmt.Sprintf("%s-", toLowerRFC1123(workload.Meta.Identifier, -1))
	suffix := fmt.Sprintf("-%d", index)

	// Pod names can be at most 63 characters long
	return prefix + toLowerRFC1123(workload.Meta.Name, 63-len(prefix)-len(suffix)) + suffix
}

// toLowerRFC1123 converts the input string into a lowercase RFC 1123 compliant string (without periods). Useful for
// constructing valid Pod names. If maxLen is positive, the output string will be at most maxLen runes long.
func toLowerRFC1123(input string, maxLen int) string {
	var result []rune

	for _, c := range strings.ToLower(input) {
		if c <= unicode.MaxLatin1 && (unicode.IsLetter(c) || unicode.IsDigit(c)) {
			result = append(result, c)
		} else {
			if len(result) > 0 && result[len(result)-1] == '-' {
				continue // Avoid repeated dashes
			}

			result = append(result, '-')
		}

		if maxLen > 0 && len(result) == maxLen {
			break // Length limit reached
		}
	}

	// Must start with and end in an alphanumeric character
	return strings.Trim(string(result), "-")
}

func workloadPhaseToPodPhase(phase api.WorkloadPhase) corev1.PodPhase {
	switch phase {
	case api.WorkloadPhase_Pending:
		return corev1.PodPending
	case api.WorkloadPhase_Running:
		return corev1.PodRunning
	case api.WorkloadPhase_Succeeded:
		return corev1.PodSucceeded
	case api.WorkloadPhase_Failed:
		return corev1.PodFailed
	case api.WorkloadPhase_Unknown:
		return corev1.PodUnknown
	}

	panic(fmt.Sprintf("encountered invalid workload phase %q", phase))
}
