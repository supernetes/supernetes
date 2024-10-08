// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package vk

import (
	"context"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/controller/pkg/provider"
	sulog "github.com/supernetes/supernetes/util/pkg/log"
	vklog "github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// Instance combines all Virtual Kubelet controllers for handling a single node and its workloads
type Instance interface {
	Run(context.Context) error
}

type instance struct {
	cfg *nodeutil.NodeConfig
}

// TODO: This doesn't re-create the node if it's deleted from the API server
//  However, it should now support invoking Instance.Run multiple times to solve that

// NewInstance creates a new Instance for the given node.
func NewInstance(client kubernetes.Interface, n *api.Node) Instance {
	// TODO: This needs to be properly populated based on `n`
	cfg := nodeutil.NodeConfig{
		Client:               client,
		NumWorkers:           1, // TODO: Scaling
		InformerResyncPeriod: time.Minute,
		NodeSpec: corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: n.Meta.Name,
				Labels: map[string]string{
					"type":                   "virtual-kubelet",
					"kubernetes.io/role":     "supernetes",
					"kubernetes.io/hostname": n.Meta.Name,
				},
			},
			Spec: corev1.NodeSpec{
				Taints: []corev1.Taint{{
					Key:    "supernetes-node/no-schedule",
					Effect: corev1.TaintEffectNoSchedule,
				}},
			},
			Status: corev1.NodeStatus{
				Phase: corev1.NodePending,
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady},
					{Type: corev1.NodeDiskPressure},
					{Type: corev1.NodeMemoryPressure},
					{Type: corev1.NodePIDPressure},
					{Type: corev1.NodeNetworkUnavailable},
				},
				// TODO: Apply these as a post-operation or source them directly from api.Node here?
				Capacity: corev1.ResourceList{
					"cpu":    resource.MustParse("1"),
					"memory": resource.MustParse("1Gi"),
					"pods":   resource.MustParse("1"),
				},
			},
		},
	}

	return &instance{
		cfg: &cfg,
	}
}

// Run starts the instance controllers with the given context
func (i *instance) Run(ctx context.Context) error {
	cfg := *i.cfg                                        // Instance configuration, shallow copy for assignment overrides
	nodeName := cfg.NodeSpec.ObjectMeta.Name             // Shorthand for node name
	log := sulog.Scoped().Str("node", nodeName).Logger() // Node-specific scoped logger

	// Configure the error handler for status updates
	cfg.NodeStatusUpdateErrorHandler = func(_ context.Context, err error) error {
		log.Err(err).Msg("status update failed")
		return err
	}

	// Configure the event recorder for the pod controller
	podEvents := record.NewBroadcaster()
	cfg.EventRecorder = podEvents.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: path.Join(nodeName, "pod-controller")},
	)

	// Set up node controller
	// TODO: Currently the node status is externally managed, but we could consider implementing `NodeProvider` here
	nodeProvider := node.NewNaiveNodeProvider()
	nodeController, err := node.NewNodeController(nodeProvider, &cfg.NodeSpec, cfg.Client.CoreV1().Nodes(),
		node.WithNodeEnableLeaseV1(nodeutil.NodeLeaseV1Client(cfg.Client), 0),
		node.WithNodeStatusUpdateErrorHandler(cfg.NodeStatusUpdateErrorHandler),
	)
	if err != nil {
		return errors.Wrap(err, "creating node controller failed")
	}

	// Set up informers
	scmInformerFactory := informers.NewSharedInformerFactory(
		cfg.Client,
		cfg.InformerResyncPeriod,
	)
	podInformerFactory := informers.NewSharedInformerFactoryWithOptions(
		cfg.Client,
		cfg.InformerResyncPeriod,
		nodeutil.PodInformerFilter(nodeName), // Node-specific informer for pod events
	)

	// Set up pod controller
	podProviderLogger := log.With().Str("scope", "provider").Logger()
	podProvider := provider.NewPodProvider(&podProviderLogger)
	podControllerCfg := node.PodControllerConfig{
		PodClient:         cfg.Client.CoreV1(),
		EventRecorder:     cfg.EventRecorder,
		Provider:          podProvider,
		PodInformer:       podInformerFactory.Core().V1().Pods(),
		ConfigMapInformer: scmInformerFactory.Core().V1().ConfigMaps(),
		SecretInformer:    scmInformerFactory.Core().V1().Secrets(),
		ServiceInformer:   scmInformerFactory.Core().V1().Services(),
	}
	podController, err := node.NewPodController(podControllerCfg)
	if err != nil {
		return errors.Wrap(err, "creating pod controller failed")
	}

	vkLogger := log.Level(zerolog.InfoLevel)               // TODO: Configurability, VK is noisy
	ctx = vklog.WithLogger(ctx, sulog.VKLogger(&vkLogger)) // Virtual Kubelet logging
	ctx, cancel := context.WithCancel(ctx)                 // Controllers must be able to stop each other

	// Start all informers
	log.Debug().Msg("starting informers")
	go podInformerFactory.Start(ctx.Done())
	go scmInformerFactory.Start(ctx.Done())

	// Start pod controller
	go func() {
		defer cancel()

		// Start recoding pod controller events
		log.Debug().Msg("starting event broadcaster for pod controller")
		podEvents.StartLogging(log.Info().Str("scope", "pod-events").Msgf) // TODO: Log pod controller events?
		podEvents.StartRecordingToSink(&corev1client.EventSinkImpl{
			Interface: cfg.Client.CoreV1().Events(corev1.NamespaceAll),
		})
		defer podEvents.Shutdown()

		log.Debug().Msg("starting pod controller")
		if err := podController.Run(ctx, cfg.NumWorkers); err != nil {
			log.Err(err).Msg("running pod controller failed")
			return
		}

		log.Debug().Msg("stopping pod controller")
	}()

	// Wait for pod controller to become ready
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-podController.Ready():
	case <-podController.Done():
		return podController.Err() // Pod controller failed to become ready, return its error
	}

	// Start node controller
	go func() {
		defer cancel()
		log.Debug().Msg("starting node controller")
		if err := nodeController.Run(ctx); err != nil {
			log.Err(err).Msg("running node controller failed")
			return
		}

		// TODO: We need to defer/handle node deletion here, Virtual Kubelet doesn't seem to do it automatically.
		//  Normally, deletion of stale nodes after a timeout would normally be handled by the Cluster Autoscaler
		//  (https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler), but this seems to heavily rely
		//  on cloud-provider-specific APIs and won't work with Talos.
		// TODO: Maybe Supernetes should run another controller/reconciliation loop for handling virtual node pruning?
		// TODO: Another option is the Kyverno cleanup controller (https://kyverno.io/docs/writing-policies/cleanup/)

		log.Debug().Msg("stopping node controller")
	}()

	// Wait for node controller to become ready
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-nodeController.Ready():
	case <-nodeController.Done():
		return nodeController.Err() // Node controller failed to become ready, return its error
	}

	// Mark node as ready, operate on a copy to support invoking i.Run multiple times
	log.Debug().Msg("marking node as ready")
	nodeReady := setReady(cfg.NodeSpec.DeepCopy())
	return errors.Wrap(nodeProvider.UpdateStatus(ctx, nodeReady), "error marking node as ready")
}

func setReady(n *corev1.Node) *corev1.Node {
	n.Status.Phase = corev1.NodeRunning
	for i, c := range n.Status.Conditions {
		if c.Type != "Ready" {
			continue
		}

		c.Message = "Kubelet is ready"
		c.Reason = "KubeletReady"
		c.Status = corev1.ConditionTrue
		c.LastHeartbeatTime = metav1.Now()
		c.LastTransitionTime = metav1.Now()
		n.Status.Conditions[i] = c
		break
	}

	return n
}
