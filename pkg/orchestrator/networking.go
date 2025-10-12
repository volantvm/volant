// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package orchestrator

import (
	"net"

	internalorchestrator "github.com/volantvm/volant/internal/server/orchestrator"
)

// BuildKernelCmdline replicates the orchestrator's static networking kernel args builder.
func BuildKernelCmdline(ip, gateway, netmask, hostname, extra string) string {
	return internalorchestrator.BuildKernelCmdline(ip, gateway, netmask, hostname, extra)
}

// AppendKernelArgs merges additional key=value pairs onto a kernel cmdline.
func AppendKernelArgs(cmdline string, args map[string]string) string {
	return internalorchestrator.AppendKernelArgs(cmdline, args)
}

// FormatNetmask converts an IP mask into dotted quad representation.
func FormatNetmask(mask net.IPMask) string {
	return internalorchestrator.FormatNetmask(mask)
}

// SanitizeHostname coerces hostnames to the orchestrator-safe form.
func SanitizeHostname(name string) string {
	return internalorchestrator.SanitizeHostname(name)
}

// DeriveMAC generates a deterministic MAC address for a VM.
func DeriveMAC(name, ip string) string {
	return internalorchestrator.DeriveMAC(name, ip)
}
