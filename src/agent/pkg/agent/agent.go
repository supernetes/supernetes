// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package agent

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"os/signal"
	"syscall"

	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	"github.com/pkg/errors"
	"github.com/supernetes/supernetes/agent/pkg/sbatch"
	"github.com/supernetes/supernetes/agent/pkg/server"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/common/pkg/util"
	suconfig "github.com/supernetes/supernetes/config/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type Flags struct {
	Options
}

func NewFlags() *Flags {
	return &Flags{}
}

type Options struct {
	ConfigPath string
}

func (f *Flags) NewOptions(_ []string) (*Options, error) {
	if f.ConfigPath == "" {
		return nil, errors.New("path to agent configuration file must be provided")
	}

	return &f.Options, nil
}

func Run(options *Options) {
	log.Debug().Str("path", options.ConfigPath).Msg("reading configuration file")
	configBytes, err := os.ReadFile(options.ConfigPath)
	log.FatalErr(err).Msg("unable to read configuration file")

	log.Debug().Msg("decoding configuration")
	config, err := suconfig.Decode[suconfig.AgentConfig](configBytes)
	log.FatalErr(err).Msg("decoding configuration failed")

	// Sanity check: this is required for Supernetes to track its own jobs
	if !config.SlurmConfig.Filter.Partition(config.SlurmConfig.Partition) {
		log.Fatal().Msg("Slurm partition filter must match default Slurm partition")
	}

	log.Info().Msg("starting Supernetes agent")

	log.Info().Msgf("connecting to endpoint %q", config.Endpoint)
	conn, err := grpc.NewClient(config.Endpoint, loadCreds(config))
	log.FatalErr(err).Msg("failed to connect")
	defer func() { log.FatalErr(conn.Close()).Msg("failed to close connection") }()

	// Create the sbatch runtime
	runtime := sbatch.NewRuntime(&config.SlurmConfig)

	// Register services for reverse tunnels
	tunnelServer := grpctunnel.NewReverseTunnelServer(tunnelpb.NewTunnelServiceClient(conn))
	nodeApiServer := server.NewNodeApiServer(config.SlurmConfig.Filter)
	workloadApiServer := server.NewWorkloadServer(runtime, config.SlurmConfig.Filter)
	api.RegisterNodeApiServer(tunnelServer, nodeApiServer)
	api.RegisterWorkloadApiServer(tunnelServer, workloadApiServer)

	controllerDone := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// Open the reverse tunnel and serve requests
		log.Info().Msgf("listening for requests from controller")
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

func loadCreds(config *suconfig.AgentConfig) grpc.DialOption {
	mTlsConfig := &config.MTlsConfig

	cert, err := tls.X509KeyPair([]byte(mTlsConfig.Cert), []byte(mTlsConfig.Key))
	log.FatalErr(err).Msg("failed to load client key pair")

	ca := x509.NewCertPool()
	if ok := ca.AppendCertsFromPEM([]byte(mTlsConfig.Ca)); !ok {
		log.Fatal().Msg("failed to parse CA certificate")
	}

	hostname, err := util.Hostname(config.Endpoint)
	log.FatalErr(err).Msg("failed to parse hostname from endpoint")

	tlsConfig := &tls.Config{
		MinVersion:   tls.VersionTLS13,
		ServerName:   hostname,
		Certificates: []tls.Certificate{cert},
		RootCAs:      ca,
	}

	return grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig))
}
