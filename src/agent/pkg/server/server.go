// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package server

import (
	"regexp"
	"slices"

	"github.com/pkg/errors"
	"github.com/supernetes/supernetes/agent/pkg/scontrol"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/util/pkg/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// server is used to implement supernetes.NodeApiServer
type server struct {
	api.UnimplementedNodeApiServer
}

func NewServer() api.NodeApiServer {
	return &server{}
}

func (s *server) GetNodes(_ *emptypb.Empty, a api.NodeApi_GetNodesServer) error {
	log.Debug().Msg("received node list request")

	partition := "standard" // TODO: Pass this from the controller

	nodeInfo, err := scontrol.ReadNodeInfo()
	if err != nil {
		return errors.WithMessage(err, "unable to read node info")
	}

	// TODO: Temporarily cap the number of nodes
	pattern := `^nid001[0-9]{3}$`
	//pattern := `^nid00100[0-9]{1}$`
	regex := regexp.MustCompile(pattern)
	log.Debug().Str("pattern", pattern).Msg("TODO: limiting filtered nodes with regex")

	var filteredNodes []scontrol.Node
	for _, n := range nodeInfo.Nodes {
		if slices.Contains(n.Partitions, partition) {
			if ok := regex.Match([]byte(n.Name)); !ok {
				continue
			}

			filteredNodes = append(filteredNodes, n)
		}
	}

	log.Debug().
		Int("all", len(nodeInfo.Nodes)).
		Int("filtered", len(filteredNodes)).
		Msg("sending node list")

	for _, n := range filteredNodes {
		if err := a.Send(toNode(&n)); err != nil {
			return err
		}
	}

	return nil
}

// TODO: This needs to populate all available fields
func toNode(node *scontrol.Node) *api.Node {
	return &api.Node{
		Meta: &api.NodeMeta{
			Name: node.Name,
		},
		Spec: &api.NodeSpec{},
	}
}
