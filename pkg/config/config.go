// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package config

import internalconfig "github.com/volantvm/volant/internal/server/config"

// ServerConfig re-exports the daemon server configuration for shared consumers.
type ServerConfig = internalconfig.ServerConfig

// FromEnv loads server configuration using Volant's internal defaults.
func FromEnv() (ServerConfig, error) {
	return internalconfig.FromEnv()
}
