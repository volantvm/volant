//go:build linux

package dataplane

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

type manager struct {
	logger          *slog.Logger
	ingressProgram  *ebpf.Program
	egressProgram   *ebpf.Program
	portmap         *ebpf.Map
	conntrack       *ebpf.Map
	stats           *ebpf.Map
	backends        *ebpf.Map
	backendConfig   *ebpf.Map
	ingressLink     link.Link
	egressLink      link.Link
	iface           string
	mu              sync.Mutex
	closed          bool
	programs        *ebpf.Collection
}

func newManager(opts Options) (Interface, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.ObjectPath == "" {
		return nil, fmt.Errorf("dataplane: bpf object path required")
	}
	if opts.Interface == "" {
		return nil, fmt.Errorf("dataplane: interface name required")
	}

	spec, err := ebpf.LoadCollectionSpec(opts.ObjectPath)
	if err != nil {
		return nil, fmt.Errorf("dataplane: load collection spec: %w", err)
	}

	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return nil, fmt.Errorf("dataplane: create collection: %w", err)
	}

	// Load ingress program
	ingressProg, ok := coll.Programs["drift_l4_ingress"]
	if !ok {
		coll.Close()
		return nil, errors.New("dataplane: program drift_l4_ingress not found")
	}

	// Load egress program
	egressProg, ok := coll.Programs["drift_l4_egress"]
	if !ok {
		coll.Close()
		return nil, errors.New("dataplane: program drift_l4_egress not found")
	}

	// Load maps
	portmap, ok := coll.Maps["portmap"]
	if !ok {
		coll.Close()
		return nil, errors.New("dataplane: portmap not found")
	}

	conntrack, ok := coll.Maps["conntrack"]
	if !ok {
		coll.Close()
		return nil, errors.New("dataplane: conntrack not found")
	}

	stats, ok := coll.Maps["stats"]
	if !ok {
		coll.Close()
		return nil, errors.New("dataplane: stats not found")
	}

	backends, ok := coll.Maps["backends"]
	if !ok {
		coll.Close()
		return nil, errors.New("dataplane: backends not found")
	}

	backendConfig, ok := coll.Maps["backend_config"]
	if !ok {
		coll.Close()
		return nil, errors.New("dataplane: backend_config not found")
	}

	iface, err := net.InterfaceByName(opts.Interface)
	if err != nil {
		coll.Close()
		return nil, fmt.Errorf("dataplane: lookup interface %s: %w", opts.Interface, err)
	}

	// Attach ingress
	ingressLink, err := link.AttachTCX(link.TCXOptions{
		Program:   ingressProg,
		Interface: iface.Index,
		Attach:    ebpf.AttachTCXIngress,
	})
	if err != nil {
		coll.Close()
		return nil, fmt.Errorf("dataplane: attach ingress tcx: %w", err)
	}

	// Attach egress
	egressLink, err := link.AttachTCX(link.TCXOptions{
		Program:   egressProg,
		Interface: iface.Index,
		Attach:    ebpf.AttachTCXEgress,
	})
	if err != nil {
		ingressLink.Close()
		coll.Close()
		return nil, fmt.Errorf("dataplane: attach egress tcx: %w", err)
	}

	opts.Logger.Info("drift dataplane attached", "interface", opts.Interface, "ingress", true, "egress", true)

	return &manager{
		logger:         opts.Logger.With("component", "dataplane"),
		ingressProgram: ingressProg,
		egressProgram:  egressProg,
		portmap:        portmap,
		conntrack:      conntrack,
		stats:          stats,
		backends:       backends,
		backendConfig:  backendConfig,
		ingressLink:    ingressLink,
		egressLink:     egressLink,
		iface:          opts.Interface,
		programs:       coll,
	}, nil
}

func (m *manager) ApplyBridge(_ context.Context, proto uint8, hostPort uint16, destIP net.IP, destPort uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("dataplane: manager closed")
	}

	ip4 := destIP.To4()
	if ip4 == nil {
		return fmt.Errorf("dataplane: destination ip %s not ipv4", destIP)
	}

	key := portmapKey{
		Proto: proto,
		Port:  htons(hostPort),
	}

	value := portmapValue{
		DestIP:   binary.BigEndian.Uint32(ip4),
		DestPort: htons(destPort),
	}

	if err := m.portmap.Put(&key, &value); err != nil {
		return fmt.Errorf("dataplane: portmap update: %w", err)
	}

	m.logger.Info("route applied", "proto", protoName(proto), "host_port", hostPort, "dest_ip", destIP.String(), "dest_port", destPort)
	return nil
}

func (m *manager) ConfigureLoadBalancer(_ context.Context, config LoadBalancerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("dataplane: manager closed")
	}

	if len(config.Backends) == 0 {
		return fmt.Errorf("dataplane: at least one backend required")
	}
	if len(config.Backends) > 16 {
		return fmt.Errorf("dataplane: maximum 16 backends supported")
	}

	// Remove from simple portmap if exists (migrate to LB)
	pmKey := portmapKey{
		Proto: config.Proto,
		Port:  htons(config.HostPort),
	}
	_ = m.portmap.Delete(&pmKey)

	// Configure backend config
	bcKey := backendConfigKey{
		Proto: config.Proto,
		Port:  htons(config.HostPort),
	}
	bcValue := backendConfigValue{
		BackendCount: uint32(len(config.Backends)),
		LBAlgorithm:  uint32(config.Algorithm),
		NextBackend:  0,
	}
	if err := m.backendConfig.Put(&bcKey, &bcValue); err != nil {
		return fmt.Errorf("dataplane: backend_config update: %w", err)
	}

	// Configure backends
	for idx, backend := range config.Backends {
		ip4 := backend.IP.To4()
		if ip4 == nil {
			return fmt.Errorf("dataplane: backend %d ip %s not ipv4", idx, backend.IP)
		}

		bKey := backendKey{
			Proto:      config.Proto,
			Port:       htons(config.HostPort),
			BackendIdx: uint32(idx),
		}
		bValue := backendValue{
			DestIP:       binary.BigEndian.Uint32(ip4),
			DestPort:     htons(backend.Port),
			HealthStatus: boolToUint8(backend.Healthy),
			ConnCount:    0,
		}
		if err := m.backends.Put(&bKey, &bValue); err != nil {
			return fmt.Errorf("dataplane: backend %d update: %w", idx, err)
		}
	}

	m.logger.Info("load balancer configured",
		"proto", protoName(config.Proto),
		"host_port", config.HostPort,
		"algorithm", config.Algorithm,
		"backends", len(config.Backends))
	return nil
}

func (m *manager) SetBackendHealth(_ context.Context, proto uint8, hostPort uint16, backendIdx uint32, healthy bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("dataplane: manager closed")
	}

	bKey := backendKey{
		Proto:      proto,
		Port:       htons(hostPort),
		BackendIdx: backendIdx,
	}

	var bValue backendValue
	if err := m.backends.Lookup(&bKey, &bValue); err != nil {
		return fmt.Errorf("dataplane: backend lookup: %w", err)
	}

	bValue.HealthStatus = boolToUint8(healthy)
	if err := m.backends.Put(&bKey, &bValue); err != nil {
		return fmt.Errorf("dataplane: backend health update: %w", err)
	}

	m.logger.Info("backend health updated",
		"proto", protoName(proto),
		"host_port", hostPort,
		"backend_idx", backendIdx,
		"healthy", healthy)
	return nil
}

func (m *manager) Remove(_ context.Context, proto uint8, hostPort uint16) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("dataplane: manager closed")
	}

	// Remove from portmap (single backend)
	pmKey := portmapKey{
		Proto: proto,
		Port:  htons(hostPort),
	}
	_ = m.portmap.Delete(&pmKey)

	// Remove from backend_config (load balancer)
	bcKey := backendConfigKey{
		Proto: proto,
		Port:  htons(hostPort),
	}
	var bcValue backendConfigValue
	if err := m.backendConfig.Lookup(&bcKey, &bcValue); err == nil {
		// Remove all backends
		for i := uint32(0); i < bcValue.BackendCount; i++ {
			bKey := backendKey{
				Proto:      proto,
				Port:       htons(hostPort),
				BackendIdx: i,
			}
			_ = m.backends.Delete(&bKey)
		}
		_ = m.backendConfig.Delete(&bcKey)
	}

	m.logger.Info("route removed", "proto", protoName(proto), "host_port", hostPort)
	return nil
}

func (m *manager) Stats(ctx context.Context) (Stats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return Stats{}, errors.New("dataplane: manager closed")
	}

	var result Stats
	var value uint64

	// Read per-CPU stats and sum them up
	for idx := uint32(0); idx < 4; idx++ {
		if err := m.stats.Lookup(&idx, &value); err != nil {
			return Stats{}, fmt.Errorf("dataplane: stats lookup: %w", err)
		}
		switch idx {
		case 0: // STAT_INGRESS_PACKETS
			result.IngressPackets = value
		case 1: // STAT_INGRESS_BYTES
			result.IngressBytes = value
		case 2: // STAT_EGRESS_PACKETS
			result.EgressPackets = value
		case 3: // STAT_EGRESS_BYTES
			result.EgressBytes = value
		}
	}

	return result, nil
}

func (m *manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	if m.ingressLink != nil {
		_ = m.ingressLink.Close()
	}
	if m.egressLink != nil {
		_ = m.egressLink.Close()
	}
	if m.programs != nil {
		m.programs.Close()
	}
	m.logger.Info("drift dataplane closed")
	return nil
}

type portmapKey struct {
	Proto uint8
	_     uint8
	Port  uint16
}

type portmapValue struct {
	DestIP   uint32
	DestPort uint16
	_        uint16
}

type backendKey struct {
	Proto      uint8
	_          uint8
	Port       uint16
	BackendIdx uint32
}

type backendValue struct {
	DestIP       uint32
	DestPort     uint16
	HealthStatus uint8
	_            uint8
	ConnCount    uint64
}

type backendConfigKey struct {
	Proto uint8
	_     uint8
	Port  uint16
}

type backendConfigValue struct {
	BackendCount uint32
	LBAlgorithm  uint32
	NextBackend  uint32
}

func htons(value uint16) uint16 {
	return value<<8 | value>>8
}

func boolToUint8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func protoName(proto uint8) string {
	switch proto {
	case 6:
		return "tcp"
	case 17:
		return "udp"
	default:
		return fmt.Sprintf("%d", proto)
	}
}
