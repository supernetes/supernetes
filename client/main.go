// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/goombaio/namegenerator"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/supernetes/supernetes/api"
	"github.com/supernetes/supernetes/common/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	server string
)

func main() {
	// TODO: Implement full CLI with Cobra in `cmd`
	pflag.StringVarP(&server, "server", "s", "localhost:40404", "Address of server endpoint")
	pflag.Parse()

	log.Init(zerolog.TraceLevel)
	log.Info().Msg("starting dummy client")

	conn, err := grpc.NewClient(server, grpc.WithTransportCredentials(insecure.NewCredentials())) // TODO: TLS
	if err != nil {
		log.Fatal().Msgf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := api.NewNodeApiClient(conn)

	nodeCount := 10
	changeProbability := float32(0.1)

	nameGenerator := namegenerator.NewNameGenerator(time.Now().UTC().UnixNano())
	fakeNodes := make([]string, nodeCount)
	for i := 0; i < nodeCount; i++ {
		fakeNodes[i] = nameGenerator.Generate()
	}

	fmt.Println(len(fakeNodes))

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	ticker := time.NewTicker(30 * time.Second)

done:
	for {
		listNodes(client, fakeNodes)

		select {
		case <-ticker.C:
			for i := 0; i < nodeCount; i++ {
				if rand.Float32() < changeProbability {
					fakeNodes[i] = nameGenerator.Generate()
				}
			}
		case <-done:
			ticker.Stop()
			break done
		}
	}
}

func listNodes(client api.NodeApiClient, nodes []string) {
	nodeList := &api.NodeList{
		Nodes: make([]*api.Node, len(nodes)),
	}

	for i, v := range nodes {
		nodeList.Nodes[i] = &api.Node{Name: v}
	}

	log.Info().Msgf("sending node list: %v", nodes)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := client.List(ctx, nodeList)
	if err != nil {
		log.Fatal().Err(err).Msg("sending node list failed")
	}
}
