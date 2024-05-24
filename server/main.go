// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/supernetes/supernetes/common/pkg/log"
	"github.com/supernetes/supernetes/server/pkg/config"
	"github.com/supernetes/supernetes/server/pkg/daemon"
)

var (
	port     uint16
	logLevel string
)

func main() {
	// TODO: Implement full CLI with Cobra in `cmd`
	pflag.Uint16VarP(&port, "port", "p", 40404, "Server port")
	pflag.StringVarP(&logLevel, "log-level", "l", "trace", "Log level") // TODO: Change to "info"
	pflag.Parse()

	level, err := zerolog.ParseLevel(logLevel)
	log.Init(level) // `level` is always well-defined
	if err != nil {
		log.Warn().Err(err).Msg("parsing log level failed")
	}

	dc := &config.DaemonConfig{
		Port: port,
	}

	if err := daemon.Run(dc); err != nil {
		log.Fatal().Err(err).Msg("")
	}
}
