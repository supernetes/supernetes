// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package server

import (
	"fmt"

	"github.com/goombaio/namegenerator"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// server is used to implement supernetes.NodeApiServer
type server struct {
	api.UnimplementedNodeApiServer
	generator         namegenerator.Generator
	changeProbability float32
	nodes             []string
}

func NewServer(nodeCount int, changeProb float32) api.NodeApiServer {
	nodes := make([]string, nodeCount)

	//generator := namegenerator.NewNameGenerator(time.Now().UTC().UnixNano())
	//for i := 0; i < nodeCount; i++ {
	//	nodes[i] = generator.Generate()
	//}

	for i := 0; i < nodeCount; i++ {
		nodes[i] = fmt.Sprintf("test%d", i)
	}

	return &server{
		generator:         nil,
		changeProbability: changeProb,
		nodes:             nodes,
	}
}

func (s *server) GetNodes(_ *emptypb.Empty, a api.NodeApi_GetNodesServer) error {
	log.Debug().Msg("received node list request")

	//for i := 0; i < len(s.nodes); i++ {
	//	if rand.Float32() < s.changeProbability {
	//		s.nodes[i] = s.generator.Generate()
	//	}
	//}

	// TODO: Just a dummy implementation for now
	log.Debug().Msgf("sending node list: %v", s.nodes)

	for _, n := range s.nodes {
		if err := a.Send(toNode(n)); err != nil {
			return err
		}
	}

	return nil
}

func toNode(name string) *api.Node {
	return &api.Node{
		Meta: &api.NodeMeta{
			Name: name,
		},
		Spec: &api.NodeSpec{},
	}
}
