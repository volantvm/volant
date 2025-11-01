//go:build !linux

package dataplane

import (
	"context"
	"net"
)

func newManager(Options) (Interface, error) {
	return nil, ErrUnsupported
}

type noopManager struct{}

func (noopManager) ApplyBridge(context.Context, uint8, uint16, net.IP, uint16) error { return nil }
func (noopManager) ConfigureLoadBalancer(context.Context, LoadBalancerConfig) error  { return nil }
func (noopManager) SetBackendHealth(context.Context, uint8, uint16, uint32, bool) error {
	return nil
}
func (noopManager) Remove(context.Context, uint8, uint16) error { return nil }
func (noopManager) Stats(context.Context) (Stats, error)       { return Stats{}, nil }
func (noopManager) Close() error                               { return nil }
