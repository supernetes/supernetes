// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package endpoint

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	suconfig "github.com/supernetes/supernetes/config/pkg/config"
	"github.com/supernetes/supernetes/util/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Callbacks invoke functions depending on endpoint state transitions
type Callbacks struct {
	OnConnected func() // OnConnected will be invoked every time an agent connects
	OnIdle      func() // OnIdle will be invoked once all agents have disconnected
}

// Endpoint represents the network endpoint that remote agents connect to.
type Endpoint interface {
	Node() api.NodeApiClient
	Workload() api.WorkloadApiClient
	SetCallbacks(Callbacks)
	Close()
}

// endpoint implements the Endpoint interface
type endpoint struct {
	nodeClient     api.NodeApiClient
	workloadClient api.WorkloadApiClient
	closing        atomic.Bool
	grpcServer     *grpc.Server
	handler        *grpctunnel.TunnelServiceHandler
	callbacks      Callbacks
}

// Compile-time type check
var _ Endpoint = &endpoint{}

/*
TODO: If there are multiple agents, they all need to have unanimity in what the Slurm/filtering configuration should be,
 otherwise, stuff is not going to work properly. If the agents are supposed to be mostly stateless, then these options
 need to be configured through the controller, but that breaks separation of responsibilty a bit since the controller
 itself does not interact with Slurm directly, and aims to be agnostic of it (to support, e.g., HTCondor later). Maybe
 the smartest way would be to have the agents configure themselves, but not require all agents to have the same Slurm
 options etc. at some point? Since that is probably overkill, it might be reasonable to simply hash the configuration of
 the first connecting agent, and then enforce all other agents to also have that configuration. For now, the simplest
 solution is to simply limit the controller to only accept one agent connection at a time.

TODO: Actually, the GRPC tunnel system supports grouping reverse tunnels with a key, that key should probably ultimately
 be the configuration hash so that all agents acting together will be automatically grouped.
*/

// Serve creates and serves an Endpoint according to the given configuration
func Serve(config *suconfig.ControllerConfig) Endpoint {
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

				if srv.callbacks.OnConnected != nil {
					srv.callbacks.OnConnected()
				}
			},
			OnReverseTunnelClose: func(_ grpctunnel.TunnelChannel) {
				log.Debug().Msg("reverse tunnel closed")

				// This will be invoked once the last reverse tunnel is closed
				if !srv.handler.AsChannel().Ready() {
					if srv.callbacks.OnIdle != nil {
						srv.callbacks.OnIdle()
					}
				}
			},
			AffinityKey:        nil,
			DisableFlowControl: false,
		},
	)

	// TODO: Multitenancy, this will round-robin through all reverse tunnels. Use handler.KeyAsChannel()
	//  instead with an AffinityKey function above to scope it to only clients of a particular HPC environment

	// TODO: Potentially, if Cilium is able to do TLS termination and apply L7 logic to gRPC traffic, we could
	//  also just route different agents (HPC environments) to different controller instances -> better scalability

	srv.nodeClient = api.NewNodeApiClient(srv.handler.AsChannel())
	srv.workloadClient = api.NewWorkloadApiClient(srv.handler.AsChannel())

	// Register reverse tunnel handler to a server that the agents can connect to
	srv.grpcServer = grpc.NewServer(loadCreds(&config.MTlsConfig))
	tunnelpb.RegisterTunnelServiceServer(srv.grpcServer, srv.handler.Service())

	go func() {
		// Start serving the endpoint
		log.FatalErr(srv.serve(config.Port)).Msg("gRPC endpoint error")
		log.Debug().Msg("gRPC endpoint closed")
	}()

	return srv
}

// loadCreds sets up TLS for a GRPC server from the given mTLS configuration
func loadCreds(mTlsConfig *suconfig.MTlsConfig) grpc.ServerOption {
	cert, err := tls.X509KeyPair([]byte(mTlsConfig.Cert), []byte(mTlsConfig.Key))
	log.FatalErr(err).Msg("failed to load server key pair")

	ca := x509.NewCertPool()
	if ok := ca.AppendCertsFromPEM([]byte(mTlsConfig.Ca)); !ok {
		log.Fatal().Msg("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // Enforce mTLS
		ClientCAs:    ca,
	}

	return grpc.Creds(credentials.NewTLS(tlsConfig))
}

// serve synchronously serves the endpoint on the given port
func (e *endpoint) serve(port uint16) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	log.Info().Msgf("endpoint listening at %v", listener.Addr())
	if err := e.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}

// Node returns an API client for sending RPCs to the agents
func (e *endpoint) Node() api.NodeApiClient {
	return e.nodeClient
}

// Workload returns an API client for sending RPCs to the agents
func (e *endpoint) Workload() api.WorkloadApiClient {
	return e.workloadClient
}

// SetCallbacks sets the state transition callbacks
func (e *endpoint) SetCallbacks(callbacks Callbacks) {
	e.callbacks = callbacks
}

// Close disconnects all clients and stops the endpoint
func (e *endpoint) Close() {
	e.handler.InitiateShutdown() // This is basically a no-op with only reverse tunnels
	e.closing.Store(true)        // Prevent new connections from being established

	// Close all existing reverse tunnels (disconnect clients)
	for _, c := range e.handler.AllReverseTunnels() {
		c.Close()
	}

	// Wait for graceful stop
	e.grpcServer.GracefulStop()
}
