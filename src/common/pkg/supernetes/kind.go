// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package supernetes

const (
	KindUntracked Kind = "untracked"
	KindTracked   Kind = "tracked"
)

// Kind represents the two types of workloads Supernetes distinguishes: tracked and untracked. In short, tracked
// workloads are created through the Kubernetes interface (user creates a Pod), while untracked workloads are populated
// through the agent (user deploys, e.g., a Slurm job). Tracked workloads must adhere to stricter Kubernetes standards,
// which includes being deployed through a container image. Untracked workloads can represent anything that is gathered
// from the agent environment (including jobs from other users), but have limited utility in the Kubernetes environment.
type Kind string
