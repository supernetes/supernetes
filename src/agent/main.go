// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"os/signal"
	"syscall"

	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/supernetes/supernetes/agent/pkg/server"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddr string
	useTls     bool
)

func main() {
	// TODO: Implement full CLI with Cobra in `cmd`
	pflag.StringVarP(&serverAddr, "server", "s", "localhost:40404", "Address of server endpoint")
	pflag.BoolVar(&useTls, "tls", true, "Use TLS to connect to server endpoint")
	pflag.Parse()

	log.Init(zerolog.TraceLevel)
	log.Info().Msg("starting dummy agent")

	log.Info().Msgf("connecting to server %q", serverAddr)

	var transportCredentials credentials.TransportCredentials
	if useTls {
		certPool, err := x509.SystemCertPool()
		if err != nil {
			log.Fatal().Err(err).Msg("unable to load system certificate pool")
		}

		transportCredentials = credentials.NewTLS(&tls.Config{
			RootCAs: certPool,
		})
	} else {
		transportCredentials = insecure.NewCredentials()
	}

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		log.Fatal().Msgf("failed to connect: %v", err)
	}
	defer conn.Close()

	// Register services for reverse tunnels
	tunnelServer := grpctunnel.NewReverseTunnelServer(tunnelpb.NewTunnelServiceClient(conn))
	agentServer := server.NewServer(1, 0.1)
	api.RegisterNodeApiServer(tunnelServer, agentServer)

	controllerDone := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// Open the reverse tunnel and serve requests
		log.Info().Msgf("listening for requests from server")
		if started, err := tunnelServer.Serve(ctx); err != nil {
			msg := "unable to start listening"
			if started {
				msg = "connection closed unexpectedly"
			}
			log.Fatal().Err(err).Msg(msg)
		}

		controllerDone <- struct{}{}
	}()

	agentDone := make(chan os.Signal, 1)
	signal.Notify(agentDone, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-controllerDone:
		log.Info().Msg("controller initiated shutdown")
	case <-agentDone:
		log.Info().Msg("agent initiated shutdown")
	}

	log.Debug().Msg("stopping gRPC tunnel")
	tunnelServer.Stop()

	log.Info().Msg("agent finished")
}
