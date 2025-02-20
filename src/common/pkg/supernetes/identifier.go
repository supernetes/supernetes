// SPDX-License-Identifier: MPL-2.0
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package supernetes

const NamespaceWorkload = "supernetes-workload" // TODO: This should be configurable

const ScopeNode = "supernetes-node"
const TaintNoSchedule = ScopeNode + "/no-schedule"

const ScopeWorkload = "supernetes-workload"
const LabelWorkloadKind = ScopeWorkload + "/kind"
const LabelWorkloadIdentifier = ScopeWorkload + "/idenfitier"
const LabelAdditionalNodes = ScopeWorkload + "/additional-nodes"
const SGWorkloadUnallocated = ScopeWorkload + "/unallocated"

const ScopeExtra = "supernetes-extra"
const ScopeOption = "supernetes-option"

const ScopeController = "supernetes-controller"
const Group = "supernetes" // TODO: Proper FQDN group for Supernetes

const ContainerPlaceholder = "workload"
const ImagePlaceholder = "none"

const NodeTypeVirtualKubelet = "virtual-kubelet"
const NodeRoleSupernetes = "supernetes"

const CertSANSupernetes = "supernetes.internal"
const CertFormatSupernetesVK = "supernetes-vk-%s-%d"
