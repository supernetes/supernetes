// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package scontrol

// Number represents a numeric value type in the `scontrol` JSON output
type Number struct {
	Set      bool    `json:"set"`
	Infinite bool    `json:"infinite"`
	Number   float32 `json:"number"`
}

type Plugin struct {
	Type              string `json:"type,omitempty"` // Present on Mahti, Absent on LUMI
	Name              string `json:"name,omitempty"` // Present on Mahti, Absent on LUMI
	DataParser        string `json:"data_parser"`
	AccountingStorage string `json:"accounting_storage"`
}

type Client struct {
	Source string `json:"source"`
	User   string `json:"user"`
	Group  string `json:"group"`
}

type Version struct {
	Major string `json:"major"`
	Minor string `json:"minor"`
	Micro string `json:"micro"`
}

type Slurm struct {
	Version Version `json:"version"`
	Release string  `json:"release"`
	Cluster string  `json:"cluster,omitempty"` // Present on Mahti, Absent on LUMI
}

// TODO: Implement custom parser that unifies the fields
type Meta struct {
	Plugin  *Plugin  `json:"plugin,omitempty"`  // Present on Mahti, and the reference slurmrestd docs
	Plugins *Plugin  `json:"plugins,omitempty"` // Same as "plugin", but only present on LUMI...
	Client  *Client  `json:"client,omitempty"`  // Present on Mahti, absent on LUMI
	Command []string `json:"command"`
	Slurm   *Slurm   `json:"slurm,omitempty"` // Present on Mahti, and the reference slurmrestd docs
	Slurm2  *Slurm   `json:"Slurm,omitempty"` // Same as "slurm", but only present on LUMI...
}
