// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package config

import "regexp"

// AgentConfig encapsulates all relevant configuration for deploying an agent
// TODO: Versioning
type AgentConfig struct {
	// Controller endpoint that the agent should connect to. Format specification:
	// https://github.com/grpc/grpc/blob/18c42a21af2331c4c755257a968490ab74c587b7/doc/naming.md
	Endpoint    string      `json:"endpoint"`
	MTlsConfig  MTlsConfig  `json:"mTLSConfig"`  // mTLS configuration for the agent
	SlurmConfig SlurmConfig `json:"slurmConfig"` // Slurm configuration (required until other providers are implemented)
}

// SlurmConfig configures how to interact with Slurm
type SlurmConfig struct {
	Account   string  `json:"account"`   // Default Slurm account to use for dispatching jobs
	Partition string  `json:"partition"` // Default Slurm partition to use for dispatching jobs
	Filter    *Filter `json:"filter,omitempty"`
}

// Filter is used to limit the HPC-side resources exposed by the agent
type Filter struct {
	PartitionRegex regexp.Regexp `json:"partitionRegex"` // Match specific partitions
	NodeRegex      regexp.Regexp `json:"nodeRegex"`      // Match specific node names
}
