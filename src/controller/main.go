// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	suconfig "github.com/supernetes/supernetes/config/pkg/config"
	"github.com/supernetes/supernetes/controller/pkg/client"
	"github.com/supernetes/supernetes/controller/pkg/endpoint"
	"github.com/supernetes/supernetes/controller/pkg/node"
	"github.com/supernetes/supernetes/controller/pkg/vk"
	"github.com/supernetes/supernetes/controller/pkg/workload"
	"github.com/supernetes/supernetes/util/pkg/log"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	logLevel   string
	configPath string
)

func main() {
	pflag.StringVarP(&logLevel, "log-level", "l", "info", "Log level") // TODO: Persistently default to "info"
	pflag.StringVarP(&configPath, "config", "c", "", "path to controller configuration file (mandatory)")
	pflag.Parse()

	log.Init(logLevel)
	crlog.SetLogger(log.CRLogger(nil))

	// Configuration file path must be provided
	if len(configPath) == 0 {
		pflag.Usage()
		os.Exit(1)
	}

	log.Debug().Str("path", configPath).Msg("reading configuration file")
	configBytes, err := os.ReadFile(configPath)
	log.FatalErr(err).Str("path", configPath).Msg("unable to read configuration file")

	log.Debug().Msg("decoding configuration file")
	config, err := suconfig.Decode[suconfig.ControllerConfig](configBytes)
	log.FatalErr(err).Msg("decoding configuration file failed")

	ep := endpoint.Serve(config)
	defer ep.Close()

	k8sConfig, err := client.NewK8sConfig()
	log.FatalErr(err).Msg("failed to create K8s client")

	log.FatalErr(vk.DisableKubeProxy(k8sConfig)).Msg("disabling kube-proxy for Virtual Kubelet nodes failed")

	ctx := context.Background()
	nodeReconciler, err := node.NewReconciler(ctx, node.ReconcilerConfig{
		Interval:  time.Minute,
		Client:    ep.Node(),
		K8sConfig: k8sConfig,
	})
	log.FatalErr(err).Msg("failed to create node reconciler")
	workloadReconciler, err := workload.NewReconciler(ctx, workload.ReconcilerConfig{
		Interval:  time.Minute,
		Client:    ep.Workload(),
		K8sConfig: k8sConfig,
	})
	log.FatalErr(err).Msg("failed to create workload reconciler")

	// Use callbacks to automatically start/stop the control loops
	ep.SetCallbacks(endpoint.Callbacks{
		OnConnected: func() {
			nodeReconciler.Start()
			workloadReconciler.Start()
		},
		OnIdle: func() {
			nodeReconciler.Stop()
			workloadReconciler.Stop()
		},
	})

	defer nodeReconciler.Stop()
	defer workloadReconciler.Stop()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	<-done
}
