// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package workload

import (
	"context"
	"io"
	"time"

	"github.com/fluxcd/pkg/ssa"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/controller/pkg/reconciler"
	"github.com/supernetes/supernetes/util/pkg/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ReconcilerConfig struct {
	Interval  time.Duration         // Reconciliation interval
	Client    api.WorkloadApiClient // Client for accessing the workload API
	K8sConfig *rest.Config          // Configuration for accessing Kubernetes
}

type wlReconciler struct {
	client api.WorkloadApiClient
	resMgr *ssa.ResourceManager
}

func NewReconciler(ctx context.Context, config ReconcilerConfig) (reconciler.Reconciler, error) {
	mgr, err := ctrl.NewManager(config.K8sConfig, ctrl.Options{})
	if err != nil {
		return nil, err
	}

	resMgr := ssa.NewResourceManager(mgr.GetClient(), nil, ssa.Owner{
		Field: "supernetes-controller",
		Group: "supernetes", // TODO: Proper group for Supernetes
	})

	logger := log.Scoped().Str("type", "workload").Logger()
	return reconciler.New(ctx, &logger, config.Interval, &wlReconciler{
		client: config.Client,
		resMgr: resMgr,
	})
}

func (r *wlReconciler) Reconcile(ctx context.Context) error {
	stream, err := r.client.List(ctx, nil)
	if err != nil {
		return err
	}

	for {
		// Query the agent for workloads
		workload, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Convert the returned workloads into Pods
		pods := convertToPods(workload)
		unstructuredPods, err := toUnstructured(pods)
		if err != nil {
			return err
		}

		// Create/update/delete the Pods
		changeSet, err := r.resMgr.ApplyAll(ctx, unstructuredPods, ssa.ApplyOptions{Force: true})
		if err != nil {
			return err
		}

		// Log any changes
		for _, change := range changeSet.Entries {
			if change.Action != ssa.UnchangedAction {
				log.Debug().Str("subject", change.Subject).Stringer("action", change.Action).Msg("applied pod")
			}
		}
	}

	return nil
}

func toUnstructured[T any](objects []T) ([]*unstructured.Unstructured, error) {
	uObjects := make([]*unstructured.Unstructured, 0, len(objects))
	for _, o := range objects {
		uObject, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
		if err != nil {
			return nil, err
		}
		uObjects = append(uObjects, &unstructured.Unstructured{Object: uObject})
	}
	return uObjects, nil
}
