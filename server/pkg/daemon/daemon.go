// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package daemon

import (
	"fmt"
	"net"

	"github.com/supernetes/supernetes/api"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/server/pkg/config"
	"github.com/supernetes/supernetes/server/pkg/server"
	"google.golang.org/grpc"
)

func Run(config *config.DaemonConfig) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	apiServer, err := server.NewServer()
	if err != nil {
		return fmt.Errorf("failed to create server: %v", err)
	}

	grpcServer := grpc.NewServer()
	api.RegisterNodeApiServer(grpcServer, apiServer)
	log.Info().Msgf("server listening at %v", listener.Addr())
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}

	return nil
}
