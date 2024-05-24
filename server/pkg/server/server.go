// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package server

import (
	"context"
	"fmt"

	"github.com/supernetes/supernetes/api"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/server/pkg/vk"
)

// Server is used to implement supernetes.NodeServiceServer.
type Server struct {
	api.UnimplementedNodeApiServer
	manager *vk.Manager
}

func NewServer() (*Server, error) {
	manager, err := vk.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create VK manager: %v", err)
	}

	return &Server{
		manager: manager,
	}, nil
}

func (s *Server) List(_ context.Context, list *api.NodeList) (*api.Empty, error) {
	log.Debug().Msgf("received node list: %v", list.Nodes)

	if err := s.manager.Reconcile(list.Nodes); err != nil {
		log.Err(err).Msg("failed to reconcile nodes")
		return nil, fmt.Errorf("error processing node list")
	}

	return nil, nil
}
