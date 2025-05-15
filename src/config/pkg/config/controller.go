// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ControllerConfig encapsulates all relevant configuration for deploying a controller
// TODO: Versioning
type ControllerConfig struct {
	Port       uint16          `json:"port"`       // Port that the controller binds to
	MTlsConfig MTlsConfig      `json:"mTLSConfig"` // mTLS configuration for the controller
	Reconcile  ReconcileConfig `json:"reconcile"`  // Reconciliation configuration
}

type ReconcileConfig struct {
	NodeInterval     time.Duration `json:"nodeInterval"`     // Node reconciliation interval
	WorkloadInterval time.Duration `json:"workloadInterval"` // Workload reconciliation interval
}

// ToSecret converts the ControllerConfig into a corev1.Secret
func (c *ControllerConfig) ToSecret(meta metav1.ObjectMeta) (*corev1.Secret, error) {
	bytes, err := Encode(c)
	if err != nil {
		return nil, err
	}

	return &corev1.Secret{
		ObjectMeta: meta,
		Data:       map[string][]byte{"config.yaml": bytes},
		Type:       corev1.SecretTypeOpaque,
	}, nil
}
