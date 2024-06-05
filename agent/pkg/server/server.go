// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package server

import (
	"context"
	"math/rand"
	"time"

	"github.com/goombaio/namegenerator"
	"github.com/supernetes/supernetes/api"
	"github.com/supernetes/supernetes/common/pkg/log"
)

// server is used to implement supernetes.NodeApiServer
type server struct {
	api.UnimplementedAgentServer
	generator         namegenerator.Generator
	changeProbability float32
	nodes             []string
}

func NewServer(nodeCount int, changeProb float32) api.AgentServer {
	nodes := make([]string, nodeCount)

	generator := namegenerator.NewNameGenerator(time.Now().UTC().UnixNano())
	for i := 0; i < nodeCount; i++ {
		nodes[i] = generator.Generate()
	}

	return &server{
		generator:         generator,
		changeProbability: changeProb,
		nodes:             nodes,
	}
}

func (s *server) List(_ context.Context, _ *api.Empty) (*api.NodeList, error) {
	log.Debug().Msg("received node list request")

	for i := 0; i < len(s.nodes); i++ {
		if rand.Float32() < s.changeProbability {
			s.nodes[i] = s.generator.Generate()
		}
	}

	// TODO: Just a dummy implementation for now
	log.Debug().Msgf("sending node list: %v", s.nodes)
	return toNodeList(s.nodes), nil
}

func toNodeList(names []string) *api.NodeList {
	nodes := make([]*api.Node, 0, len(names))

	for _, n := range names {
		nodes = append(nodes, &api.Node{Name: n})
	}

	return &api.NodeList{Nodes: nodes}
}
