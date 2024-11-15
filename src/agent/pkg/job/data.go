// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package job

import "github.com/supernetes/supernetes/agent/pkg/scontrol"

// TODO: Make sure all the UNIX timestamp fields are at least int64

type Data struct {
	Meta         scontrol.Meta   `json:"meta"`
	Jobs         []Job           `json:"jobs"`
	LastBackfill scontrol.Number `json:"last_backfill,omitempty"` // Present on Mahti, absent on LUMI
	LastUpdate   scontrol.Number `json:"last_update,omitempty"`   // Present on Mahti, absent on LUMI
	Warnings     []any           `json:"warnings"`
	Errors       []any           `json:"errors"`
}

func (d *Data) GetWarnings() []any {
	return d.Warnings
}

func (d *Data) GetErrors() []any {
	return d.Errors
}

type Power struct {
	Flags []any `json:"flags"`
}

type Socket struct {
	Cores map[int]string `json:"cores"`
}

type AllocatedNode struct {
	Sockets         map[int]Socket `json:"sockets"`
	Nodename        string         `json:"nodename"`
	CpusUsed        int            `json:"cpus_used"`
	MemoryUsed      int            `json:"memory_used"`
	MemoryAllocated int            `json:"memory_allocated"`
}

type JobResources struct {
	Nodes          string          `json:"nodes"`
	AllocatedCores int             `json:"allocated_cores"`
	AllocatedCpus  int             `json:"allocated_cpus"`
	AllocatedHosts int             `json:"allocated_hosts"`
	AllocatedNodes []AllocatedNode `json:"allocated_nodes"`
}

type Job struct {
	Account                  string          `json:"account"`
	AccrueTime               scontrol.Number `json:"accrue_time"` // Integer on LUMI, scontrol.Number on Mahti
	AdminComment             string          `json:"admin_comment"`
	AllocatingNode           string          `json:"allocating_node"`
	ArrayJobID               scontrol.Number `json:"array_job_id"`
	ArrayTaskID              scontrol.Number `json:"array_task_id"`
	ArrayMaxTasks            scontrol.Number `json:"array_max_tasks"`
	ArrayTaskString          string          `json:"array_task_string"`
	AssociationID            int             `json:"association_id"`
	BatchFeatures            string          `json:"batch_features"`
	BatchFlag                bool            `json:"batch_flag"`
	BatchHost                string          `json:"batch_host"`
	Flags                    []string        `json:"flags"`
	BurstBuffer              string          `json:"burst_buffer"`
	BurstBufferState         string          `json:"burst_buffer_state"`
	Cluster                  string          `json:"cluster"`
	ClusterFeatures          string          `json:"cluster_features"`
	Command                  string          `json:"command"`
	Comment                  string          `json:"comment"`
	Container                string          `json:"container"`
	ContainerID              string          `json:"container_id"`
	Contiguous               bool            `json:"contiguous"`
	CoreSpec                 int             `json:"core_spec"`
	ThreadSpec               int             `json:"thread_spec"`
	CoresPerSocket           scontrol.Number `json:"cores_per_socket"`
	BillableTres             scontrol.Number `json:"billable_tres"`
	CpusPerTask              scontrol.Number `json:"cpus_per_task"`
	CPUFrequencyMinimum      scontrol.Number `json:"cpu_frequency_minimum"`
	CPUFrequencyMaximum      scontrol.Number `json:"cpu_frequency_maximum"`
	CPUFrequencyGovernor     scontrol.Number `json:"cpu_frequency_governor"`
	CpusPerTres              string          `json:"cpus_per_tres"`
	Cron                     string          `json:"cron"`
	Deadline                 scontrol.Number `json:"deadline"` // Integer on LUMI, scontrol.Number on Mahti
	DelayBoot                scontrol.Number `json:"delay_boot"`
	Dependency               string          `json:"dependency"`
	DerivedExitCode          scontrol.Number `json:"derived_exit_code"`
	EligibleTime             scontrol.Number `json:"eligible_time"` // Integer on LUMI, scontrol.Number on Mahti
	EndTime                  scontrol.Number `json:"end_time"`      // Integer on LUMI, scontrol.Number on Mahti
	ExcludedNodes            string          `json:"excluded_nodes"`
	ExitCode                 scontrol.Number `json:"exit_code"`
	Extra                    string          `json:"extra"`
	FailedNode               string          `json:"failed_node"`
	Features                 string          `json:"features"`
	FederationOrigin         string          `json:"federation_origin"`
	FederationSiblingsActive string          `json:"federation_siblings_active"`
	FederationSiblingsViable string          `json:"federation_siblings_viable"`
	GresDetail               []string        `json:"gres_detail"`
	GroupID                  int             `json:"group_id"`
	GroupName                string          `json:"group_name"`
	HetJobID                 scontrol.Number `json:"het_job_id"`
	HetJobIDSet              string          `json:"het_job_id_set"`
	HetJobOffset             scontrol.Number `json:"het_job_offset"`
	JobID                    int             `json:"job_id"`
	JobResources             JobResources    `json:"job_resources,omitempty"`
	JobSizeStr               []string        `json:"job_size_str"`
	JobState                 JobState        `json:"job_state"`
	LastSchedEvaluation      scontrol.Number `json:"last_sched_evaluation"` // Integer on LUMI, scontrol.Number on Mahti
	Licenses                 string          `json:"licenses"`
	MailType                 []string        `json:"mail_type"`
	MailUser                 string          `json:"mail_user"`
	MaxCpus                  scontrol.Number `json:"max_cpus"`
	MaxNodes                 scontrol.Number `json:"max_nodes"`
	McsLabel                 string          `json:"mcs_label"`
	MemoryPerTres            string          `json:"memory_per_tres"`
	Name                     string          `json:"name"`
	Network                  string          `json:"network"`
	Nodes                    string          `json:"nodes"`
	Nice                     int             `json:"nice"`
	TasksPerCore             scontrol.Number `json:"tasks_per_core"`
	TasksPerTres             scontrol.Number `json:"tasks_per_tres"`
	TasksPerNode             scontrol.Number `json:"tasks_per_node"`
	TasksPerSocket           scontrol.Number `json:"tasks_per_socket"`
	TasksPerBoard            scontrol.Number `json:"tasks_per_board"`
	Cpus                     scontrol.Number `json:"cpus"`
	NodeCount                scontrol.Number `json:"node_count"`
	Tasks                    scontrol.Number `json:"tasks"`
	Partition                string          `json:"partition"`
	Prefer                   string          `json:"prefer"`
	MemoryPerCPU             scontrol.Number `json:"memory_per_cpu"`
	MemoryPerNode            scontrol.Number `json:"memory_per_node"`
	MinimumCpusPerNode       scontrol.Number `json:"minimum_cpus_per_node"`
	MinimumTmpDiskPerNode    scontrol.Number `json:"minimum_tmp_disk_per_node"`
	Power                    Power           `json:"power"`
	PreemptTime              scontrol.Number `json:"preempt_time"`     // Integer on LUMI, scontrol.Number on Mahti
	PreemptableTime          scontrol.Number `json:"preemptable_time"` // Integer on LUMI, scontrol.Number on Mahti
	PreSusTime               scontrol.Number `json:"pre_sus_time"`     // Integer on LUMI, scontrol.Number on Mahti
	Hold                     bool            `json:"hold"`
	Priority                 scontrol.Number `json:"priority"`
	Profile                  []string        `json:"profile"`
	Qos                      string          `json:"qos"`
	Reboot                   bool            `json:"reboot"`
	RequiredNodes            string          `json:"required_nodes"`
	MinimumSwitches          int             `json:"minimum_switches"`
	Requeue                  bool            `json:"requeue"`
	ResizeTime               scontrol.Number `json:"resize_time"` // Integer on LUMI, scontrol.Number on Mahti
	RestartCnt               int             `json:"restart_cnt"`
	ResvName                 string          `json:"resv_name"`
	ScheduledNodes           string          `json:"scheduled_nodes"`
	SelinuxContext           string          `json:"selinux_context"`
	Shared                   []string        `json:"shared"`
	Exclusive                []string        `json:"exclusive"`
	Oversubscribe            bool            `json:"oversubscribe"`
	ShowFlags                []string        `json:"show_flags"`
	SocketsPerBoard          int             `json:"sockets_per_board"`
	SocketsPerNode           scontrol.Number `json:"sockets_per_node"`
	StartTime                scontrol.Number `json:"start_time"` // Integer on LUMI, scontrol.Number on Mahti
	StateDescription         string          `json:"state_description"`
	StateReason              string          `json:"state_reason"`
	StandardError            string          `json:"standard_error"`
	StandardInput            string          `json:"standard_input"`
	StandardOutput           string          `json:"standard_output"`
	SubmitTime               scontrol.Number `json:"submit_time"`  // Integer on LUMI, scontrol.Number on Mahti
	SuspendTime              scontrol.Number `json:"suspend_time"` // Integer on LUMI, scontrol.Number on Mahti
	SystemComment            string          `json:"system_comment"`
	TimeLimit                scontrol.Number `json:"time_limit"`
	TimeMinimum              scontrol.Number `json:"time_minimum"`
	ThreadsPerCore           scontrol.Number `json:"threads_per_core"`
	TresBind                 string          `json:"tres_bind"`
	TresFreq                 string          `json:"tres_freq"`
	TresPerJob               string          `json:"tres_per_job"`
	TresPerNode              string          `json:"tres_per_node"`
	TresPerSocket            string          `json:"tres_per_socket"`
	TresPerTask              string          `json:"tres_per_task"`
	TresReqStr               string          `json:"tres_req_str"`
	TresAllocStr             string          `json:"tres_alloc_str"`
	UserID                   int             `json:"user_id"`
	UserName                 string          `json:"user_name"`
	MaximumSwitchWaitTime    int             `json:"maximum_switch_wait_time"`
	Wckey                    string          `json:"wckey"`
	CurrentWorkingDirectory  string          `json:"current_working_directory"`
}
