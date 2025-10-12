// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package db

import internaldb "github.com/volantvm/volant/internal/server/db"

type (
	Store                    = internaldb.Store
	Queries                  = internaldb.Queries
	VM                       = internaldb.VM
	VMStatus                 = internaldb.VMStatus
	VMGroup                  = internaldb.VMGroup
	Plugin                   = internaldb.Plugin
	PluginArtifact           = internaldb.PluginArtifact
	VMCloudInit              = internaldb.VMCloudInit
	VMConfig                 = internaldb.VMConfig
	VMConfigHistoryEntry     = internaldb.VMConfigHistoryEntry
	PluginRepository         = internaldb.PluginRepository
	VMRepository             = internaldb.VMRepository
	VMConfigRepository       = internaldb.VMConfigRepository
	VMGroupRepository        = internaldb.VMGroupRepository
	PluginArtifactRepository = internaldb.PluginArtifactRepository
	VMCloudInitRepository    = internaldb.VMCloudInitRepository
	IPRepository             = internaldb.IPRepository
	IPAllocation             = internaldb.IPAllocation
	IPStatus                 = internaldb.IPStatus
)

const (
	VMStatusPending   = internaldb.VMStatusPending
	VMStatusStarting  = internaldb.VMStatusStarting
	VMStatusRunning   = internaldb.VMStatusRunning
	VMStatusStopped   = internaldb.VMStatusStopped
	VMStatusCrashed   = internaldb.VMStatusCrashed
	IPStatusAvailable = internaldb.IPStatusAvailable
	IPStatusLeased    = internaldb.IPStatusLeased
)

var ErrNoAvailableIPs = internaldb.ErrNoAvailableIPs
