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

	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/util/pkg/log"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

func convertToPods(workload *api.Workload) []*corev1.Pod {
	pods := make([]*corev1.Pod, 0, max(len(workload.Status.Nodes), 1))

	if len(workload.Status.Nodes) == 0 {
		// Map the workload into a single pod if it's not allocated to any nodes
		pods = append(pods, convertToPod(workload, nil, 0))
		return pods
	}

	for i, node := range workload.Status.Nodes {
		pods = append(pods, convertToPod(workload, &node.Name, i))
	}

	return pods
}

func convertToPod(workload *api.Workload, node *string, index int) *corev1.Pod {
	nodeName := ""
	schedulingGates := make([]corev1.PodSchedulingGate, 0)
	if node != nil {
		nodeName = *node
	} else {
		schedulingGates = append(schedulingGates, corev1.PodSchedulingGate{
			Name: "supernetes-workload/unallocated", // Unallocated workloads do not get scheduled
		})
	}

	labels := map[string]string{
		"supernetes-workload/idenfitier": workload.Meta.Identifier,
	}

	// Add diagnostics metadata under supernetes-extra
	for k, v := range workload.Meta.Extra {
		labels[fmt.Sprintf("supernetes-extra/%s", k)] = v
	}

	return addGVK(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName(workload, index),
			Namespace: "supernetes-workload", // Namespace for untracked workloads TODO: make this configurable
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:                     "workload", // TODO: Move to common labels
				Image:                    "none",     // TODO: Reasonable placeholder
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
				Key:      "supernetes-node/no-schedule", // TODO: Move to common code
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			}},
			SchedulingGates: schedulingGates,
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
			StartTime:         &metav1.Time{Time: time.Unix(workload.Status.StartTime, 0)},
			ContainerStatuses: nil,
		},
	})
}

func podName(workload *api.Workload, index int) string {
	return fmt.Sprintf("%s-%s-%d", toLowerRFC1123(workload.Meta.Identifier), toLowerRFC1123(workload.Meta.Name), index)
}

// toLowerRFC1123 converts the input string into a lowercase RFC 1123 compliant
// string (without periods). Useful for constructing valid Pod names.
func toLowerRFC1123(input string) string {
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

// addGVK is a helper for essentially adding TypeMeta information to runtime.Objects.
// This resolves a persistent issue with different components (such as fluxcd/pkg/ssa)
// requiring it to be set: https://github.com/kubernetes-sigs/controller-runtime/issues/1735
func addGVK[T interface {
	runtime.Object
	SetGroupVersionKind(gvk schema.GroupVersionKind) // This is compatible with metav1.TypeMeta
}](object T) T {
	gvks, unversioned, err := scheme.Scheme.ObjectKinds(object)
	if err != nil {
		log.Err(err).Msg("unable to set GVK for object")
		return object
	}

	if !unversioned && len(gvks) == 1 {
		object.SetGroupVersionKind(gvks[0])
		return object // Success
	}

	log.Error().
		Type("type", object).
		Bool("unversioned", unversioned).
		Int("GVK count", len(gvks)).
		Msg("unable to set GVK for object")

	return object
}
