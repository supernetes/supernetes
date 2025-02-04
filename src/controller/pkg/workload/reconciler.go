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
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
	"github.com/supernetes/supernetes/controller/pkg/client"
	"github.com/supernetes/supernetes/controller/pkg/inventory"
	"github.com/supernetes/supernetes/controller/pkg/reconciler"
	"github.com/supernetes/supernetes/controller/pkg/tracker"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
)

type ReconcilerConfig struct {
	Interval      time.Duration         // Reconciliation interval
	Client        api.WorkloadApiClient // Client for accessing the workload API
	K8sConfig     *rest.Config          // Configuration for accessing Kubernetes
	StatusUpdater tracker.StatusUpdater // Callback to trigger manual Pod status updates
	Tracker       tracker.Tracker       // Manager for tracked Pods
}

type wlReconciler struct {
	client        api.WorkloadApiClient
	statusUpdater tracker.StatusUpdater
	tracker       tracker.Tracker
	resMgr        *ssa.ResourceManager
	k8sClient     kubernetes.Interface
	inventory     inventory.Inventory
}

func NewReconciler(ctx context.Context, config ReconcilerConfig) (reconciler.Reconciler, error) {
	mgr, err := ctrl.NewManager(config.K8sConfig, ctrl.Options{})
	if err != nil {
		return nil, err
	}

	k8sClient, err := client.NewKubeClient(config.K8sConfig)
	if err != nil {
		return nil, err
	}

	resMgr := ssa.NewResourceManager(mgr.GetClient(), nil, ssa.Owner{
		Field: supernetes.ScopeController,
		Group: supernetes.Group,
	})

	logger := log.Scoped().Str("type", "workload").Logger()
	return reconciler.New(ctx, &logger, config.Interval, &wlReconciler{
		client:        config.Client,
		statusUpdater: config.StatusUpdater,
		tracker:       config.Tracker,
		resMgr:        resMgr,
		k8sClient:     k8sClient,
	})
}

func (r *wlReconciler) Reconcile(ctx context.Context) error {
	stream, err := r.client.List(ctx, nil)
	if err != nil {
		return err
	}

	if r.inventory == nil {
		// Initialize the Pod tracking inventory
		r.inventory = inventory.New(r.k8sClient)

		// Populate it with the untracked pods currently present in the cluster
		// TODO: It might be useful to call this periodically as well for consistency
		if err := r.inventory.Populate(ctx, supernetes.KindUntracked); err != nil {
			return err
		}
	}

	// Create an inventory for the reconciled resources
	newInventory := inventory.New(r.k8sClient)

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
		pods, err := convertToPods(workload)
		if err != nil {
			return err
		}
		unstructuredPods, err := toUnstructured(pods)
		if err != nil {
			return err
		}

		// Create/update the Pods
		changeSet, err := r.resMgr.ApplyAll(ctx, unstructuredPods, ssa.ApplyOptions{Force: true})
		if err != nil {
			return err
		}

		// Track the changes
		newInventory.AddChangeSet(changeSet)

		for i, change := range changeSet.Entries {
			if change.Action == ssa.CreatedAction || change.Action == ssa.UnchangedAction {
				// For any created Pods or Pods with an unchanged PodSpec (but possibly changed
				// PodStatus), we need to perform a manual status update in the provider
				if err := r.statusUpdater.UpdateStatus(ctx, pods[i], true); err != nil {
					return err
				}

				// If this is the primary Pod (index 0), also update the corresponding tracked Pod
				// (if present). Tracked Pod status updates should not be cached in the provider.
				if i == 0 {
					if err := r.tracker.UpdateStatus(ctx, pods[0], false); err != nil {
						return err
					}
				}
			}

			// Log any changes
			if change.Action != ssa.UnchangedAction {
				log.Debug().Str("subject", change.Subject).Stringer("action", change.Action).Msg("applied pod")
			}
		}
	}

	// Detect stale resources which are subject to garbage collection
	staleObjects, err := r.inventory.Diff(newInventory)
	if err != nil {
		return err
	}

	// Run garbage collection for stale resources
	changeSet, err := r.resMgr.DeleteAll(ctx, staleObjects, ssa.DeleteOptions{
		PropagationPolicy: metav1.DeletePropagationBackground,
	})
	if err != nil {
		return err
	}

	// Log any changes
	for _, change := range changeSet.Entries {
		if change.Action != ssa.UnchangedAction {
			log.Debug().Str("subject", change.Subject).Stringer("action", change.Action).Msg("applied pod")
		}
	}

	// New inventory is now current
	r.inventory = newInventory

	return nil
}

// toUnstructured explicitly does not sort the objects to allow for indexing later
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
