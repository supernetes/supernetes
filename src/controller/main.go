// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"errors"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	suconfig "github.com/supernetes/supernetes/config/pkg/config"
	"github.com/supernetes/supernetes/controller/pkg/client"
	"github.com/supernetes/supernetes/controller/pkg/controller"
	"github.com/supernetes/supernetes/controller/pkg/endpoint"
	"github.com/supernetes/supernetes/controller/pkg/vk"
	"github.com/supernetes/supernetes/util/pkg/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	logLevel   string
	configPath string
)

func main() {
	pflag.StringVarP(&logLevel, "log-level", "l", "trace", "Log level") // TODO: Change to "info"
	pflag.StringVarP(&configPath, "config", "c", "", "path to controller configuration file (mandatory)")
	pflag.Parse()

	log.Init(logLevel)

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

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(10 * time.Second)

	k8sClient, err := client.NewK8sClient()
	log.FatalErr(err).Msg("failed to create K8s client")

	log.FatalErr(vk.DisableKubeProxy(k8sClient)).Msg("disabling kube-proxy for Virtual Kubelet nodes failed")

	manager := controller.NewManager(context.Background(), k8sClient)

done:
	for {
		log.Debug().Msg("requesting list of nodes")
		nodeList, err := ep.Node().GetNodes(context.Background(), &emptypb.Empty{})
		if err != nil {
			log.Err(err).Msg("")
		} else {
			nodes := make([]*api.Node, 0)
			for {
				n, err := nodeList.Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}

					log.Fatal().Err(err).Msg("receiving nodes failed")
				}

				nodes = append(nodes, n)
			}

			log.Debug().Msg("reconciling received nodes")
			if err := manager.Reconcile(nodes); err != nil {
				log.Err(err).Msg("")
			}
		}

		select {
		case <-ticker.C:
		case <-done:
			ticker.Stop()
			break done
		}
	}
}
