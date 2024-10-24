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

	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/util/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/kubernetes"
)

type Reconciler interface {
	Start()
	Stop()
}

type reconciler struct {
	ctx       context.Context
	cancel    func()
	client    api.WorkloadApiClient
	k8sClient kubernetes.Interface
}

func NewReconciler(ctx context.Context, client api.WorkloadApiClient, k8sClient kubernetes.Interface) Reconciler {
	ctx, cancel := context.WithCancel(ctx)
	return &reconciler{
		ctx:       ctx,
		cancel:    cancel,
		client:    client,
		k8sClient: k8sClient,
	}
}

func (j *reconciler) Start() {
	// TODO: Prevent this from starting multiple times
	go func() {
		ticker := time.NewTicker(10 * time.Second) // TODO: Configurable time

		for {
			err := j.reconcile()
			if err != nil {
				log.Err(err).Msg("reconciling workloads failed")
			}

			select {
			case <-ticker.C:

			case <-j.ctx.Done():
				return
			}
		}
	}()
}

func (j *reconciler) Stop() {
	j.cancel()
}

func (j *reconciler) reconcile() error {
	// TODO: Query the agent for all workloads (jobs)
	// TODO: Convert the returned workloads into Pods
	// TODO: Create all the Pods that are not present
	// TODO: Delete all the Pods that are no longer tracked

	stream, err := j.client.List(j.ctx, nil)
	if err != nil {
		return err
	}

	podInterface := j.k8sClient.CoreV1().Pods("supernetes-workload")
	pods, err := podInterface.List(j.ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	// Lookup table for deciding whether to create or update Pods
	names := make(map[string]any)
	for _, p := range pods.Items {
		names[p.Name] = struct{}{}
	}

	for {
		workload, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		pods := convertToPods(workload)

		for _, pod := range pods {
			if _, ok := names[pod.Name]; ok {
				continue // Pod already created TODO: update/diff support
			}

			log.Debug().
				Str("workload", workload.Meta.Name).
				Str("name", pod.Name).
				Msg("applying pod for workload")

			podApply, err := applyv1.ExtractPod(pod, "supernetes") // TODO: fieldManager?
			if err != nil {
				return err
			}

			//_, err := j.k8sClient.CoreV1().Pods("supernetes-workload").Create(j.ctx, pod, metav1.CreateOptions{})
			_, err = j.k8sClient.CoreV1().Pods("supernetes-workload").Apply(j.ctx, podApply, metav1.ApplyOptions{
				FieldManager: "supernetes",
			})
			if err != nil {
				return err
			}
		}
	}
}
