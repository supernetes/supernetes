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

	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/controller/pkg/config"
	"github.com/supernetes/supernetes/controller/pkg/endpoint"
	"github.com/supernetes/supernetes/controller/pkg/vk"
	"google.golang.org/protobuf/types/known/emptypb"
)

var (
	port     uint16
	logLevel string
)

func main() {
	// TODO: Implement full CLI with Cobra in `cmd`
	pflag.Uint16VarP(&port, "port", "p", 40404, "Server port")
	pflag.StringVarP(&logLevel, "log-level", "l", "trace", "Log level") // TODO: Change to "info"
	pflag.Parse()

	level, err := zerolog.ParseLevel(logLevel)
	log.Init(level) // `level` is always well-defined
	if err != nil {
		log.Warn().Err(err).Msg("parsing log level failed")
	}

	conf := &config.Controller{
		Port: port,
	}

	ep, err := endpoint.Serve(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create endpoint")
	}
	defer ep.Close()

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(30 * time.Second)

	manager, err := vk.NewManager()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create manager")
	}

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
