// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package node

import "github.com/supernetes/supernetes/agent/pkg/scontrol"

// TODO: Make sure all the UNIX timestamp fields are int64

type Data struct {
	Meta     Meta   `json:"meta"`
	Nodes    []Node `json:"nodes"`
	Warnings []any  `json:"warnings"`
	Errors   []any  `json:"errors"`
}

func (d *Data) GetWarnings() []any {
	return d.Warnings
}

func (d *Data) GetErrors() []any {
	return d.Errors
}

type Plugins struct {
	DataParser        string `json:"data_parser"`
	AccountingStorage string `json:"accounting_storage"`
}

type Version struct {
	Major int `json:"major"`
	Micro int `json:"micro"`
	Minor int `json:"minor"`
}

type Slurm struct {
	Version Version `json:"version"`
	Release string  `json:"release"`
}

type Meta struct {
	Plugins Plugins  `json:"plugins"`
	Command []string `json:"command"`
	Slurm   Slurm    `json:"Slurm"`
}

type Energy struct {
	AverageWatts           int             `json:"average_watts"`
	BaseConsumedEnergy     int             `json:"base_consumed_energy"`
	ConsumedEnergy         int             `json:"consumed_energy"`
	CurrentWatts           scontrol.Number `json:"current_watts"`
	PreviousConsumedEnergy int             `json:"previous_consumed_energy"`
	LastCollected          int             `json:"last_collected"`
}

type ExternalSensors struct {
	ConsumedEnergy   scontrol.Number `json:"consumed_energy"`
	Temperature      scontrol.Number `json:"temperature"`
	EnergyUpdateTime int             `json:"energy_update_time"`
	CurrentWatts     int             `json:"current_watts"`
}

type Power struct {
	MaximumWatts    scontrol.Number `json:"maximum_watts"`
	CurrentWatts    int             `json:"current_watts"`
	TotalEnergy     int             `json:"total_energy"`
	NewMaximumWatts int             `json:"new_maximum_watts"`
	PeakWatts       int             `json:"peak_watts"`
	LowestWatts     int             `json:"lowest_watts"`
	NewJobTime      int             `json:"new_job_time"`
	State           int             `json:"state"`
	TimeStartDay    int             `json:"time_start_day"`
}

type Node struct {
	Architecture              string          `json:"architecture"`
	BurstbufferNetworkAddress string          `json:"burstbuffer_network_address"`
	Boards                    int             `json:"boards"`
	BootTime                  int             `json:"boot_time"`
	ClusterName               string          `json:"cluster_name"`
	Cores                     int             `json:"cores"`
	SpecializedCores          int             `json:"specialized_cores"`
	CPUBinding                int             `json:"cpu_binding"`
	CPULoad                   scontrol.Number `json:"cpu_load"`
	FreeMem                   scontrol.Number `json:"free_mem"`
	Cpus                      int             `json:"cpus"`
	EffectiveCpus             int             `json:"effective_cpus"`
	SpecializedCpus           string          `json:"specialized_cpus"`
	Energy                    Energy          `json:"energy"`
	ExternalSensors           ExternalSensors `json:"external_sensors"`
	Extra                     string          `json:"extra"`
	Power                     Power           `json:"power"`
	Features                  []string        `json:"features"`
	ActiveFeatures            []string        `json:"active_features"`
	Gres                      string          `json:"gres"`
	GresDrained               string          `json:"gres_drained"`
	GresUsed                  string          `json:"gres_used"`
	LastBusy                  int             `json:"last_busy"`
	McsLabel                  string          `json:"mcs_label"`
	SpecializedMemory         int             `json:"specialized_memory"`
	Name                      string          `json:"name"`
	NextStateAfterReboot      []string        `json:"next_state_after_reboot"`
	Address                   string          `json:"address"`
	Hostname                  string          `json:"hostname"`
	State                     []string        `json:"state"`
	OperatingSystem           string          `json:"operating_system"`
	Owner                     string          `json:"owner"`
	Partitions                []string        `json:"partitions"`
	Port                      int             `json:"port"`
	RealMemory                int             `json:"real_memory"`
	Comment                   string          `json:"comment"`
	Reason                    string          `json:"reason"`
	ReasonChangedAt           int             `json:"reason_changed_at"`
	ReasonSetByUser           string          `json:"reason_set_by_user"`
	ResumeAfter               scontrol.Number `json:"resume_after"`
	Reservation               string          `json:"reservation"`
	AllocMemory               int             `json:"alloc_memory"`
	AllocCpus                 int             `json:"alloc_cpus"`
	AllocIdleCpus             int             `json:"alloc_idle_cpus"`
	TresUsed                  string          `json:"tres_used"`
	TresWeighted              float64         `json:"tres_weighted"`
	SlurmdStartTime           int             `json:"slurmd_start_time"`
	Sockets                   int             `json:"sockets"`
	Threads                   int             `json:"threads"`
	TemporaryDisk             int             `json:"temporary_disk"`
	Weight                    int             `json:"weight"`
	Tres                      string          `json:"tres"`
	Version                   string          `json:"version"`
}
