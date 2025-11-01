package dataplane

import (
	"context"
	"log/slog"
	"net"
)

// Options configures dataplane manager construction.
type Options struct {
	ObjectPath string
	Interface  string
	Logger     *slog.Logger
}

// Stats holds dataplane packet/byte counters.
type Stats struct {
	IngressPackets uint64
	IngressBytes   uint64
	EgressPackets  uint64
	EgressBytes    uint64
}

// LoadBalancerAlgorithm specifies the backend selection algorithm.
type LoadBalancerAlgorithm uint32

const (
	RoundRobin LoadBalancerAlgorithm = 0
	LeastConn  LoadBalancerAlgorithm = 1
	IPHash     LoadBalancerAlgorithm = 2
)

// Backend represents a single backend server.
type Backend struct {
	IP      net.IP
	Port    uint16
	Healthy bool
}

// LoadBalancerConfig configures a load-balanced service.
type LoadBalancerConfig struct {
	Proto     uint8
	HostPort  uint16
	Algorithm LoadBalancerAlgorithm
	Backends  []Backend
}

// Interface describes bridge dataplane interactions.
type Interface interface {
	ApplyBridge(ctx context.Context, proto uint8, hostPort uint16, destIP net.IP, destPort uint16) error
	ConfigureLoadBalancer(ctx context.Context, config LoadBalancerConfig) error
	SetBackendHealth(ctx context.Context, proto uint8, hostPort uint16, backendIdx uint32, healthy bool) error
	Remove(ctx context.Context, proto uint8, hostPort uint16) error
	Stats(ctx context.Context) (Stats, error)
	Close() error
}

// ErrUnsupported indicates the dataplane is not available on the current platform.
var ErrUnsupported = errorUnsupported{}

type errorUnsupported struct{}

func (errorUnsupported) Error() string { return "dataplane: unsupported platform" }

// New constructs a platform-appropriate dataplane manager.
func New(opts Options) (Interface, error) {
	return newManager(opts)
}
