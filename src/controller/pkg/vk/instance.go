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
	suerr "github.com/supernetes/supernetes/common/pkg/error"
	sulog "github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/common/pkg/supernetes"
	"github.com/supernetes/supernetes/controller/pkg/environment"
	"github.com/supernetes/supernetes/controller/pkg/provider"
	"github.com/supernetes/supernetes/controller/pkg/tracker"
	vkauth "github.com/supernetes/supernetes/controller/pkg/vk/auth"
	vklog "github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/node"
	vkapi "github.com/virtual-kubelet/virtual-kubelet/node/api"
	"github.com/virtual-kubelet/virtual-kubelet/node/nodeutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// Instance combines all Virtual Kubelet controllers for handling a single node and its pods
type Instance interface {
	// Run starts the instance's controllers. The controllers use cancel() to stop each other.
	Run(ctx context.Context, cancel func()) error
	// UpdateNodeStatus is used to asynchronously update the status of the node
	UpdateNodeStatus(status *api.NodeStatus)
	// StatusUpdater can be used to trigger Pod status updates in the associated Pod provider
	tracker.StatusUpdater
}

type instance struct {
	cfg                 *nodeutil.NodeConfig
	podProvider         provider.PodProvider
	workloadClient      api.WorkloadApiClient
	tracker             tracker.Tracker
	metricsProvider     provider.MetricsProvider
	vkAuth              vkauth.Auth
	enableKubeletServer bool
	disableAuth         bool
}

type InstanceConfig struct {
	KubeClient     kubernetes.Interface
	Node           *api.Node
	WorkloadClient api.WorkloadApiClient
	Tracker        tracker.Tracker
	Environment    environment.Environment
	VkAuth         vkauth.Auth
	IsOpenShift    bool
}

// TODO: This doesn't re-create the node if it's deleted from the API server
//  However, it should now support invoking Instance.Run multiple times to solve that

// NewInstance creates a new Instance for the given node
func NewInstance(instanceCfg InstanceConfig) Instance {
	// TODO: This needs to be properly populated based on `node`
	// TODO: That includes labeling/tainting the node with its partitions, so that the Kubernetes scheduler doesn't
	//  attempt to schedule workloads onto nodes that can't receive them. This also requires, that the controller is
	//  either aware of the partition used by the agent, or that the agent can tell it which partitions it can schedule
	//  to. Also keep in mind the future support of labeling/annotating the partition that should be used in the pod.
	cfg := nodeutil.NodeConfig{
		Client:               instanceCfg.KubeClient,
		NumWorkers:           1,           // TODO: Scaling
		InformerResyncPeriod: time.Minute, // TODO: Configurability
		NodeSpec: corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: instanceCfg.Node.Meta.Name,
				Labels: map[string]string{
					"type":                   supernetes.NodeTypeVirtualKubelet,
					"kubernetes.io/role":     supernetes.NodeRoleSupernetes,
					"kubernetes.io/hostname": instanceCfg.Node.Meta.Name,
				},
			},
			Spec: corev1.NodeSpec{
				Taints: []corev1.Taint{{
					Key:    supernetes.TaintNoSchedule,
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
					"cpu":    *resource.NewQuantity(int64(instanceCfg.Node.Spec.CpuCount), resource.DecimalSI),
					"memory": *resource.NewQuantity(int64(instanceCfg.Node.Spec.MemBytes), resource.BinarySI),
					"pods":   resource.MustParse("100"), // TODO: This must be configurable
				},
				Addresses: []corev1.NodeAddress{
					{
						Type:    corev1.NodeHostName,
						Address: instanceCfg.Node.Meta.Name,
					},
				},
			},
		},
	}

	enableKubeletServer := false
	if addr := instanceCfg.Environment.ControllerAddress(); addr != nil {
		enableKubeletServer = true
		status := &cfg.NodeSpec.Status
		status.Addresses = append(status.Addresses, corev1.NodeAddress{
			Type:    corev1.NodeInternalIP,
			Address: addr.String(),
		})
	}

	return &instance{
		cfg:                 &cfg,
		workloadClient:      instanceCfg.WorkloadClient,
		metricsProvider:     provider.NewMetricsProvider(), // TODO: Take this in from the config?
		tracker:             instanceCfg.Tracker,
		vkAuth:              instanceCfg.VkAuth,
		enableKubeletServer: enableKubeletServer,
		disableAuth:         instanceCfg.IsOpenShift, // OpenShift/OKD do not support Kubelet HTTP server authentication
	}
}

// Run starts the instance controllers with the given context
func (i *instance) Run(ctx context.Context, cancel func()) error {
	cfg := *i.cfg                                        // Instance configuration, shallow copy for assignment overrides
	nodeName := cfg.NodeSpec.Name                        // Shorthand for node name
	log := sulog.Scoped().Str("node", nodeName).Logger() // Node-specific scoped logger

	// Configure the error handler for status updates
	cfg.NodeStatusUpdateErrorHandler = func(_ context.Context, err error) error {
		if !suerr.IsContextCanceled(err) {
			log.Err(err).Msg("status update failed")
		}

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
	i.podProvider = provider.NewPodProvider(&podProviderLogger, nodeName, i.workloadClient, i.tracker, i.metricsProvider)
	podControllerCfg := node.PodControllerConfig{
		PodClient:         cfg.Client.CoreV1(),
		EventRecorder:     cfg.EventRecorder,
		Provider:          i.podProvider,
		PodInformer:       podInformerFactory.Core().V1().Pods(),
		ConfigMapInformer: scmInformerFactory.Core().V1().ConfigMaps(),
		SecretInformer:    scmInformerFactory.Core().V1().Secrets(),
		ServiceInformer:   scmInformerFactory.Core().V1().Services(),
	}

	podController, err := node.NewPodController(podControllerCfg)
	if err != nil {
		return errors.Wrap(err, "creating pod controller failed")
	}

	vkLogger := log.Level(zerolog.InfoLevel) // TODO: Configurability, VK is noisy
	ctx = vklog.WithLogger(ctx, sulog.VKLogger(&vkLogger, sulog.VKLoggerConfig{
		ClampToDebug:        true,
		SuppressCtxCanceled: true,
	})) // Virtual Kubelet logging

	// Start all informers
	log.Debug().Msg("starting informers")
	go podInformerFactory.Start(ctx.Done())
	go scmInformerFactory.Start(ctx.Done())

	// Start pod controller
	go func() {
		defer cancel()

		// Start recoding pod controller events
		log.Debug().Msg("starting event broadcaster for pod controller")
		podEvents.StartLogging(log.Debug().Str("scope", "pod-events").Msgf)
		podEvents.StartRecordingToSink(&corev1client.EventSinkImpl{
			Interface: cfg.Client.CoreV1().Events(corev1.NamespaceAll),
		})
		defer podEvents.Shutdown()

		log.Debug().Msg("starting pod controller")
		if err := podController.Run(ctx, cfg.NumWorkers); err != nil {
			if !suerr.IsContextCanceled(err) {
				log.Err(err).Msg("running pod controller failed")
			}

			return
		}

		log.Debug().Msg("stopped pod controller")
	}()

	// Wait for pod controller to become ready
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-podController.Ready():
	case <-podController.Done():
		return podController.Err() // Pod controller failed to become ready, return its error
	}

	if i.enableKubeletServer {
		// Set up Kubelet server
		handler := i.podHandlerConfig(podInformerFactory)
		kubeletServer := NewKubeletServer(cfg.Client, handler, i.vkAuth, i.disableAuth, nodeName, func() []corev1.NodeAddress {
			return cfg.NodeSpec.Status.Addresses
		})

		// Start Kubelet server
		go func() {
			defer cancel()
			log.Debug().Msg("starting Kubelet server")
			if err := kubeletServer.Run(ctx, &log); err != nil {
				if !suerr.IsContextCanceled(err) {
					log.Err(err).Msg("running Kubelet server failed")
				}

				return
			}

			log.Debug().Msg("stopped Kubelet server")
		}()

		// Wait for the Kubelet server to become ready
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-kubeletServer.Ready():
		}

		// Update the node daemon endpoint with the port from the Kubelet server
		cfg.NodeSpec.Status.DaemonEndpoints = corev1.NodeDaemonEndpoints{
			KubeletEndpoint: corev1.DaemonEndpoint{
				Port: kubeletServer.Port(),
			},
		}
	}

	// Start node controller
	go func() {
		defer cancel()
		log.Debug().Msg("starting node controller")
		if err := nodeController.Run(ctx); err != nil {
			if !suerr.IsContextCanceled(err) {
				log.Err(err).Msg("running node controller failed")
			}

			return
		}

		// TODO: We need to defer/handle node deletion here, Virtual Kubelet doesn't seem to do it automatically.
		//  Normally, deletion of stale nodes after a timeout would normally be handled by the Cluster Autoscaler
		//  (https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler), but this seems to heavily rely
		//  on cloud-provider-specific APIs and won't work with Talos.
		// TODO: Maybe Supernetes should run another controller/reconciliation loop for handling virtual node pruning?
		// TODO: Another option is the Kyverno cleanup controller (https://kyverno.io/docs/writing-policies/cleanup/)

		log.Debug().Msg("stopped node controller")
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

func (i *instance) UpdateStatus(ctx context.Context, pod *corev1.Pod, cache bool) error {
	// This is a no-op if the instance is not running
	if i.podProvider == nil {
		return nil
	}

	return i.podProvider.UpdateStatus(ctx, pod, cache)
}

func (i *instance) UpdateNodeStatus(status *api.NodeStatus) {
	i.metricsProvider.Update(status)
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

func (i *instance) podHandlerConfig(podInformerFactory informers.SharedInformerFactory) vkapi.PodHandlerConfig {
	return vkapi.PodHandlerConfig{
		RunInContainer:    i.podProvider.RunInContainer,
		AttachToContainer: i.podProvider.AttachToContainer,
		PortForward:       i.podProvider.PortForward,
		GetContainerLogs:  i.podProvider.GetContainerLogs,
		GetPods:           i.podProvider.GetPods,
		GetPodsFromKubernetes: func(context.Context) ([]*corev1.Pod, error) {
			return podInformerFactory.Core().V1().Pods().Lister().List(labels.Everything())
		},
		GetStatsSummary:       i.podProvider.GetStatsSummary,
		GetMetricsResource:    i.podProvider.GetMetricsResource,
		StreamIdleTimeout:     i.cfg.StreamIdleTimeout,     // Defaults to 30s: https://github.com/virtual-kubelet/virtual-kubelet/blob/5c534ffcd6074044b00a5151da84ac2cc8ce3f12/node/api/portforward.go#L73
		StreamCreationTimeout: i.cfg.StreamCreationTimeout, // Defaults to 30s: https://github.com/virtual-kubelet/virtual-kubelet/blob/5c534ffcd6074044b00a5151da84ac2cc8ce3f12/node/api/portforward.go#L76
	}
}
