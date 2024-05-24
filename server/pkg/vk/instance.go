// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vk

import (
	"context"
	"strconv"

	"github.com/supernetes/supernetes/api"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

// instance represents a singular Virtual Kubelet node
type instance struct {
	tracked bool
	cancel  func()
}

func newInstance(nodeInterface v1.NodeInterface, n *api.Node) *instance {
	ctx, cancel := context.WithCancel(context.Background())

	// TODO: This needs to be properly populated based on `n`
	nodeCfg := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: n.Name,
		},
		Spec: corev1.NodeSpec{
			Taints: []corev1.Taint{{
				Key:    "supernetes-node/no-schedule",
				Value:  strconv.FormatBool(true),
				Effect: corev1.TaintEffectNoSchedule,
			}},
		},
		Status: corev1.NodeStatus{
			// NodeInfo: v1.NodeSystemInfo{
			// 	KubeletVersion:  Version,
			// 	Architecture:    architecture,
			// 	OperatingSystem: linuxos,
			// },
			//Addresses:       []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: internalIP}},
			//DaemonEndpoints: corev1.NodeDaemonEndpoints{KubeletEndpoint: corev1.DaemonEndpoint{Port: int32(daemonEndpointPort)}},
			Capacity: corev1.ResourceList{
				"cpu":    resource.MustParse("1"),
				"memory": resource.MustParse("1Gi"),
				"pods":   resource.MustParse("0"),
			},
			Allocatable: corev1.ResourceList{
				"cpu":    resource.MustParse("0"),
				"memory": resource.MustParse("0"),
				"pods":   resource.MustParse("0"),
			},
			//Conditions: nodeConditions(), // TODO: This needs to be dynamically synchronized
		},
	}

	// TODO: Currently the node status is externally managed, but we could consider implementing `NodeProvider` here
	provider := &node.NaiveNodeProvider{}
	nodeRunner, _ := node.NewNodeController(provider, nodeCfg, nodeInterface)
	go func() {
		log.Debug().Msgf("starting controller for node %q", n.Name)
		if err := nodeRunner.Run(ctx); err != nil {
			log.Err(err).Msgf("controller for node %q failed", n.Name)
			return
		}
		log.Debug().Msgf("stopping controller for node %q", n.Name)
	}()

	// setup other things
	//podRunner, _ := node.NewPodController(...)

	//go podRunner.Run(ctx)
	//
	//select {
	//case <-podRunner.Ready():
	//case <-podRunner.Done():
	//}
	//if podRunner.Err() != nil {
	//	// handle error
	//}

	return &instance{
		tracked: true, // Newly created instances are always tracked
		cancel:  cancel,
	}
}
