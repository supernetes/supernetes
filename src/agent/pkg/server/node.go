// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package server

import (
	"slices"

	"github.com/pkg/errors"
	"github.com/supernetes/supernetes/agent/pkg/filter"
	"github.com/supernetes/supernetes/agent/pkg/node"
	api "github.com/supernetes/supernetes/api/v1alpha1"
	"github.com/supernetes/supernetes/common/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// nodeServer is used to implement supernetes.NodeApiServer
type nodeServer struct {
	api.UnimplementedNodeApiServer
	filter filter.Filter
}

func NewNodeApiServer(filter filter.Filter) api.NodeApiServer {
	return &nodeServer{
		filter: filter,
	}
}

func (s *nodeServer) GetNodes(_ *emptypb.Empty, a grpc.ServerStreamingServer[api.Node]) error {
	log.Debug().Msg("GetNodes invoked")

	nodeData, err := node.ReadNodeData(nil)
	if err != nil {
		msg := "failed to read node data from Slurm"
		log.Err(err).Msg(msg)
		return errors.New(msg)
	}

	filteredCount := 0
	for _, n := range nodeData.Nodes {
		if !s.filter.Node(n.Name) {
			continue // Node name excluded by filter
		}

		if !slices.ContainsFunc(n.Partitions, s.filter.Partition) {
			continue // Node not in filtered partitions
		}

		if err := a.Send(toNode(&n)); err != nil {
			return err
		}
		filteredCount++
	}

	log.Debug().
		Int("all", len(nodeData.Nodes)).
		Int("filtered", filteredCount).
		Msg("sent node list")

	return nil
}

// TODO: This needs to populate all available fields
func toNode(node *node.Node) *api.Node {
	realMem := uint64(node.RealMemory) * 1024 * 1024        // Slurm says this is in "MB", I'm assuming MiB
	freeMem := uint64(node.FreeMem.ToFloat() * 1024 * 1024) // Slurm says this is in "MB", I'm assuming MiB
	usedMem := realMem - freeMem
	if freeMem > realMem {
		// For some reason, on some nodes the free memory can be larger than real memory, fall back to allocated memory
		// in this case. Slurm gives us no details on what these values actually represent or how they're computed.
		usedMem = uint64(node.AllocMemory) * 1024 * 1024
	}

	return &api.Node{
		Meta: &api.NodeMeta{
			Name: node.Name,
		},
		Spec: &api.NodeSpec{
			CpuCount: uint32(node.Cpus),
			MemBytes: realMem,
		},
		Status: &api.NodeStatus{
			// Seems to be fixed point, at least on LUMI
			CpuLoad: node.CPULoad.ToFloat() / 100,
			// This is not accurate, since Slurm has no way to retrieve the
			// precise memory use on the node, let alone the working set.
			WsBytes: usedMem,
		},
	}
}
