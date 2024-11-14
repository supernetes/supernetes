// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package util

import (
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
)

// AddGVK is a helper for essentially adding TypeMeta information to runtime.Objects.
// This resolves a persistent issue with different components (such as fluxcd/pkg/ssa)
// requiring it to be set: https://github.com/kubernetes-sigs/controller-runtime/issues/1735
func AddGVK[T interface {
	runtime.Object
	SetGroupVersionKind(gvk schema.GroupVersionKind) // This is compatible with metav1.TypeMeta
}](object T) error {
	gvks, unversioned, err := scheme.Scheme.ObjectKinds(object)
	if err != nil {
		return err
	}

	if unversioned {
		return errors.New("object is unversioned")
	}

	if len(gvks) == 0 {
		return errors.New("no GVKs found for object")
	}

	if len(gvks) > 1 {
		return fmt.Errorf("ambiguous object, found %d GVKs", len(gvks))
	}

	object.SetGroupVersionKind(gvks[0])
	return nil // Success
}
