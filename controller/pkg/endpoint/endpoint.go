// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package endpoint

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	"github.com/supernetes/supernetes/api"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/controller/pkg/config"
	"google.golang.org/grpc"
)

// Endpoint represents the network endpoint that remote agents connect to.
type Endpoint interface {
	Client() api.AgentClient
	Close()
}

// endpoint implements the Endpoint interface
type endpoint struct {
	client     api.AgentClient
	closing    atomic.Bool
	grpcServer *grpc.Server
	handler    *grpctunnel.TunnelServiceHandler
}

// Compile-time type check
var _ Endpoint = &endpoint{}

// Serve creates and serves an Endpoint according to the given configuration
func Serve(config *config.Controller) (Endpoint, error) {
	srv := &endpoint{}

	// Create handler for reverse tunnels
	srv.handler = grpctunnel.NewTunnelServiceHandler(
		grpctunnel.TunnelServiceHandlerOptions{
			NoReverseTunnels: false,
			OnReverseTunnelOpen: func(channel grpctunnel.TunnelChannel) {
				log.Debug().Msg("reverse tunnel opened")
				if srv.closing.Load() {
					log.Debug().Msg("rejecting connection to closing endpoint")
					channel.Close()
				}
			},
			OnReverseTunnelClose: func(_ grpctunnel.TunnelChannel) {
				log.Debug().Msg("reverse tunnel closed")
			},
			AffinityKey:        nil,
			DisableFlowControl: false,
		},
	)

	// TODO: Multitenancy, this will round-robin through all reverse tunnels. Use handler.KeyAsChannel()
	//  instead with an AffinityKey function above to scope it to only clients of a particular HPC environment

	// TODO: Potentially, if Cilium is able to do TLS termination and apply L7 logic to gRPC traffic, we could
	//  also just route different agents (HPC environments) to different controller instances -> better scalability

	srv.client = api.NewAgentClient(srv.handler.AsChannel())

	// Register reverse tunnel handler to a server that the agents can connect to
	srv.grpcServer = grpc.NewServer()
	tunnelpb.RegisterTunnelServiceServer(srv.grpcServer, srv.handler.Service())

	go func() {
		// Start serving the endpoint
		if err := srv.serve(config.Port); err != nil {
			log.Fatal().Err(err).Msg("gRPC endpoint error")
		}

		log.Debug().Msg("gRPC endpoint closed")
	}()

	return srv, nil
}

// serve synchronously serves the endpoint on the given port
func (s *endpoint) serve(port uint16) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	log.Info().Msgf("endpoint listening at %v", listener.Addr())
	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

// Client returns the API client for sending RPCs to the agents
func (s *endpoint) Client() api.AgentClient {
	return s.client
}

// Close disconnects all clients and stops the endpoint
func (s *endpoint) Close() {
	s.handler.InitiateShutdown() // This is basically a no-op with only reverse tunnels
	s.closing.Store(true)        // Prevent new connections from being established

	// Close all existing reverse tunnels (disconnect clients)
	for _, c := range s.handler.AllReverseTunnels() {
		c.Close()
	}

	// Wait for graceful stop
	s.grpcServer.GracefulStop()
}
