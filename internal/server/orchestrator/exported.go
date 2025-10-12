// Copyright (c) 2025 HYPR. PTE. LTD.
//
// Business Source License 1.1
// See LICENSE file in the project root for details.

package orchestrator

import "net"

// BuildKernelCmdline exposes the internal kernel cmdline builder for external consumers.
func BuildKernelCmdline(ip, gateway, netmask, hostname, extra string) string {
	return buildKernelCmdline(ip, gateway, netmask, hostname, extra)
}

// AppendKernelArgs appends additional key=value parameters to a kernel cmdline.
func AppendKernelArgs(cmdline string, args map[string]string) string {
	return appendKernelArgs(cmdline, args)
}

// FormatNetmask converts an IP mask to dotted quad format.
func FormatNetmask(mask net.IPMask) string {
	return formatNetmask(mask)
}

// SanitizeHostname normalizes a VM hostname to match kernel cmdline expectations.
func SanitizeHostname(name string) string {
	return sanitizeHostname(name)
}

// DeriveMAC deterministically generates a MAC address for a VM based on its name and IP.
func DeriveMAC(name, ip string) string {
	return deriveMAC(name, ip)
}
