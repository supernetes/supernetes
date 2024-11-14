// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

// Adapted from https://pkg.go.dev/github.com/fluxcd/kustomize-controller/internal/inventory.
// Original license follows.

/*
Copyright 2021 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package inventory

import (
	"context"
	"fmt"

	"github.com/fluxcd/cli-utils/pkg/object"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	"github.com/fluxcd/pkg/ssa"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
	"github.com/supernetes/supernetes/common/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

type Inventory interface {
	Populate(ctx context.Context, kind supernetes.Kind) error
	AddChangeSet(set *ssa.ChangeSet)
	ListMetadata() (object.ObjMetadataSet, error)
	Diff(target Inventory) ([]*unstructured.Unstructured, error)
}

type inventory struct {
	kustomizev1.ResourceInventory
	k8sClient kubernetes.Interface
}

func New(k8sClient kubernetes.Interface) Inventory {
	return &inventory{
		ResourceInventory: kustomizev1.ResourceInventory{
			Entries: make([]kustomizev1.ResourceRef, 0),
		},
		k8sClient: k8sClient,
	}
}

// Populate adds all Pods with the given Supernetes workload kind label to the inventory
func (i *inventory) Populate(ctx context.Context, kind supernetes.Kind) error {
	podList, err := i.k8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", supernetes.LabelWorkloadKind, kind),
	})
	if err != nil {
		return err
	}

	for j := range podList.Items {
		pod := &podList.Items[j]
		if err := util.AddGVK(pod); err != nil {
			return err
		}

		i.Entries = append(i.Entries, kustomizev1.ResourceRef{
			ID: object.ObjMetadata{
				Namespace: pod.GetNamespace(),
				Name:      pod.GetName(),
				GroupKind: pod.GroupVersionKind().GroupKind(),
			}.String(),
			Version: pod.GroupVersionKind().GroupVersion().String(),
		})
	}

	return nil
}

// AddChangeSet extracts the metadata from the given objects and adds it to the inventory
func (i *inventory) AddChangeSet(set *ssa.ChangeSet) {
	if set == nil {
		return
	}

	for _, entry := range set.Entries {
		i.Entries = append(i.Entries, kustomizev1.ResourceRef{
			ID:      entry.ObjMetadata.String(),
			Version: entry.GroupVersion,
		})
	}
}

// ListMetadata returns the inventory entries as object.ObjMetadata objects
func (i *inventory) ListMetadata() (object.ObjMetadataSet, error) {
	var metas []object.ObjMetadata
	for _, e := range i.Entries {
		m, err := object.ParseObjMetadata(e.ID)
		if err != nil {
			return metas, err
		}
		metas = append(metas, m)
	}

	return metas, nil
}

// Diff returns the slice of objects that do not exist in the target inventory.
func (i *inventory) Diff(target Inventory) ([]*unstructured.Unstructured, error) {
	versionOf := func(i *inventory, objMetadata object.ObjMetadata) string {
		for _, entry := range i.Entries {
			if entry.ID == objMetadata.String() {
				return entry.Version
			}
		}
		return ""
	}

	objects := make([]*unstructured.Unstructured, 0)
	aList, err := i.ListMetadata()
	if err != nil {
		return nil, err
	}

	bList, err := target.ListMetadata()
	if err != nil {
		return nil, err
	}

	list := aList.Diff(bList)
	if len(list) == 0 {
		return objects, nil
	}

	for _, metadata := range list {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   metadata.GroupKind.Group,
			Kind:    metadata.GroupKind.Kind,
			Version: versionOf(i, metadata),
		})
		u.SetName(metadata.Name)
		u.SetNamespace(metadata.Namespace)
		objects = append(objects, u)
	}

	return objects, nil
}
