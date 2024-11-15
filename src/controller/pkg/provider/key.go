// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package provider

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// podKey is used to index Pod resources tracked by this provider
type podKey struct {
	name      string
	namespace string
}

func keyFor(pod metav1.Object) podKey {
	return podKey{
		name:      pod.GetName(),
		namespace: pod.GetNamespace(),
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
